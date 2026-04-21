package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
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

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func readBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
