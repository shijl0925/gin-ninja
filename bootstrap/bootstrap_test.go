package bootstrap

import (
	"testing"

	applogger "github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"
)

func TestBuildDialector(t *testing.T) {
	cases := []struct {
		name    string
		cfg     settings.DatabaseConfig
		wantErr bool
	}{
		{name: "sqlite", cfg: settings.DatabaseConfig{Driver: "sqlite", DSN: "file::memory:?cache=shared"}},
		{name: "sqlite3", cfg: settings.DatabaseConfig{Driver: "sqlite3", DSN: "file::memory:?cache=shared"}},
		{name: "mysql", cfg: settings.DatabaseConfig{Driver: "mysql", DSN: "dsn"}, wantErr: true},
		{name: "postgres", cfg: settings.DatabaseConfig{Driver: "postgres", DSN: "dsn"}, wantErr: true},
		{name: "unsupported", cfg: settings.DatabaseConfig{Driver: "oracle", DSN: "dsn"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dialector, err := buildDialector(&tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || dialector == nil {
				t.Fatalf("expected dialector, got dialector=%v err=%v", dialector, err)
			}
		})
	}
}

func TestSQLiteDialectorRequiresDSN(t *testing.T) {
	if _, err := sqliteDialector(""); err == nil {
		t.Fatal("expected sqlite dsn validation error")
	}
}

func TestInitDBAndMustInitDB(t *testing.T) {
	cfg := &settings.DatabaseConfig{
		Driver:                 "sqlite",
		DSN:                    "file::memory:?cache=shared",
		MaxIdleConns:           1,
		MaxOpenConns:           2,
		ConnMaxLifetimeMinutes: 1,
	}

	db, err := InitDB(cfg)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("expected query to succeed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 2 {
		t.Fatalf("expected max open connections to be set, got %d", sqlDB.Stats().MaxOpenConnections)
	}

	if MustInitDB(cfg) == nil {
		t.Fatal("expected MustInitDB to return db")
	}
}

func TestMustInitDBPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInitDB(&settings.DatabaseConfig{Driver: "sqlite"})
}

func TestInitLoggerSetsGlobal(t *testing.T) {
	cfg := &settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"}
	logger := InitLogger(cfg)
	if logger == nil {
		t.Fatal("expected logger")
	}
	if applogger.Global() != logger {
		t.Fatal("expected InitLogger to replace global logger")
	}
}
