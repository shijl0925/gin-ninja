package settings_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestServerConfig_TimeoutDurations(t *testing.T) {
	s := settings.ServerConfig{}
	if s.ReadTimeoutDuration() != 60*time.Second {
		t.Fatalf("expected default read timeout, got %v", s.ReadTimeoutDuration())
	}
	if s.WriteTimeoutDuration() != 60*time.Second {
		t.Fatalf("expected default write timeout, got %v", s.WriteTimeoutDuration())
	}

	s = settings.ServerConfig{ReadTimeout: 5, WriteTimeout: 7}
	if s.ReadTimeoutDuration() != 5*time.Second {
		t.Fatalf("expected custom read timeout, got %v", s.ReadTimeoutDuration())
	}
	if s.WriteTimeoutDuration() != 7*time.Second {
		t.Fatalf("expected custom write timeout, got %v", s.WriteTimeoutDuration())
	}
}

func TestJWTConfig_ExpireDuration(t *testing.T) {
	if got := (settings.JWTConfig{}).ExpireDuration(); got != 24*time.Hour {
		t.Fatalf("expected default jwt ttl, got %v", got)
	}
	if got := (settings.JWTConfig{ExpireHours: 2}).ExpireDuration(); got != 2*time.Hour {
		t.Fatalf("expected custom jwt ttl, got %v", got)
	}
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	t.Setenv("SERVER__PORT", "7070")

	path := writeTempConfig(t, "{}")
	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 7070 {
		t.Fatalf("expected env override port 7070, got %d", cfg.Server.Port)
	}
}

func TestMustLoadPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	settings.MustLoad(filepath.Join(t.TempDir(), "missing.yaml"))
}
