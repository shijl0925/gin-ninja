// Package main is the entry-point for the gin-ninja full example.
//
// Run:
//
//	go run ./examples/full
//
// Then visit:
//   - http://localhost:8080/docs
//   - http://localhost:8080/docs/v2
//   - http://localhost:8080/docs/v1
//   - http://localhost:8080/docs/v0
//   - http://localhost:8080/openapi.json
//   - http://localhost:8080/openapi/v2.json
//   - http://localhost:8080/openapi/v1.json
//   - http://localhost:8080/openapi/v0.json
//   - http://localhost:8080/admin/login
//   - http://localhost:8080/admin
//   - http://localhost:8080/admin-prototype
package main

import (
	"log"
	"path/filepath"
	"runtime"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/examples/internal/fullapp"
	"github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var runFullMain = run
var fatalFull = func(v ...any) { log.Fatal(v...) }

func initDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
	return fullapp.InitDB(cfg)
}

func buildAPI(cfg settings.Config, db *gorm.DB, log_ *zap.Logger) *ninja.NinjaAPI {
	return fullapp.BuildAPI(cfg, db, log_, fullapp.FullOptions())
}

func run(cfg settings.Config, log_ *zap.Logger) error {
	return fullapp.Run(cfg, log_, fullapp.FullOptions())
}

func main() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fatalFull("resolve config path")
	}
	cfg := fullapp.MustLoadConfig(filepath.Join(filepath.Dir(file), "config.yaml"))

	log_ := bootstrap.InitLogger(&cfg.Log)
	defer logger.Sync()

	if err := runFullMain(*cfg, log_); err != nil {
		fatalFull(err)
	}
}
