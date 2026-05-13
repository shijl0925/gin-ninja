package postgres

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	ginbootstrap.MustRegisterDialector(Postgres, "postgres", "postgresql")
}

func Postgres(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	dsn, err := PostgresDSN(cfg)
	if err != nil {
		return nil, err
	}
	return postgres.Open(dsn), nil
}

func PostgresDSN(cfg settings.DatabaseConfig) (string, error) {
	if useRawPostgresDSN(cfg) {
		return cfg.DSN, nil
	}
	if !cfg.Postgres.IsConfigured() {
		return "", fmt.Errorf("bootstrap: postgres DSN must not be empty")
	}
	if strings.TrimSpace(cfg.Postgres.Host) == "" {
		return "", fmt.Errorf("bootstrap: postgres host must not be empty")
	}
	if strings.TrimSpace(cfg.Postgres.Name) == "" {
		return "", fmt.Errorf("bootstrap: postgres database name must not be empty")
	}
	if strings.TrimSpace(cfg.Postgres.User) == "" && strings.TrimSpace(cfg.Postgres.Password) != "" {
		return "", fmt.Errorf("bootstrap: postgres user must not be empty when password is provided")
	}

	port := cfg.Postgres.Port
	if port <= 0 {
		port = 5432
	}

	parts := []string{
		postgresDSNPair("host", strings.TrimSpace(cfg.Postgres.Host)),
		postgresDSNPair("port", strconv.Itoa(port)),
		postgresDSNPair("dbname", strings.TrimSpace(cfg.Postgres.Name)),
	}
	if user := strings.TrimSpace(cfg.Postgres.User); user != "" {
		parts = append(parts, postgresDSNPair("user", user))
	}
	if cfg.Postgres.Password != "" {
		parts = append(parts, postgresDSNPair("password", cfg.Postgres.Password))
	}
	if sslmode := strings.TrimSpace(cfg.Postgres.SSLMode); sslmode != "" {
		parts = append(parts, postgresDSNPair("sslmode", sslmode))
	}
	if timeZone := strings.TrimSpace(cfg.Postgres.TimeZone); timeZone != "" {
		parts = append(parts, postgresDSNPair("TimeZone", timeZone))
	}

	params := sanitizeParams(cfg.Postgres.Params)
	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for key := range params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, postgresDSNPair(key, params[key]))
		}
	}

	return strings.Join(parts, " "), nil
}

func postgresDSNPair(key, value string) string {
	return key + "=" + PostgresDSNValue(value)
}

func PostgresDSNValue(value string) string {
	if value == "" {
		return "''"
	}
	if strings.ContainsAny(value, " \t\n\r\v\f'\\") {
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `'`, `\'`)
		return "'" + escaped + "'"
	}
	return value
}

func useRawPostgresDSN(cfg settings.DatabaseConfig) bool {
	return strings.TrimSpace(cfg.DSN) != "" && !shouldIgnoreImplicitDefaultDSN(cfg.DSN, cfg.Driver, cfg.Postgres.IsConfigured())
}

func shouldIgnoreImplicitDefaultDSN(dsn, driver string, hasStructuredConfig bool) bool {
	trimmedDriver := strings.TrimSpace(driver)
	return hasStructuredConfig && trimmedDriver != "sqlite" && trimmedDriver != "sqlite3" && strings.TrimSpace(dsn) == "app.db"
}

func sanitizeParams(params map[string]string) map[string]string {
	values := make(map[string]string, len(params))
	for key, value := range params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		values[trimmedKey] = value
	}
	return values
}
