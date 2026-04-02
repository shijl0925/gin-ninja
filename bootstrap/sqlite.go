package bootstrap

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func sqliteDialector(dsn string) (gorm.Dialector, error) {
	if dsn == "" {
		return nil, fmt.Errorf("bootstrap: sqlite DSN must not be empty")
	}
	return sqlite.Open(dsn), nil
}

// mysqlDialector and postgresDialector are stub implementations.
// To use MySQL or PostgreSQL, import the appropriate GORM driver in your
// application and override these functions, or add build-tag-based files.

func mysqlDialector(dsn string) (gorm.Dialector, error) {
	return nil, fmt.Errorf("bootstrap: MySQL support requires importing gorm.io/driver/mysql " +
		"and registering it via bootstrap.RegisterDialector(\"mysql\", ...)")
}

func postgresDialector(dsn string) (gorm.Dialector, error) {
	return nil, fmt.Errorf("bootstrap: PostgreSQL support requires importing gorm.io/driver/postgres " +
		"and registering it via bootstrap.RegisterDialector(\"postgres\", ...)")
}
