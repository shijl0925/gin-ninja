package bootstrap

import (
	"fmt"

	"gorm.io/driver/mysql"
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
	return mysql.Open(dsn), nil
}

func postgresDialector(dsn string) (gorm.Dialector, error) {
	if dsn == "" {
		return nil, fmt.Errorf("bootstrap: postgres DSN must not be empty")
	}
	return postgres.Open(dsn), nil
}
