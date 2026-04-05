package bootstrap

import (
	"fmt"
	"net"
	"net/url"
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

func sqliteDialector(dsn string) (gorm.Dialector, error) {
	if dsn == "" {
		return nil, fmt.Errorf("bootstrap: sqlite DSN must not be empty")
	}
	return sqlite.Open(dsn), nil
}

func mysqlDialector(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	dsn, err := mysqlDSN(cfg)
	if err != nil {
		return nil, err
	}
	decodedDSN, err := url.PathUnescape(dsn)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: decode mysql DSN: %w", err)
	}
	return gormmysql.Open(decodedDSN), nil
}

func postgresDialector(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	dsn, err := postgresDSN(cfg)
	if err != nil {
		return nil, err
	}
	return postgres.Open(dsn), nil
}

func mysqlDSN(cfg settings.DatabaseConfig) (string, error) {
	if strings.TrimSpace(cfg.DSN) != "" {
		return cfg.DSN, nil
	}
	if !hasMySQLConfig(cfg.MySQL) {
		return "", fmt.Errorf("bootstrap: mysql DSN must not be empty")
	}

	if strings.TrimSpace(cfg.MySQL.Host) == "" {
		return "", fmt.Errorf("bootstrap: mysql host must not be empty")
	}
	if strings.TrimSpace(cfg.MySQL.Name) == "" {
		return "", fmt.Errorf("bootstrap: mysql database name must not be empty")
	}

	port := cfg.MySQL.Port
	if port <= 0 {
		port = 3306
	}

	dsnCfg := drivermysql.Config{
		User:                 strings.TrimSpace(cfg.MySQL.User),
		Passwd:               cfg.MySQL.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(strings.TrimSpace(cfg.MySQL.Host), strconv.Itoa(port)),
		DBName:               strings.TrimSpace(cfg.MySQL.Name),
		AllowNativePasswords: true,
		ParseTime:            cfg.MySQL.ParseTime,
		Loc:                  timeLocation(cfg.MySQL.Loc),
		Params:               sanitizeParams(cfg.MySQL.Params),
	}
	if charset := strings.TrimSpace(cfg.MySQL.Charset); charset != "" {
		dsnCfg.Params["charset"] = charset
	}
	return dsnCfg.FormatDSN(), nil
}

func postgresDSN(cfg settings.DatabaseConfig) (string, error) {
	if strings.TrimSpace(cfg.DSN) != "" {
		return cfg.DSN, nil
	}
	if !hasPostgresConfig(cfg.Postgres) {
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

	query := url.Values{}
	if sslmode := strings.TrimSpace(cfg.Postgres.SSLMode); sslmode != "" {
		query.Set("sslmode", sslmode)
	}
	if timeZone := strings.TrimSpace(cfg.Postgres.TimeZone); timeZone != "" {
		query.Set("TimeZone", timeZone)
	}
	for key, value := range sanitizeParams(cfg.Postgres.Params) {
		query.Set(key, value)
	}

	dsn := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(strings.TrimSpace(cfg.Postgres.Host), strconv.Itoa(port)),
		Path:     "/" + strings.TrimSpace(cfg.Postgres.Name),
		RawQuery: query.Encode(),
	}
	user := strings.TrimSpace(cfg.Postgres.User)
	switch {
	case user != "" && cfg.Postgres.Password != "":
		dsn.User = url.UserPassword(user, cfg.Postgres.Password)
	case user != "":
		dsn.User = url.User(user)
	}
	return dsn.String(), nil
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

func hasMySQLConfig(cfg settings.MySQLConfig) bool {
	return cfg.Host != "" || cfg.Port != 0 || cfg.User != "" || cfg.Password != "" || cfg.Name != "" || cfg.Charset != "" || cfg.ParseTime || cfg.Loc != "" || len(cfg.Params) > 0
}

func hasPostgresConfig(cfg settings.PostgresConfig) bool {
	return cfg.Host != "" || cfg.Port != 0 || cfg.User != "" || cfg.Password != "" || cfg.Name != "" || cfg.SSLMode != "" || cfg.TimeZone != "" || len(cfg.Params) > 0
}
