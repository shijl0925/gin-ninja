package mysql

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func init() {
	ginbootstrap.MustRegisterDialector(MySQL, "mysql")
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
		Params:               sanitizeParams(cfg.MySQL.Params),
	}
	if charset := strings.TrimSpace(cfg.MySQL.Charset); charset != "" {
		dsnCfg.Params["charset"] = charset
	}
	return dsnCfg, nil
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
