package settings

import (
	"testing"
)

// ---------------------------------------------------------------------------
// expandPlaceholders
// ---------------------------------------------------------------------------

func TestExpandPlaceholders_NoPlaceholder(t *testing.T) {
	if got := expandPlaceholders("hello world"); got != "hello world" {
		t.Fatalf("expected unchanged string, got %q", got)
	}
}

func TestExpandPlaceholders_EnvSetReplacesToken(t *testing.T) {
	t.Setenv("NINJA_TEST_HOST", "db.example.com")
	if got := expandPlaceholders("${NINJA_TEST_HOST}"); got != "db.example.com" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestExpandPlaceholders_EnvSetOverridesDefault(t *testing.T) {
	t.Setenv("NINJA_TEST_PORT", "5433")
	if got := expandPlaceholders("${NINJA_TEST_PORT:5432}"); got != "5433" {
		t.Fatalf("expected env value to override default, got %q", got)
	}
}

func TestExpandPlaceholders_EnvUnsetUsesDefault(t *testing.T) {
	t.Setenv("NINJA_TEST_ABSENT", "")
	if got := expandPlaceholders("${NINJA_TEST_ABSENT:localhost}"); got != "localhost" {
		t.Fatalf("expected default value, got %q", got)
	}
}

func TestExpandPlaceholders_DefaultContainsColon(t *testing.T) {
	t.Setenv("NINJA_TEST_DSN", "")
	// Default value itself contains colons – only the first colon is the separator.
	got := expandPlaceholders("${NINJA_TEST_DSN:host=pg port=5432 user=postgres}")
	if got != "host=pg port=5432 user=postgres" {
		t.Fatalf("expected default with spaces, got %q", got)
	}
}

func TestExpandPlaceholders_EnvUnsetNoDefaultBecomesEmpty(t *testing.T) {
	t.Setenv("NINJA_TEST_EMPTY", "")
	if got := expandPlaceholders("${NINJA_TEST_EMPTY}"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestExpandPlaceholders_MultiplePlaceholders(t *testing.T) {
	t.Setenv("NINJA_TEST_U", "admin")
	t.Setenv("NINJA_TEST_P", "")
	got := expandPlaceholders("${NINJA_TEST_U}:${NINJA_TEST_P:secret}@localhost")
	want := "admin:secret@localhost"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExpandConfigStrings_ExpandsNestedFields(t *testing.T) {
	t.Setenv("NINJA_TEST_DB_PASS", "s3cr3t")
	cfg := &Config{
		Database: DatabaseConfig{
			DSN: "${NINJA_TEST_DB_DSN:app.db}",
			MySQL: MySQLConfig{
				Password: "${NINJA_TEST_DB_PASS}",
				Host:     "${NINJA_TEST_DB_HOST:127.0.0.1}",
			},
		},
	}
	expandConfigStrings(cfg)
	if cfg.Database.DSN != "app.db" {
		t.Fatalf("expected default dsn, got %q", cfg.Database.DSN)
	}
	if cfg.Database.MySQL.Password != "s3cr3t" {
		t.Fatalf("expected expanded password, got %q", cfg.Database.MySQL.Password)
	}
	if cfg.Database.MySQL.Host != "127.0.0.1" {
		t.Fatalf("expected default host, got %q", cfg.Database.MySQL.Host)
	}
}

func TestExpandConfigStrings_ExpandsMapValues(t *testing.T) {
	t.Setenv("NINJA_TEST_PARAM_VAL", "utf8mb4")
	cfg := &Config{
		Database: DatabaseConfig{
			MySQL: MySQLConfig{
				Params: map[string]string{
					"charset": "${NINJA_TEST_PARAM_VAL:latin1}",
				},
			},
		},
	}
	expandConfigStrings(cfg)
	if got := cfg.Database.MySQL.Params["charset"]; got != "utf8mb4" {
		t.Fatalf("expected expanded map value, got %q", got)
	}
}

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
