// Package logger provides a Zap-based structured logger for gin-ninja
// applications.
//
// Usage:
//
//	import "github.com/shijl0925/gin-ninja/pkg/logger"
//
//	// In main / bootstrap:
//	log := logger.New(settings.Global.Log)
//	logger.SetGlobal(log)
//
//	// In handlers:
//	logger.Info("user created", zap.Uint("id", user.ID))
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/shijl0925/gin-ninja/settings"
)

var global *zap.Logger

// New creates a new *zap.Logger configured from the supplied LogConfig.
func New(cfg settings.LogConfig) *zap.Logger {
	level := parseLevel(cfg.Level)
	encoder := buildEncoder(cfg.Format)
	sink := buildSink(cfg.Output)

	core := zapcore.NewCore(encoder, sink, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
}

// SetGlobal replaces the package-level logger that is used by the top-level
// helper functions (Info, Warn, Error, etc.).
func SetGlobal(l *zap.Logger) {
	global = l
	zap.ReplaceGlobals(l)
}

// Global returns the package-level *zap.Logger.  If SetGlobal has not been
// called, a default production logger is returned.
func Global() *zap.Logger {
	if global == nil {
		l, _ := zap.NewProduction()
		return l
	}
	return global
}

// Named returns a child logger with the given name.
func Named(name string) *zap.Logger {
	return Global().Named(name)
}

// With returns a child logger pre-populated with the supplied fields.
func With(fields ...zap.Field) *zap.Logger {
	return Global().With(fields...)
}

// Debug logs a message at DEBUG level.
func Debug(msg string, fields ...zap.Field) { Global().Debug(msg, fields...) }

// Info logs a message at INFO level.
func Info(msg string, fields ...zap.Field) { Global().Info(msg, fields...) }

// Warn logs a message at WARN level.
func Warn(msg string, fields ...zap.Field) { Global().Warn(msg, fields...) }

// Error logs a message at ERROR level.
func Error(msg string, fields ...zap.Field) { Global().Error(msg, fields...) }

// Fatal logs a message at FATAL level and then calls os.Exit(1).
func Fatal(msg string, fields ...zap.Field) { Global().Fatal(msg, fields...) }

// Sync flushes any buffered log entries.  Call on application shutdown.
func Sync() { _ = Global().Sync() }

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

func buildEncoder(format string) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder

	if format == "console" {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func buildSink(output string) zapcore.WriteSyncer {
	switch output {
	case "", "stdout":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		f, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			// Fall back to stdout if the file cannot be opened.
			return zapcore.AddSync(os.Stdout)
		}
		return zapcore.AddSync(f)
	}
}
