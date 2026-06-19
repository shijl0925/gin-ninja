package main

import (
	"testing"

	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	ninjatest "github.com/shijl0925/gin-ninja/testing"
)

func newFullTestClient(t *testing.T) *ninjatest.TestClient {
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
	log := bootstrap.InitLogger(&cfg.Log)
	db, err := initDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}

	return ninjatest.NewWithT(t, buildAPI(cfg, db, log))
}

func doFullJSON(t *testing.T, client *ninjatest.TestClient, method, path string, body any, token string) *ninjatest.Response {
	t.Helper()

	opts := []ninjatest.RequestOption{}
	if token != "" {
		opts = append(opts, ninjatest.Header("Authorization", "Bearer "+token))
	}
	return client.Request(method, path, body, opts...)
}

func readBody(t *testing.T, resp *ninjatest.Response) string {
	t.Helper()
	return resp.String()
}
