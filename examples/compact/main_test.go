package main

import (
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/shijl0925/gin-ninja/examples/full/app"
	"github.com/shijl0925/gin-ninja/settings"
	ninjatest "github.com/shijl0925/gin-ninja/testing"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildCompactAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	if err := db.AutoMigrate(&app.User{}, &app.Role{}, &app.Project{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	cfg := settings.Config{
		App: settings.AppConfig{
			Name:    "compact-test",
			Version: "1.0.0",
		},
		JWT: settings.JWTConfig{
			Secret: "test-secret",
			Issuer: "compact-test",
		},
	}
	api := buildAPI(cfg, db, zap.NewNop())
	client := ninjatest.NewWithT(t, api)

	rec := client.Get("/health")
	if rec.StatusCode != http.StatusOK {
		t.Fatalf("GET /health status = %d", rec.StatusCode)
	}

	rec = client.Get("/docs")
	if rec.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs status = %d", rec.StatusCode)
	}

	rec = client.Get("/api/v1/examples/features")
	if rec.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/examples/features status = %d", rec.StatusCode)
	}
}

func TestCompactInitHelpersAndMain(t *testing.T) {
	store, shutdown := initCacheStore(settings.Config{})
	if store == nil {
		t.Fatal("expected memory cache store")
	}
	if err := shutdown(t.Context()); err != nil {
		t.Fatalf("memory cache shutdown: %v", err)
	}

	store, shutdown = initCacheStore(settings.Config{
		Redis: settings.RedisConfig{Enabled: true, Addr: "127.0.0.1:1"},
	})
	if store == nil {
		t.Fatal("expected fallback memory cache store")
	}
	if err := shutdown(t.Context()); err != nil {
		t.Fatalf("fallback cache shutdown: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "compact.db")
	db, err := initDB(&settings.DatabaseConfig{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("initDB sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	_ = sqlDB.Close()

	origRun := runCompactMain
	origFatal := fatalCompact
	defer func() {
		runCompactMain = origRun
		fatalCompact = origFatal
	}()
	runCompactMain = func(cfg settings.Config, log *zap.Logger) error {
		if cfg.App.Name == "" {
			t.Fatal("expected config loaded by main")
		}
		return errors.New("boom")
	}
	fatalCalled := false
	fatalCompact = func(v ...any) { fatalCalled = true }
	main()
	if !fatalCalled {
		t.Fatal("expected main to call fatal on run error")
	}
}
