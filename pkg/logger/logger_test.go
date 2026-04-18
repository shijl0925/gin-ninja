package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]zapcore.Level{
		"debug":   zapcore.DebugLevel,
		"warn":    zapcore.WarnLevel,
		"warning": zapcore.WarnLevel,
		"error":   zapcore.ErrorLevel,
		"fatal":   zapcore.FatalLevel,
		"info":    zapcore.InfoLevel,
		"":        zapcore.InfoLevel,
	}

	for input, want := range cases {
		if got := parseLevel(input); got != want {
			t.Fatalf("parseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestBuildEncoder(t *testing.T) {
	jsonEncoder := buildEncoder("json", false)
	jsonEntry, err := jsonEncoder.EncodeEntry(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}, nil)
	if err != nil {
		t.Fatalf("EncodeEntry(json): %v", err)
	}
	if !strings.Contains(jsonEntry.String(), `"msg":"hello"`) {
		t.Fatalf("expected json log output, got %s", jsonEntry.String())
	}

	consoleEncoder := buildEncoder("console", true)
	consoleEntry, err := consoleEncoder.EncodeEntry(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}, nil)
	if err != nil {
		t.Fatalf("EncodeEntry(console): %v", err)
	}
	if !strings.Contains(consoleEntry.String(), "hello") {
		t.Fatalf("expected console log output, got %s", consoleEntry.String())
	}

	plainConsoleEncoder := buildEncoder("console", false)
	plainConsoleEntry, err := plainConsoleEncoder.EncodeEntry(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}, nil)
	if err != nil {
		t.Fatalf("EncodeEntry(console plain): %v", err)
	}
	if strings.Contains(plainConsoleEntry.String(), "\x1b[") {
		t.Fatalf("expected plain console log output without ANSI color codes, got %q", plainConsoleEntry.String())
	}
}

func TestBuildSinkAndHelpers(t *testing.T) {
	globalMu.RLock()
	oldGlobal := global
	globalMu.RUnlock()
	t.Cleanup(func() {
		globalMu.Lock()
		global = oldGlobal
		globalMu.Unlock()
		if oldGlobal != nil {
			zap.ReplaceGlobals(oldGlobal)
		}
	})

	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	sink := buildSink(settings.LogConfig{Output: logFile})
	if _, err := sink.Write([]byte("seed")); err != nil {
		t.Fatalf("sink.Write: %v", err)
	}
	if err := sink.Sync(); err != nil {
		t.Fatalf("sink.Sync: %v", err)
	}
	if data, err := os.ReadFile(logFile); err != nil || string(data) != "seed" {
		t.Fatalf("expected sink to write to file, got data=%q err=%v", data, err)
	}

	cfg := settings.LogConfig{
		Level:      "debug",
		Format:     "json",
		Output:     logFile,
		MaxSizeMB:  1,
		MaxAgeDays: 2,
		MaxBackups: 4,
		Compress:   true,
	}
	l := New(cfg)
	if l == nil {
		t.Fatal("expected logger")
	}
	SetGlobal(l)
	if Global() != l {
		t.Fatal("expected SetGlobal to update global logger")
	}
	if Named("component") == nil || With(zap.String("key", "value")) == nil {
		t.Fatal("expected logger helper constructors")
	}

	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	Sync()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	for _, want := range []string{"debug message", "info message", "warn message", "error message"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected log content %q in %s", want, content)
		}
	}

	if sink := buildSink(settings.LogConfig{Output: "stderr"}); sink == nil {
		t.Fatal("expected stderr sink")
	}
}

func TestBuildRollingLoggerDefaultsAndDirectories(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "logs", "app.log")

	rotator, err := buildRollingLogger(settings.LogConfig{
		Output:     logFile,
		MaxSizeMB:  0,
		MaxAgeDays: 0,
		MaxBackups: 0,
	})
	if err != nil {
		t.Fatalf("buildRollingLogger: %v", err)
	}
	if rotator.Filename != logFile {
		t.Fatalf("expected filename %q, got %q", logFile, rotator.Filename)
	}
	if rotator.MaxSize != defaultMaxSizeMB {
		t.Fatalf("expected default max size %d, got %d", defaultMaxSizeMB, rotator.MaxSize)
	}
	if rotator.MaxAge != defaultMaxAgeDays {
		t.Fatalf("expected default max age %d, got %d", defaultMaxAgeDays, rotator.MaxAge)
	}
	if rotator.MaxBackups != defaultMaxBackups {
		t.Fatalf("expected default max backups %d, got %d", defaultMaxBackups, rotator.MaxBackups)
	}

	sink := buildSink(settings.LogConfig{Output: logFile})
	if _, err := sink.Write([]byte("rotating")); err != nil {
		t.Fatalf("sink.Write: %v", err)
	}
	if err := sink.Sync(); err != nil {
		t.Fatalf("sink.Sync: %v", err)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "rotating" {
		t.Fatalf("expected nested log file write, got %q", string(data))
	}
}

func TestNewWithFileOutputMirrorsToStdoutAndUsesPlainFileLogs(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	l := New(settings.LogConfig{
		Level:  "info",
		Format: "console",
		Output: logFile,
	})
	l.Info("mirrored message")
	_ = l.Sync()
	if err := w.Close(); err != nil {
		t.Fatalf("Close stdout pipe writer: %v", err)
	}

	stdoutData, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout): %v", err)
	}
	stdoutText := string(stdoutData)
	if !strings.Contains(stdoutText, "mirrored message") {
		t.Fatalf("expected mirrored stdout log, got %q", stdoutText)
	}

	fileData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	fileText := string(fileData)
	if !strings.Contains(fileText, "mirrored message") {
		t.Fatalf("expected file log output, got %q", fileText)
	}
	if strings.Contains(fileText, "\x1b[") {
		t.Fatalf("expected file log output without ANSI color codes, got %q", fileText)
	}
}

func TestSetGlobalNilFallsBackAndIsConcurrentSafe(t *testing.T) {
	globalMu.RLock()
	oldGlobal := global
	globalMu.RUnlock()
	t.Cleanup(func() {
		globalMu.Lock()
		global = oldGlobal
		globalMu.Unlock()
		if oldGlobal != nil {
			zap.ReplaceGlobals(oldGlobal)
		}
	})

	SetGlobal(nil)
	if Global() == nil {
		t.Fatal("expected fallback logger when setting nil global")
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				SetGlobal(zap.NewNop())
				return
			}
			if Global() == nil {
				t.Error("expected non-nil logger during concurrent access")
			}
		}(i)
	}
	wg.Wait()
}
