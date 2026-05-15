package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shijl0925/gin-ninja/examples/full/app"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/settings"
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
	orm.Init(db)

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

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /docs status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/examples/features", nil)
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/examples/features status = %d", rec.Code)
	}
}
