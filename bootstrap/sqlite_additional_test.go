package bootstrap

import (
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestSQLiteAndPostgresHelpers(t *testing.T) {
	t.Parallel()

	if _, err := sqliteDialector(""); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("expected sqlite DSN validation error, got %v", err)
	}

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

	mysqlCfg := settings.DatabaseConfig{
		MySQL: settings.MySQLConfig{
			Host:      "127.0.0.1",
			Port:      3307,
			User:      "root",
			Password:  "secret",
			Name:      "gin_ninja",
			Charset:   "utf8mb4",
			ParseTime: true,
			Loc:       "UTC",
			Params:    map[string]string{"tls": "skip-verify", " ": "ignored"},
		},
	}
	dsn, err := mysqlDSN(mysqlCfg)
	if err != nil {
		t.Fatalf("mysqlDSN: %v", err)
	}
	for _, want := range []string{"root:secret@", "127.0.0.1:3307", "gin_ninja", "charset=utf8mb4", "parseTime=true"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("expected mysql DSN to contain %q, got %q", want, dsn)
		}
	}
	if _, err := mysqlDialector(mysqlCfg); err != nil {
		t.Fatalf("mysqlDialector structured: %v", err)
	}

	rawMySQL := settings.DatabaseConfig{Driver: "mysql", DSN: "root:secret%21@tcp(localhost:3306)/app"}
	if _, err := mysqlDialector(rawMySQL); err != nil {
		t.Fatalf("mysqlDialector raw: %v", err)
	}

	if _, err := mysqlDriverConfig(settings.DatabaseConfig{MySQL: settings.MySQLConfig{Name: "app"}}); err == nil || !strings.Contains(err.Error(), "host must not be empty") {
		t.Fatalf("expected mysql host validation error, got %v", err)
	}
	if _, err := mysqlDriverConfig(settings.DatabaseConfig{MySQL: settings.MySQLConfig{Host: "127.0.0.1"}}); err == nil || !strings.Contains(err.Error(), "database name must not be empty") {
		t.Fatalf("expected mysql database validation error, got %v", err)
	}

	if _, err := postgresDSN(settings.DatabaseConfig{
		Postgres: settings.PostgresConfig{
			Host: "127.0.0.1",
			User: "postgres",
		},
	}); err == nil || !strings.Contains(err.Error(), "database name must not be empty") {
		t.Fatalf("expected postgres database validation error, got %v", err)
	}

	pgRaw := settings.DatabaseConfig{Driver: "postgres", DSN: "postgres://localhost/app"}
	if dsn, err := postgresDSN(pgRaw); err != nil || dsn != pgRaw.DSN {
		t.Fatalf("expected raw postgres dsn passthrough, got %q err=%v", dsn, err)
	}
	if _, err := postgresDialector(pgRaw); err != nil {
		t.Fatalf("postgresDialector raw: %v", err)
	}

	pgStructured := settings.DatabaseConfig{
		Postgres: settings.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5433,
			User:     "postgres",
			Password: "se cret",
			Name:     "gin_ninja",
			SSLMode:  "disable",
			TimeZone: "Asia/Shanghai",
			Params:   map[string]string{"application_name": "gin ninja", "search_path": "public"},
		},
	}
	pgDSN, err := postgresDSN(pgStructured)
	if err != nil {
		t.Fatalf("postgresDSN structured: %v", err)
	}
	for _, want := range []string{"host=127.0.0.1", "port=5433", "user=postgres", "dbname=gin_ninja", "sslmode=disable", "TimeZone=Asia/Shanghai", "application_name='gin ninja'"} {
		if !strings.Contains(pgDSN, want) {
			t.Fatalf("expected postgres DSN to contain %q, got %q", want, pgDSN)
		}
	}
}
