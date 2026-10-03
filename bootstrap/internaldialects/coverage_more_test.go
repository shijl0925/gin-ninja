package internaldialects

import (
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestStructuredDSNValidationBranches(t *testing.T) {
	if _, err := MySQLDSN(settings.DatabaseConfig{MySQL: settings.MySQLConfig{Name: "app"}}); err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("expected mysql host error, got %v", err)
	}
	if _, err := MySQLDSN(settings.DatabaseConfig{MySQL: settings.MySQLConfig{Host: "localhost"}}); err == nil || !strings.Contains(err.Error(), "database name") {
		t.Fatalf("expected mysql database name error, got %v", err)
	}
	if _, err := PostgresDSN(settings.DatabaseConfig{Postgres: settings.PostgresConfig{Name: "app"}}); err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("expected postgres host error, got %v", err)
	}
	if _, err := PostgresDSN(settings.DatabaseConfig{Postgres: settings.PostgresConfig{Host: "localhost"}}); err == nil || !strings.Contains(err.Error(), "database name") {
		t.Fatalf("expected postgres database name error, got %v", err)
	}
	if _, err := PostgresDSN(settings.DatabaseConfig{Postgres: settings.PostgresConfig{Host: "localhost", Name: "app", Password: "secret"}}); err == nil || !strings.Contains(err.Error(), "user") {
		t.Fatalf("expected postgres user error, got %v", err)
	}

	dsn, err := PostgresDSN(settings.DatabaseConfig{Postgres: settings.PostgresConfig{
		Host: "localhost",
		Name: "app",
		Params: map[string]string{
			" z":     "last",
			"":       "ignored",
			"search": "with space",
		},
	}})
	if err != nil {
		t.Fatalf("PostgresDSN: %v", err)
	}
	if strings.Contains(dsn, "ignored") || !strings.Contains(dsn, "search='with space'") || !strings.Contains(dsn, "z=last") {
		t.Fatalf("unexpected postgres params dsn: %q", dsn)
	}
	if got := PostgresDSNValue(`a'b\c`); got != `'a\'b\\c'` {
		t.Fatalf("PostgresDSNValue escaping = %q", got)
	}
	if _, err := DecodeRawMySQLDSN("bad%zz@tcp(localhost:3306)/app"); err == nil {
		t.Fatal("expected bad raw mysql escape before @ to fail")
	}
}
