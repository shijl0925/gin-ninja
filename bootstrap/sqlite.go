package bootstrap

import (
	"fmt"
	"net/url"

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

func mysqlDialector(dsn string) (gorm.Dialector, error) {
	if dsn == "" {
		return nil, fmt.Errorf("bootstrap: mysql DSN must not be empty")
	}
	decodedDSN, err := url.QueryUnescape(dsn)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: decode mysql DSN: %w", err)
	}
	return gormmysql.Open(decodedDSN), nil
}

func postgresDialector(dsn string) (gorm.Dialector, error) {
	if dsn == "" {
		return nil, fmt.Errorf("bootstrap: postgres DSN must not be empty")
	}
	return postgres.Open(dsn), nil
}
