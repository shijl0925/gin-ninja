package sqlite

import (
	"fmt"
	"strings"

	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	ginbootstrap.MustRegisterDialector(SQLite, "sqlite", "sqlite3")
}

func SQLite(cfg settings.DatabaseConfig) (gorm.Dialector, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("bootstrap: sqlite DSN must not be empty")
	}
	return sqlite.Open(cfg.DSN), nil
}
