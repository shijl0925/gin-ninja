package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSession_DeletePersistsRemovalAndKeys(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(middleware.SessionMiddleware(&middleware.SessionConfig{Secret: "test-secret"}))
	r.POST("/mutate", func(c *gin.Context) {
		s := middleware.GetSession(c)
		s.Set("keep", "1")
		s.Set("drop", "2")
		s.Delete("drop")
		keys := s.Keys()
		sort.Strings(keys)
		c.JSON(http.StatusOK, keys)
	})
	r.GET("/check", func(c *gin.Context) {
		s := middleware.GetSession(c)
		keep, keepOK := s.Get("keep")
		_, dropOK := s.Get("drop")
		c.JSON(http.StatusOK, gin.H{
			"keep":    keep,
			"keep_ok": keepOK,
			"drop_ok": dropOK,
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/mutate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); body != "[\"keep\"]" {
		t.Fatalf("expected only keep key, got %s", body)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	req = httptest.NewRequest(http.MethodGet, "/check", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `"keep":"1"`) || !strings.Contains(w.Body.String(), `"keep_ok":true`) || !strings.Contains(w.Body.String(), `"drop_ok":false`) {
		t.Fatalf("unexpected persisted session state: %s", w.Body.String())
	}
}

func TestSession_ClearPersistsEmptySession(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(middleware.SessionMiddleware(&middleware.SessionConfig{Secret: "test-secret"}))
	r.POST("/set", func(c *gin.Context) {
		s := middleware.GetSession(c)
		s.Set("user_id", "42")
		c.Status(http.StatusNoContent)
	})
	r.POST("/clear", func(c *gin.Context) {
		s := middleware.GetSession(c)
		s.Clear()
		c.Status(http.StatusNoContent)
	})
	r.GET("/count", func(c *gin.Context) {
		s := middleware.GetSession(c)
		c.String(http.StatusOK, string(rune(len(s.Keys())+'0')))
	})

	req := httptest.NewRequest(http.MethodPost, "/set", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie after set")
	}

	req = httptest.NewRequest(http.MethodPost, "/clear", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var clearedCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			clearedCookie = cookie
			break
		}
	}
	if clearedCookie == nil {
		t.Fatal("expected session cookie after clear")
	}

	req = httptest.NewRequest(http.MethodGet, "/count", nil)
	req.AddCookie(clearedCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != "0" {
		t.Fatalf("expected empty session after clear, got %q", got)
	}
}

func TestNewSessionID_UniqueAndURLSafe(t *testing.T) {
	t.Parallel()

	first := middleware.NewSessionID()
	second := middleware.NewSessionID()

	if first == "" || second == "" {
		t.Fatal("expected non-empty session IDs")
	}
	if first == second {
		t.Fatal("expected generated session IDs to be unique")
	}
	if len(first) != 43 || len(second) != 43 {
		t.Fatalf("expected base64url encoded 32-byte IDs, got lengths %d and %d", len(first), len(second))
	}
	if strings.ContainsAny(first+second, "+/=") {
		t.Fatalf("expected URL-safe session IDs, got %q and %q", first, second)
	}
}

func TestRecovery_LogsAndWritesInternalError(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(logger))
	r.GET("/", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Message != "panic recovered" {
		t.Fatalf("unexpected log message %q", entry.Message)
	}
	fields := entry.ContextMap()
	if fields["request_id"] != "req-123" {
		t.Fatalf("expected request_id field, got %+v", fields)
	}
}

func TestLogger_LogsStatusQueryRequestIDAndPrivateErrors(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(logger))
	r.GET("/", func(c *gin.Context) {
		_ = c.Error(errors.New("boom")).SetType(gin.ErrorTypePrivate)
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	req.Header.Set("X-Request-ID", "req-456")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Level != zap.ErrorLevel {
		t.Fatalf("expected error level, got %s", entry.Level)
	}
	fields := entry.ContextMap()
	if fields["status"] != int64(http.StatusInternalServerError) {
		t.Fatalf("expected status field, got %+v", fields)
	}
	if fields["method"] != http.MethodGet || fields["path"] != "/" || fields["query"] != "foo=bar" {
		t.Fatalf("unexpected request fields: %+v", fields)
	}
	if fields["request_id"] != "req-456" {
		t.Fatalf("expected request_id field, got %+v", fields)
	}
	if !strings.Contains(fields["error"].(string), "boom") {
		t.Fatalf("expected private error message, got %+v", fields)
	}
}
