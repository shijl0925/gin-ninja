package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMountUIUsesConfiguredPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		Title:         `Admin <Console>`,
		APIBasePath:   "/custom/api/admin",
		AuthLoginPath: "/custom/api/auth/login",
		AdminPath:     "/console",
		LoginPath:     "/console/login",
	})

	for _, path := range []string{"/console", "/console/login"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body=%s", path, w.Code, w.Body.String())
		}
		body := w.Body.String()
		for _, snippet := range []string{
			"<title>Admin &lt;Console&gt;</title>",
			`const apiBase = "/custom/api/admin";`,
			`const adminPagePath = "/console";`,
			`const adminLoginPath = "/console/login";`,
			`const prototypePagePath = "/console";`,
			`await request("/custom/api/auth/login", {`,
			`Paste a token from /custom/api/auth/login`,
			// default extract expressions
			`const loginTokenExtractExpr = "payload.token";`,
			`const loginNameExtractExpr = "payload.name";`,
			`const loginUserIDExtractExpr = "payload.user_id || payload.userID";`,
		} {
			if !strings.Contains(body, snippet) {
				t.Fatalf("GET %s missing %q", path, snippet)
			}
		}
	}
}

func TestMountUICustomTokenExtract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath:           "/admin",
		LoginPath:           "/admin/login",
		TokenExtractExpr:    "payload.data && payload.data.accessToken",
		UserNameExtractExpr: "payload.data && payload.data.userName",
		UserIDExtractExpr:   "payload.data && payload.data.id",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d", w.Code)
	}
	body := w.Body.String()
	for _, snippet := range []string{
		`const loginTokenExtractExpr = "payload.data \u0026\u0026 payload.data.accessToken";`,
		`const loginNameExtractExpr = "payload.data \u0026\u0026 payload.data.userName";`,
		`const loginUserIDExtractExpr = "payload.data \u0026\u0026 payload.data.id";`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("GET /admin missing %q", snippet)
		}
	}
}

func TestMountUIDeduplicatesPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath: "/admin",
		LoginPath: "/admin",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestMountUIEscapesExtractorExpressionsAsData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath:        "/admin",
		LoginPath:        "/admin/login",
		TokenExtractExpr: `payload.token"; window.evil = true; "`,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, `const loginTokenExtractExpr = "payload.token"; window.evil = true; "";`) {
		t.Fatalf("extractor expression was injected as executable JavaScript: %s", body)
	}
	if !strings.Contains(body, `const loginTokenExtractExpr = "payload.token\"; window.evil = true; \"";`) {
		t.Fatal("expected extractor expression to be JSON string data")
	}
}

func TestServeDefaultUI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin", nil)

	ServeDefaultUI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if body := w.Body.String(); !strings.Contains(body, "Gin Ninja Admin") || !strings.Contains(body, `const apiBase = "/api/v1/admin";`) {
		t.Fatalf("default UI body missing expected content: %s", body)
	}
}
