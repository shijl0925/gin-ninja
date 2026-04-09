package bootstrap

import (
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestSQLiteAndPostgresHelpers(t *testing.T) {
	t.Parallel()

	if loc := timeLocation(""); loc == nil {
		t.Fatal("expected default location")
	}
	if loc := timeLocation("Definitely/Invalid"); loc == nil {
		t.Fatal("expected fallback location")
	}

	params := sanitizeParams(map[string]string{" a ": "1", "": "skip"})
	if len(params) != 1 || params["a"] != "1" {
		t.Fatalf("unexpected sanitized params: %+v", params)
	}

	if got := postgresDSNValue(""); got != "''" {
		t.Fatalf("unexpected empty postgres DSN value %q", got)
	}
	if got := postgresDSNValue(`a b'c\`); !strings.HasPrefix(got, "'") {
		t.Fatalf("expected quoted postgres DSN value, got %q", got)
	}

	cfg := settings.DatabaseConfig{
		Postgres: settings.PostgresConfig{
			Host:     "localhost",
			Name:     "app",
			Password: "secret",
		},
	}
	if _, err := postgresDSN(cfg); err == nil || !strings.Contains(err.Error(), "user must not be empty") {
		t.Fatalf("expected postgres user validation error, got %v", err)
	}

	if _, err := decodeRawMySQLDSN("root:bad%zz@tcp(localhost:3306)/app"); err == nil {
		t.Fatal("expected mysql decode error")
	}

	if !shouldIgnoreImplicitDefaultDSN("app.db", "mysql", true) {
		t.Fatal("expected implicit sqlite dsn to be ignored for structured mysql config")
	}
}
