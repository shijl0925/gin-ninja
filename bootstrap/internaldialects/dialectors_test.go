package internaldialects

import (
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestInternalDialectorsDSNHelpers(t *testing.T) {
	if _, err := SQLite(settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected empty sqlite DSN error")
	}
	if _, err := SQLite(settings.DatabaseConfig{DSN: ":memory:"}); err != nil {
		t.Fatalf("SQLite :memory:: %v", err)
	}

	mysqlDSN, err := MySQLDSN(settings.DatabaseConfig{MySQL: settings.MySQLConfig{
		Host: "localhost",
		Name: "app",
		User: "root",
		Params: map[string]string{
			" parseTime ": "true",
			"":            "ignored",
		},
	}})
	if err != nil {
		t.Fatalf("MySQLDSN: %v", err)
	}
	if !strings.Contains(mysqlDSN, "parseTime=true") {
		t.Fatalf("expected sanitized param in mysql DSN: %s", mysqlDSN)
	}

	pgDSN, err := PostgresDSN(settings.DatabaseConfig{Postgres: settings.PostgresConfig{
		Host:     "localhost",
		Name:     "app",
		User:     "pg user",
		Password: "p'ass",
		Params:   map[string]string{" application_name ": "gin ninja"},
	}})
	if err != nil {
		t.Fatalf("PostgresDSN: %v", err)
	}
	for _, want := range []string{"host=localhost", "port=5432", "user='pg user'", `password='p\'ass'`, "application_name='gin ninja'"} {
		if !strings.Contains(pgDSN, want) {
			t.Fatalf("PostgresDSN missing %q in %q", want, pgDSN)
		}
	}

	if got := PostgresDSNValue(""); got != "''" {
		t.Fatalf("empty PostgresDSNValue = %q", got)
	}
	if got, err := DecodeRawMySQLDSN("user%3Apass@tcp(localhost:3306)/app"); err != nil || got != "user:pass@tcp(localhost:3306)/app" {
		t.Fatalf("DecodeRawMySQLDSN = %q, %v", got, err)
	}
	if TimeLocation("not/a-zone") == nil {
		t.Fatal("TimeLocation should fall back to local")
	}
	if !ShouldIgnoreImplicitDefaultDSN("app.db", "mysql", true) {
		t.Fatal("expected implicit sqlite DSN to be ignored for structured mysql config")
	}
}

func TestInternalDialectorsConstructors(t *testing.T) {
	if _, err := MySQL(settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected empty mysql config error")
	}
	if _, err := MySQL(settings.DatabaseConfig{DSN: "%zz@tcp(localhost:3306)/app"}); err == nil {
		t.Fatal("expected malformed raw mysql DSN error")
	}
	if dialector, err := MySQL(settings.DatabaseConfig{DSN: "user%3Apass@tcp(localhost:3306)/app"}); err != nil || dialector == nil {
		t.Fatalf("MySQL raw DSN = (%v, %v), want dialector", dialector, err)
	}
	if dialector, err := MySQL(settings.DatabaseConfig{MySQL: settings.MySQLConfig{
		Host: "127.0.0.1",
		Name: "app",
		User: "root",
	}}); err != nil || dialector == nil {
		t.Fatalf("MySQL structured config = (%v, %v), want dialector", dialector, err)
	}

	if _, err := Postgres(settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected empty postgres config error")
	}
	if dialector, err := Postgres(settings.DatabaseConfig{DSN: "host=localhost dbname=app"}); err != nil || dialector == nil {
		t.Fatalf("Postgres raw DSN = (%v, %v), want dialector", dialector, err)
	}
	if dialector, err := Postgres(settings.DatabaseConfig{Postgres: settings.PostgresConfig{
		Host: "127.0.0.1",
		Name: "app",
		User: "postgres",
	}}); err != nil || dialector == nil {
		t.Fatalf("Postgres structured config = (%v, %v), want dialector", dialector, err)
	}
}
