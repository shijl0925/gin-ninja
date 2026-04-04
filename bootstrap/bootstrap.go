// Package bootstrap provides helpers for initialising the core application
// components (database, logger) from a settings.Config.
//
// Usage:
//
//	cfg := settings.MustLoad("config.yaml")
//	log := bootstrap.InitLogger(&cfg.Log)
//	db  := bootstrap.MustInitDB(&cfg.Database)
//
//	orm.Init(db)
//	logger.SetGlobal(log)
package bootstrap

import (
	"fmt"
	"time"

	applogger "github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// InitLogger creates a *zap.Logger from the supplied LogConfig and sets it as
// the global logger.
func InitLogger(cfg *settings.LogConfig) *zap.Logger {
	l := applogger.New(*cfg)
	applogger.SetGlobal(l)
	return l
}

// InitDB opens a GORM database connection from the supplied DatabaseConfig.
// It configures the connection pool based on the config values.
func InitDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
	dialector, err := buildDialector(cfg)
	if err != nil {
		return nil, err
	}

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: open db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("bootstrap: get sql.DB: %w", err)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetimeMinutes > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMinutes) * time.Minute)
	}

	return db, nil
}

// MustInitDB calls InitDB and panics on error.
func MustInitDB(cfg *settings.DatabaseConfig) *gorm.DB {
	db, err := InitDB(cfg)
	if err != nil {
		panic(fmt.Sprintf("bootstrap: MustInitDB: %v", err))
	}
	return db
}
