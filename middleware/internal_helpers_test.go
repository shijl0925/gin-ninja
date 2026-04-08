package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
)

func TestLocaleKeyAndClaimsGetUserID(t *testing.T) {
	t.Parallel()

	if got := LocaleKey(); got != contextkeys.Locale {
		t.Fatalf("LocaleKey() = %q, want %q", got, contextkeys.Locale)
	}
	if got := (&Claims{UserID: 42}).GetUserID(); got != 42 {
		t.Fatalf("Claims.GetUserID() = %d, want 42", got)
	}
}

func TestSessionResponseWriterPersistsOnWriteHeaderNowAndWriteString(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(SessionMiddleware(&SessionConfig{Secret: "test-secret"}))
	r.GET("/header-now", func(c *gin.Context) {
		GetSession(c).Set("mode", "header")
		c.Writer.WriteHeaderNow()
	})
	r.GET("/write-string", func(c *gin.Context) {
		GetSession(c).Set("mode", "string")
		_, _ = c.Writer.WriteString("ok")
	})
	r.GET("/read", func(c *gin.Context) {
		value, ok := GetSession(c).Get("mode")
		if !ok {
			c.String(http.StatusNotFound, "")
			return
		}
		c.String(http.StatusOK, value)
	})

	t.Run("write header now", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/header-now", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		cookie := sessionCookieFromResponse(t, w)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/read", nil)
		req.AddCookie(cookie)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Body.String() != "header" {
			t.Fatalf("expected persisted header-now session, got %q", w.Body.String())
		}
	})

	t.Run("write string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/write-string", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		cookie := sessionCookieFromResponse(t, w)
		if body := w.Body.String(); body != "ok" {
			t.Fatalf("expected body ok, got %q", body)
		}

		req = httptest.NewRequest(http.MethodGet, "/read", nil)
		req.AddCookie(cookie)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Body.String() != "string" {
			t.Fatalf("expected persisted write-string session, got %q", w.Body.String())
		}
	})
}

func sessionCookieFromResponse(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatal("expected session cookie")
	return nil
}
