package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
