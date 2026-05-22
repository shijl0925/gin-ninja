package sqlite

import (
	"path/filepath"
	"testing"

	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
)

func TestSQLiteDriverIsRegistered(t *testing.T) {
	db, err := ginbootstrap.InitDB(&settings.DatabaseConfig{
		Driver: "sqlite3",
		DSN:    filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("InitDB sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	_ = sqlDB.Close()
}
