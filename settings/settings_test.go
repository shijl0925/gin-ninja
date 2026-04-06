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

func TestLoad_DatabaseStructuredConfig(t *testing.T) {
	yaml := `
database:
  driver: "mysql"
  mysql:
    host: "127.0.0.1"
    user: "root"
    password: "p@ss:word+plus"
    name: "gin_ninja"
    charset: "utf8mb4"
    parse_time: true
    loc: "Local"
`
	path := writeTempConfig(t, yaml)
	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.Driver != "mysql" {
		t.Fatalf("expected mysql driver, got %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "" {
		t.Fatalf("expected default sqlite dsn to be cleared for mysql structured config, got %q", cfg.Database.DSN)
	}
	if cfg.Database.MySQL.Password != "p@ss:word+plus" || cfg.Database.MySQL.Host != "127.0.0.1" {
		t.Fatalf("unexpected mysql structured config: %+v", cfg.Database.MySQL)
	}
	if !cfg.Database.MySQL.ParseTime || cfg.Database.MySQL.Charset != "utf8mb4" {
		t.Fatalf("unexpected mysql defaults from file: %+v", cfg.Database.MySQL)
	}
}

func TestLoad_DatabaseStructuredPostgresClearsDefaultDSN(t *testing.T) {
	yaml := `
database:
  driver: "postgres"
  postgres:
    host: "127.0.0.1"
    user: "postgres"
    password: "postgres"
    name: "gin_ninja"
`
	path := writeTempConfig(t, yaml)
	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.DSN != "" {
		t.Fatalf("expected default sqlite dsn to be cleared for postgres structured config, got %q", cfg.Database.DSN)
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
	t.Setenv("DATABASE__MYSQL__PASSWORD", "env:p@ss+word")
	t.Setenv("DATABASE__POSTGRES__TIME_ZONE", "Asia/Shanghai")

	path := writeTempConfig(t, "{}")
	cfg, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 7070 {
		t.Fatalf("expected env override port 7070, got %d", cfg.Server.Port)
	}
	if cfg.Database.MySQL.Password != "env:p@ss+word" {
		t.Fatalf("expected env override mysql password, got %q", cfg.Database.MySQL.Password)
	}
	if cfg.Database.Postgres.TimeZone != "Asia/Shanghai" {
		t.Fatalf("expected env override postgres timezone, got %q", cfg.Database.Postgres.TimeZone)
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

// ---------------------------------------------------------------------------
// LoadWithOverrides
// ---------------------------------------------------------------------------

func TestLoadWithOverrides_MissingOverrideIsSkipped(t *testing.T) {
	base := writeTempConfig(t, `
app:
  name: "Base API"
  version: "1.0.0"
`)
	// Override file does not exist – should not error.
	cfg, err := settings.LoadWithOverrides(base, filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("LoadWithOverrides: %v", err)
	}
	if cfg.App.Name != "Base API" {
		t.Errorf("expected base name, got %q", cfg.App.Name)
	}
}

func TestLoadWithOverrides_OverrideWins(t *testing.T) {
	base := writeTempConfig(t, `
app:
  name: "Base API"
  version: "1.0.0"
server:
  port: 8080
`)
	override := writeTempConfig(t, `
app:
  name: "Override API"
server:
  port: 9090
`)
	cfg, err := settings.LoadWithOverrides(base, override)
	if err != nil {
		t.Fatalf("LoadWithOverrides: %v", err)
	}
	if cfg.App.Name != "Override API" {
		t.Errorf("expected overridden name, got %q", cfg.App.Name)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected overridden port 9090, got %d", cfg.Server.Port)
	}
	// Non-overridden field keeps base value.
	if cfg.App.Version != "1.0.0" {
		t.Errorf("expected base version, got %q", cfg.App.Version)
	}
}

func TestLoadWithOverrides_MultipleOverrides(t *testing.T) {
	base := writeTempConfig(t, `
app:
  name: "Base"
server:
  port: 8080
`)
	o1 := writeTempConfig(t, `
server:
  port: 8081
`)
	o2 := writeTempConfig(t, `
server:
  port: 8082
`)
	cfg, err := settings.LoadWithOverrides(base, o1, o2)
	if err != nil {
		t.Fatalf("LoadWithOverrides: %v", err)
	}
	// Last override wins.
	if cfg.Server.Port != 8082 {
		t.Errorf("expected port 8082, got %d", cfg.Server.Port)
	}
}

// ---------------------------------------------------------------------------
// LoadForEnv
// ---------------------------------------------------------------------------

func TestLoadForEnv_AutoMergesEnvFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config.yaml")
	envFile := filepath.Join(dir, "config.production.yaml")

	os.WriteFile(base, []byte(`
app:
  name: "My App"
  env: "production"
server:
  port: 8080
`), 0o644) //nolint:errcheck

	os.WriteFile(envFile, []byte(`
server:
  port: 9443
`), 0o644) //nolint:errcheck

	cfg, err := settings.LoadForEnv(base)
	if err != nil {
		t.Fatalf("LoadForEnv: %v", err)
	}
	if cfg.Server.Port != 9443 {
		t.Errorf("expected env-override port 9443, got %d", cfg.Server.Port)
	}
	if cfg.App.Name != "My App" {
		t.Errorf("expected base name, got %q", cfg.App.Name)
	}
}

func TestLoadForEnv_DefaultsDevelopment(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config.yaml")
	// No app.env set → defaults to "development".
	os.WriteFile(base, []byte(`
server:
  port: 8080
`), 0o644) //nolint:errcheck

	// No config.development.yaml → should just use base.
	cfg, err := settings.LoadForEnv(base)
	if err != nil {
		t.Fatalf("LoadForEnv: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadForEnv_MissingEnvFileIsOK(t *testing.T) {
	base := writeTempConfig(t, `
app:
  env: "staging"
server:
  port: 7777
`)
	// No config.staging.yaml present – should not error.
	cfg, err := settings.LoadForEnv(base)
	if err != nil {
		t.Fatalf("LoadForEnv with missing env file: %v", err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("expected 7777, got %d", cfg.Server.Port)
	}
}

func TestMustLoadWithOverridesPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	// Pass a non-existent base file to trigger an error.
	settings.MustLoadWithOverrides(filepath.Join(t.TempDir(), "missing.yaml"))
}

