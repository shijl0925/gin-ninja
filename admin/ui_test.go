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
		PrototypePath: "/console/prototype",
	})

	for _, path := range []string{"/console", "/console/login", "/console/prototype"} {
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
			`const prototypePagePath = "/console/prototype";`,
			`await request("/custom/api/auth/login", {`,
			`Paste a token from /custom/api/auth/login`,
		} {
			if !strings.Contains(body, snippet) {
				t.Fatalf("GET %s missing %q", path, snippet)
			}
		}
	}
}

func TestMountUIDeduplicatesPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath:     "/admin",
		LoginPath:     "/admin",
		PrototypePath: "/admin",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d, body=%s", w.Code, w.Body.String())
	}
}
