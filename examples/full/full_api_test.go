package main

import (
	"context"
	"errors"
	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullExampleBuildsRoutesAndEndpoints(t *testing.T) {
	client := newFullTestClient(t)

	for _, path := range []string{"/docs", "/docs/v1", "/docs/v0", "/openapi.json", "/openapi/v1.json", "/health"} {
		resp := client.Get(path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
	}

	register := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}

	login := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := login.DecodeJSON(&auth); err != nil {
		t.Fatalf("Decode login: %v", err)
	}
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	me := doFullJSON(t, client, http.MethodGet, "/api/v1/auth/me", nil, auth.Token)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("expected auth me 200, got %d", me.StatusCode)
	}
	var profile struct {
		UserID    uint   `json:"user_id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := me.DecodeJSON(&profile); err != nil {
		t.Fatalf("Decode auth me: %v", err)
	}
	if profile.UserID == 0 || profile.Name != "Alice" || profile.Email != "alice@example.com" || profile.ExpiresAt == "" {
		t.Fatalf("unexpected auth profile: %+v", profile)
	}

	list := doFullJSON(t, client, http.MethodGet, "/api/v1/users/?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}

	adminMeta := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/users/meta", nil, auth.Token)
	if adminMeta.StatusCode != http.StatusOK {
		t.Fatalf("expected admin metadata 200, got %d", adminMeta.StatusCode)
	}

	update := doFullJSON(t, client, http.MethodPut, "/api/v1/users/1", map[string]any{
		"name": "Alicia",
		"age":  19,
	}, auth.Token)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.StatusCode)
	}

	versioned := client.Get("/api/v0/examples/versioned/info")
	if versioned.StatusCode != http.StatusOK || versioned.Header.Get("Deprecation") == "" {
		t.Fatalf("expected deprecated version response, got status=%d headers=%v", versioned.StatusCode, versioned.Header)
	}
}

func TestFullExampleSmokeAuthDocsHealthAndVersioning(t *testing.T) {
	client := newFullTestClient(t)

	unauthorized := doFullJSON(t, client, http.MethodGet, "/api/v1/users/", nil, "")
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized users list to return 401, got %d", unauthorized.StatusCode)
	}

	unauthorizedAdmin := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources", nil, "")
	if unauthorizedAdmin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized admin resources to return 401, got %d", unauthorizedAdmin.StatusCode)
	}

	healthResp := client.Get("/health")
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", healthResp.StatusCode)
	}
	var health map[string]string
	if err := healthResp.DecodeJSON(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("unexpected health payload: %+v", health)
	}

	for _, tc := range []struct {
		path        string
		wantTitle   string
		wantSpecURL string
	}{
		{path: "/docs", wantTitle: "Full Example - API Docs", wantSpecURL: "/openapi.json"},
		{path: "/docs/v1", wantTitle: "Full Example (v1) - API Docs", wantSpecURL: "/openapi/v1.json"},
		{path: "/docs/v0", wantTitle: "Full Example (v0) - API Docs", wantSpecURL: "/openapi/v0.json"},
	} {
		resp := client.Get(tc.path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, resp.StatusCode)
		}
		html := resp.String()
		if !strings.Contains(html, tc.wantTitle) || !strings.Contains(html, tc.wantSpecURL) {
			t.Fatalf("%s: unexpected docs body %q", tc.path, html)
		}
	}

	register := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}

	login := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := login.DecodeJSON(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	list := doFullJSON(t, client, http.MethodGet, "/api/v1/users/?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}
	var page map[string]any
	if err := list.DecodeJSON(&page); err != nil {
		t.Fatalf("decode user list: %v", err)
	}
	if _, ok := page["items"]; !ok {
		t.Fatalf("expected paginated users payload, got %+v", page)
	}

	detail := doFullJSON(t, client, http.MethodGet, "/api/v1/users/1", nil, auth.Token)
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("expected get user 200, got %d", detail.StatusCode)
	}

	adminList := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/users?search=alice", nil, auth.Token)
	if adminList.StatusCode != http.StatusOK {
		t.Fatalf("expected admin users list 200, got %d", adminList.StatusCode)
	}

	v1 := client.Get("/api/v1/examples/versioned/info")
	if v1.StatusCode != http.StatusOK {
		t.Fatalf("expected v1 versioned info 200, got %d", v1.StatusCode)
	}
	if v1.Header.Get("Deprecation") != "" || v1.Header.Get("Sunset") != "" || v1.Header.Get("Link") != "" {
		t.Fatalf("did not expect deprecation headers on v1, got %v", v1.Header)
	}

	v0 := client.Get("/api/v0/examples/versioned/info")
	if v0.StatusCode != http.StatusOK {
		t.Fatalf("expected v0 versioned info 200, got %d", v0.StatusCode)
	}
	if v0.Header.Get("Deprecation") == "" || v0.Header.Get("Sunset") == "" || v0.Header.Get("Link") == "" {
		t.Fatalf("expected deprecation headers on v0, got %v", v0.Header)
	}

	for _, path := range []string{"/docs/v9", "/openapi/v9.json"} {
		resp := client.Get(path)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", path, resp.StatusCode)
		}
	}

	docsHeadResp := client.Request(http.MethodHead, "/docs/v1", nil)
	if docsHeadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HEAD /docs/v1 to follow current internal-route behavior and return 404, got %d", docsHeadResp.StatusCode)
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
	log := bootstrap.InitLogger(&cfg.Log)

	if err := run(cfg, log); err == nil {
		t.Fatal("expected run to fail for invalid address")
	}
}

func TestFullExampleInitDBAndMainHelpers(t *testing.T) {
	cfg := settings.Config{
		App:    settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:init-full?mode=memory&cache=shared",
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
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
