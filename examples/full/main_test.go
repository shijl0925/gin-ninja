package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
)

func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := settings.Config{
		App: settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:" + t.Name() + "?mode=memory&cache=shared",
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	settings.Global.JWT = cfg.JWT

	log := bootstrap.InitLogger(&cfg.Log)
	db, err := initDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}

	return httptest.NewServer(buildAPI(cfg, db, log).Handler())
}

func doFullJSON(t *testing.T, server *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestFullExampleBuildsRoutesAndEndpoints(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	for _, path := range []string{"/docs", "/docs/v1", "/docs/v0", "/openapi.json", "/openapi/v1.json", "/health"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("Decode login: %v", err)
	}
	login.Body.Close()
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	list := doFullJSON(t, server, http.MethodGet, "/api/v1/users?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}
	list.Body.Close()

	update := doFullJSON(t, server, http.MethodPut, "/api/v1/users/1", map[string]any{
		"name": "Alicia",
		"age":  19,
	}, auth.Token)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.StatusCode)
	}
	update.Body.Close()

	versioned, err := http.Get(server.URL + "/api/v0/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned info: %v", err)
	}
	if versioned.StatusCode != http.StatusOK || versioned.Header.Get("Deprecation") == "" {
		t.Fatalf("expected deprecated version response, got status=%d headers=%v", versioned.StatusCode, versioned.Header)
	}
	versioned.Body.Close()
}

func TestFullExampleSmokeAuthDocsHealthAndVersioning(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	unauthorized := doFullJSON(t, server, http.MethodGet, "/api/v1/users", nil, "")
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized users list to return 401, got %d", unauthorized.StatusCode)
	}
	unauthorized.Body.Close()

	healthResp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", healthResp.StatusCode)
	}
	var health map[string]string
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("unexpected health payload: %+v", health)
	}

	for _, tc := range []struct {
		path       string
		wantTitle  string
		wantSpecURL string
	}{
		{path: "/docs", wantTitle: "Full Example - API Docs", wantSpecURL: "/openapi.json"},
		{path: "/docs/v1", wantTitle: "Full Example (v1) - API Docs", wantSpecURL: "/openapi/v1.json"},
		{path: "/docs/v0", wantTitle: "Full Example (v0) - API Docs", wantSpecURL: "/openapi/v0.json"},
	} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read %s body: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, resp.StatusCode)
		}
		html := string(body)
		if !strings.Contains(html, tc.wantTitle) || !strings.Contains(html, tc.wantSpecURL) {
			t.Fatalf("%s: unexpected docs body %q", tc.path, html)
		}
	}

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	list := doFullJSON(t, server, http.MethodGet, "/api/v1/users?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(list.Body).Decode(&page); err != nil {
		t.Fatalf("decode user list: %v", err)
	}
	list.Body.Close()
	if _, ok := page["items"]; !ok {
		t.Fatalf("expected paginated users payload, got %+v", page)
	}

	detail := doFullJSON(t, server, http.MethodGet, "/api/v1/users/1", nil, auth.Token)
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("expected get user 200, got %d", detail.StatusCode)
	}
	detail.Body.Close()

	v1, err := http.Get(server.URL + "/api/v1/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned v1: %v", err)
	}
	if v1.StatusCode != http.StatusOK {
		t.Fatalf("expected v1 versioned info 200, got %d", v1.StatusCode)
	}
	if v1.Header.Get("Deprecation") != "" || v1.Header.Get("Sunset") != "" || v1.Header.Get("Link") != "" {
		t.Fatalf("did not expect deprecation headers on v1, got %v", v1.Header)
	}
	v1.Body.Close()

	v0, err := http.Get(server.URL + "/api/v0/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned v0: %v", err)
	}
	if v0.StatusCode != http.StatusOK {
		t.Fatalf("expected v0 versioned info 200, got %d", v0.StatusCode)
	}
	if v0.Header.Get("Deprecation") == "" || v0.Header.Get("Sunset") == "" || v0.Header.Get("Link") == "" {
		t.Fatalf("expected deprecation headers on v0, got %v", v0.Header)
	}
	v0.Body.Close()
}

func TestFullExampleOpenAPIContracts(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	openAPIResp, err := http.Get(server.URL + "/openapi.json")
	if err != nil {
		t.Fatalf("GET /openapi.json: %v", err)
	}
	defer openAPIResp.Body.Close()
	if openAPIResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /openapi.json 200, got %d", openAPIResp.StatusCode)
	}

	var spec map[string]any
	if err := json.NewDecoder(openAPIResp.Body).Decode(&spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}

	components := spec["components"].(map[string]any)
	securitySchemes := components["securitySchemes"].(map[string]any)
	bearerAuth := securitySchemes["bearerAuth"].(map[string]any)
	if bearerAuth["type"] != "http" || bearerAuth["scheme"] != "bearer" || bearerAuth["bearerFormat"] != "JWT" {
		t.Fatalf("unexpected bearer auth scheme: %+v", bearerAuth)
	}

	paths := spec["paths"].(map[string]any)
	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/users/",
		"/api/v1/examples/request-meta",
		"/api/v0/examples/versioned/info",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected path %s in root spec, got keys=%v", path, paths)
		}
	}

	usersGet := paths["/api/v1/users/"].(map[string]any)["get"].(map[string]any)
	security := usersGet["security"].([]any)
	if len(security) != 1 {
		t.Fatalf("expected one security requirement, got %v", security)
	}
	if _, ok := security[0].(map[string]any)["bearerAuth"]; !ok {
		t.Fatalf("expected bearerAuth security requirement, got %v", security[0])
	}

	for _, tc := range []struct {
		path       string
		wantPath   string
		missingPath string
	}{
		{path: "/openapi/v1.json", wantPath: "/api/v1/users/", missingPath: "/api/v0/examples/versioned/info"},
		{path: "/openapi/v0.json", wantPath: "/api/v0/examples/versioned/info", missingPath: "/api/v1/users/"},
	} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, resp.StatusCode)
		}
		var versionedSpec map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&versionedSpec); err != nil {
			resp.Body.Close()
			t.Fatalf("decode %s: %v", tc.path, err)
		}
		resp.Body.Close()

		versionedPaths := versionedSpec["paths"].(map[string]any)
		if _, ok := versionedPaths[tc.wantPath]; !ok {
			t.Fatalf("%s: expected path %s, got %v", tc.path, tc.wantPath, versionedPaths)
		}
		if _, ok := versionedPaths[tc.missingPath]; ok {
			t.Fatalf("%s: did not expect path %s, got %v", tc.path, tc.missingPath, versionedPaths)
		}
	}
}

func TestFullExampleRunReturnsListenError(t *testing.T) {
	cfg := settings.Config{
		App: settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{
			Host: "127.0.0.1",
			Port: -1,
		},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:run-full?mode=memory&cache=shared",
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	settings.Global.JWT = cfg.JWT
	log := bootstrap.InitLogger(&cfg.Log)

	if err := run(cfg, log); err == nil {
		t.Fatal("expected run to fail for invalid address")
	}
}

func TestFullExampleInitDBAndMainHelpers(t *testing.T) {
	cfg := settings.Config{
		App: settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:init-full?mode=memory&cache=shared",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	log := bootstrap.InitLogger(&cfg.Log)
	db, err := initDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	api := buildAPI(cfg, db, log)
	if err := api.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if _, err := initDB(&settings.DatabaseConfig{Driver: "oracle"}); err == nil {
		t.Fatal("expected initDB to fail for unsupported driver")
	}

	originalRun := runFullMain
	originalFatal := fatalFull
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() {
		runFullMain = originalRun
		fatalFull = originalFatal
		_ = os.Chdir(wd)
	})

	called := false
	runFullMain = func(cfg settings.Config, log *zap.Logger) error {
		called = true
		return nil
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	main()
	if !called {
		t.Fatal("expected main to invoke injected runner")
	}

	runFullMain = func(cfg settings.Config, log *zap.Logger) error { return errors.New("boom") }
	fatalCalled := false
	fatalFull = func(v ...any) { fatalCalled = true }
	main()
	if !fatalCalled {
		t.Fatal("expected main to invoke injected fatal handler")
	}
}
