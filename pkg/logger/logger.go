// Package logger provides a Zap-based structured logger for gin-ninja
// applications.
//
// Usage:
//
//	import "github.com/shijl0925/gin-ninja/pkg/logger"
//
//	// In main / bootstrap:
//	log := logger.New(cfg.Log)
//
// Pass the returned logger explicitly to middleware and application components.
package logger

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/shijl0925/gin-ninja/settings"
)

const (
	defaultMaxSizeMB  = 100
	defaultMaxAgeDays = 7
	defaultMaxBackups = 3
)

// New creates a new *zap.Logger configured from the supplied LogConfig.
func New(cfg settings.LogConfig) *zap.Logger {
	level := parseLevel(cfg.Level)
	core := buildCore(cfg, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func buildEncoder(format string, colorize bool) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder

	if format == "console" {
		if colorize {
			cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func buildCore(cfg settings.LogConfig, level zapcore.Level) zapcore.Core {
	output := strings.TrimSpace(cfg.Output)
	switch output {
	case "", "stdout":
		return zapcore.NewCore(buildEncoder(cfg.Format, true), zapcore.AddSync(os.Stdout), level)
	case "stderr":
		return zapcore.NewCore(buildEncoder(cfg.Format, true), zapcore.AddSync(os.Stderr), level)
	default:
		rotator, err := buildRollingLogger(cfg)
		if err != nil {
			return zapcore.NewCore(buildEncoder(cfg.Format, true), zapcore.AddSync(os.Stdout), level)
		}
		return zapcore.NewTee(
			zapcore.NewCore(buildEncoder(cfg.Format, true), zapcore.AddSync(os.Stdout), level),
			zapcore.NewCore(buildEncoder(cfg.Format, false), zapcore.AddSync(rotator), level),
		)
	}
}

func buildSink(cfg settings.LogConfig) zapcore.WriteSyncer {
	output := strings.TrimSpace(cfg.Output)
	switch output {
	case "", "stdout":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		rotator, err := buildRollingLogger(cfg)
		if err != nil {
			// Fall back to stdout if the file cannot be opened.
			return zapcore.AddSync(os.Stdout)
		}
		return zapcore.AddSync(rotator)
	}
}

func buildRollingLogger(cfg settings.LogConfig) (*lumberjack.Logger, error) {
	filename := strings.TrimSpace(cfg.Output)
	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    normalizeRotationValue(cfg.MaxSizeMB, defaultMaxSizeMB),
		MaxAge:     normalizeRotationValue(cfg.MaxAgeDays, defaultMaxAgeDays),
		MaxBackups: normalizeRotationValue(cfg.MaxBackups, defaultMaxBackups),
		Compress:   cfg.Compress,
		LocalTime:  true,
	}, nil
}

func normalizeRotationValue(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
