package internaldialects

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/shijl0925/gin-ninja/settings"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SQLite(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("bootstrap: sqlite DSN must not be empty")
	}
	return sqlite.Open(cfg.DSN), nil
}

func MySQL(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	const defaultStringSize uint = 191

	if useRawMySQLDSN(cfg) {
		dsn, err := MySQLDSN(cfg)
		if err != nil {
			return nil, err
		}
		decodedDSN, err := decodeRawMySQLDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: decode mysql DSN: %w", err)
		}
		return gormmysql.New(gormmysql.Config{
			DSN:               decodedDSN,
			DefaultStringSize: defaultStringSize,
		}), nil
	}

	dsnCfg, err := mySQLDriverConfig(cfg)
	if err != nil {
		return nil, err
	}
	return gormmysql.New(gormmysql.Config{
		DSNConfig:         dsnCfg,
		DefaultStringSize: defaultStringSize,
	}), nil
}

func Postgres(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	dsn, err := PostgresDSN(cfg)
	if err != nil {
		return nil, err
	}
	return postgres.Open(dsn), nil
}

func MySQLDSN(cfg settings.DatabaseConfig) (string, error) {
	if useRawMySQLDSN(cfg) {
		return cfg.DSN, nil
	}
	dsnCfg, err := mySQLDriverConfig(cfg)
	if err != nil {
		return "", err
	}
	return dsnCfg.FormatDSN(), nil
}

func mySQLDriverConfig(cfg settings.DatabaseConfig) (*drivermysql.Config, error) {
	if !cfg.MySQL.IsConfigured() {
		return nil, fmt.Errorf("bootstrap: mysql DSN must not be empty")
	}
	if strings.TrimSpace(cfg.MySQL.Host) == "" {
		return nil, fmt.Errorf("bootstrap: mysql host must not be empty")
	}
	if strings.TrimSpace(cfg.MySQL.Name) == "" {
		return nil, fmt.Errorf("bootstrap: mysql database name must not be empty")
	}

	port := cfg.MySQL.Port
	if port <= 0 {
		port = 3306
	}

	dsnCfg := &drivermysql.Config{
		User:                 strings.TrimSpace(cfg.MySQL.User),
		Passwd:               cfg.MySQL.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(strings.TrimSpace(cfg.MySQL.Host), strconv.Itoa(port)),
		DBName:               strings.TrimSpace(cfg.MySQL.Name),
		AllowNativePasswords: true,
		ParseTime:            cfg.MySQL.ParseTime,
		Loc:                  timeLocation(cfg.MySQL.Loc),
		Params:               SanitizeParams(cfg.MySQL.Params),
	}
	if charset := strings.TrimSpace(cfg.MySQL.Charset); charset != "" {
		dsnCfg.Params["charset"] = charset
	}
	return dsnCfg, nil
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

	params := SanitizeParams(cfg.Postgres.Params)
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

func SanitizeParams(params map[string]string) map[string]string {
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

func DecodeRawMySQLDSN(dsn string) (string, error) {
	return decodeRawMySQLDSN(dsn)
}

func TimeLocation(raw string) *time.Location {
	return timeLocation(raw)
}

func ShouldIgnoreImplicitDefaultDSN(dsn, driver string, hasStructuredConfig bool) bool {
	return shouldIgnoreImplicitDefaultDSN(dsn, driver, hasStructuredConfig)
}

func decodeRawMySQLDSN(dsn string) (string, error) {
	at := strings.IndexByte(dsn, '@')
	if at < 0 {
		return url.PathUnescape(dsn)
	}
	prefix, err := url.PathUnescape(dsn[:at])
	if err != nil {
		return "", err
	}
	return prefix + dsn[at:], nil
}

func postgresDSNPair(key, value string) string {
	return key + "=" + PostgresDSNValue(value)
}

func timeLocation(raw string) *time.Location {
	name := strings.TrimSpace(raw)
	if name == "" {
		name = "Local"
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.Local
}

func useRawMySQLDSN(cfg settings.DatabaseConfig) bool {
	return strings.TrimSpace(cfg.DSN) != "" && !shouldIgnoreImplicitDefaultDSN(cfg.DSN, cfg.Driver, cfg.MySQL.IsConfigured())
}

func useRawPostgresDSN(cfg settings.DatabaseConfig) bool {
	return strings.TrimSpace(cfg.DSN) != "" && !shouldIgnoreImplicitDefaultDSN(cfg.DSN, cfg.Driver, cfg.Postgres.IsConfigured())
}

func shouldIgnoreImplicitDefaultDSN(dsn, driver string, hasStructuredConfig bool) bool {
	trimmedDriver := strings.TrimSpace(driver)
	return hasStructuredConfig && trimmedDriver != "sqlite" && trimmedDriver != "sqlite3" && strings.TrimSpace(dsn) == "app.db"
}
