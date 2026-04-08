package settings

import "testing"

func TestEnvOverridePath(t *testing.T) {
	t.Parallel()

	if got := envOverridePath("", "production"); got != "" {
		t.Fatalf("envOverridePath(empty) = %q", got)
	}
	if got := envOverridePath("config.yaml", "production"); got != "config.production.yaml" {
		t.Fatalf("envOverridePath(config.yaml) = %q", got)
	}
	if got := envOverridePath("config/app.yaml", "staging"); got != "config/app.staging.yaml" {
		t.Fatalf("envOverridePath(config/app.yaml) = %q", got)
	}
}

func TestNormalizeDatabaseConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		normalizeDatabaseConfig(nil)
	})

	t.Run("custom dsn unchanged", func(t *testing.T) {
		cfg := &DatabaseConfig{
			Driver: "mysql",
			DSN:    "user:pass@tcp(localhost:3306)/app",
			MySQL:  MySQLConfig{Host: "127.0.0.1", User: "root", Name: "app"},
		}
		normalizeDatabaseConfig(cfg)
		if cfg.DSN == "" {
			t.Fatal("expected custom DSN to be preserved")
		}
	})

	t.Run("mysql structured config clears default dsn", func(t *testing.T) {
		cfg := &DatabaseConfig{
			Driver: "mysql",
			DSN:    " app.db ",
			MySQL:  MySQLConfig{Port: 3307},
		}
		normalizeDatabaseConfig(cfg)
		if cfg.DSN != "" {
			t.Fatalf("expected default DSN to be cleared, got %q", cfg.DSN)
		}
	})

	t.Run("postgres structured config clears default dsn", func(t *testing.T) {
		cfg := &DatabaseConfig{
			Driver:   "postgresql",
			DSN:      "app.db",
			Postgres: PostgresConfig{Params: map[string]string{"search_path": "public"}},
		}
		normalizeDatabaseConfig(cfg)
		if cfg.DSN != "" {
			t.Fatalf("expected default DSN to be cleared, got %q", cfg.DSN)
		}
	})
}

func TestStructuredDatabaseConfigHelpers(t *testing.T) {
	t.Parallel()

	if !isDefaultDatabaseDSN(" app.db ") {
		t.Fatal("expected trimmed default database DSN to match")
	}
	if isDefaultDatabaseDSN("custom.db") {
		t.Fatal("expected custom database DSN not to match default")
	}

	if hasMeaningfulStructuredDBConfig("", "", "", "", nil, 0, 3306) {
		t.Fatal("expected empty structured config to be ignored")
	}
	if !hasMeaningfulStructuredDBConfig("", "", "", "", map[string]string{"charset": "utf8mb4"}, 0, 3306) {
		t.Fatal("expected params to make structured config meaningful")
	}

	if !(MySQLConfig{Port: 3307}).IsConfigured() {
		t.Fatal("expected non-default mysql port to count as configured")
	}
	if (MySQLConfig{Port: 3306}).IsConfigured() {
		t.Fatal("expected default mysql port alone not to count as configured")
	}

	if !(PostgresConfig{Name: "app"}).IsConfigured() {
		t.Fatal("expected postgres name to count as configured")
	}
	if (PostgresConfig{Port: 5432}).IsConfigured() {
		t.Fatal("expected default postgres port alone not to count as configured")
	}
}
