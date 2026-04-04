package bootstrap

import (
	"fmt"

	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/gorm"
)

// buildDialector returns the GORM Dialector for the given DatabaseConfig.
// Drivers are resolved lazily so that only the drivers you actually import
// contribute to your binary.
func buildDialector(cfg *settings.DatabaseConfig) (gorm.Dialector, error) {
	switch cfg.Driver {
	case "sqlite", "sqlite3":
		return sqliteDialector(cfg.DSN)
	case "mysql":
		return mysqlDialector(cfg.DSN)
	case "postgres", "postgresql":
		return postgresDialector(cfg.DSN)
	default:
		return nil, fmt.Errorf("bootstrap: unsupported database driver %q", cfg.Driver)
	}
}
