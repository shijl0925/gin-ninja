package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	// Write an empty config so Load can find the file.
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{}"), 0o644) //nolint:errcheck

	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.JWT.ExpireHours != 24 {
		t.Errorf("expected default expire_hours 24, got %d", cfg.JWT.ExpireHours)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Log.Level)
	}
}

func TestLoad_FromFile(t *testing.T) {
	yaml := `
app:
  name: "Test API"
  version: "2.0.0"
server:
  port: 9090
jwt:
  secret: "super-secret"
  expire_hours: 48
`
	path := writeTempConfig(t, yaml)
	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.App.Name != "Test API" {
		t.Errorf("expected name 'Test API', got %q", cfg.App.Name)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.JWT.Secret != "super-secret" {
		t.Errorf("expected secret 'super-secret', got %q", cfg.JWT.Secret)
	}
	if cfg.JWT.ExpireHours != 48 {
		t.Errorf("expected expire_hours 48, got %d", cfg.JWT.ExpireHours)
	}
}

func TestServerConfig_Addr(t *testing.T) {
	s := settings.ServerConfig{Host: "127.0.0.1", Port: 3000}
	if s.Addr() != "127.0.0.1:3000" {
		t.Errorf("unexpected addr: %s", s.Addr())
	}
}

func TestServerConfig_Addr_Defaults(t *testing.T) {
	s := settings.ServerConfig{}
	if s.Addr() != "0.0.0.0:8080" {
		t.Errorf("unexpected addr: %s", s.Addr())
	}
}
