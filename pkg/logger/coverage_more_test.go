package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap/zapcore"
)

func TestLoggerCoreAndSinkFallbackBranches(t *testing.T) {
	if core := buildCore(settings.LogConfig{Output: ""}, zapcore.InfoLevel); core == nil {
		t.Fatal("expected stdout core")
	}
	if core := buildCore(settings.LogConfig{Output: "stderr"}, zapcore.InfoLevel); core == nil {
		t.Fatal("expected stderr core")
	}
	if sink := buildSink(settings.LogConfig{Output: ""}); sink == nil {
		t.Fatal("expected stdout sink")
	}

	parentFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parentFile, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	badOutput := filepath.Join(parentFile, "app.log")
	if _, err := buildRollingLogger(settings.LogConfig{Output: badOutput}); err == nil {
		t.Fatal("expected rolling logger directory creation to fail")
	}
	if core := buildCore(settings.LogConfig{Output: badOutput}, zapcore.InfoLevel); core == nil {
		t.Fatal("expected buildCore to fall back to stdout")
	}
	if sink := buildSink(settings.LogConfig{Output: badOutput}); sink == nil {
		t.Fatal("expected buildSink to fall back to stdout")
	}

	rotator, err := buildRollingLogger(settings.LogConfig{Output: "app.log", MaxSizeMB: 9, MaxAgeDays: 8, MaxBackups: 7, Compress: true})
	if err != nil {
		t.Fatalf("buildRollingLogger relative file: %v", err)
	}
	if rotator.MaxSize != 9 || rotator.MaxAge != 8 || rotator.MaxBackups != 7 || !rotator.Compress || !rotator.LocalTime {
		t.Fatalf("unexpected rotator config: %+v", rotator)
	}
}
