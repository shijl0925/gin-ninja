package logger

import (
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
	jsonEncoder := buildEncoder("json")
	jsonEntry, err := jsonEncoder.EncodeEntry(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}, nil)
	if err != nil {
		t.Fatalf("EncodeEntry(json): %v", err)
	}
	if !strings.Contains(jsonEntry.String(), `"msg":"hello"`) {
		t.Fatalf("expected json log output, got %s", jsonEntry.String())
	}

	consoleEncoder := buildEncoder("console")
	consoleEntry, err := consoleEncoder.EncodeEntry(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}, nil)
	if err != nil {
		t.Fatalf("EncodeEntry(console): %v", err)
	}
	if !strings.Contains(consoleEntry.String(), "hello") {
		t.Fatalf("expected console log output, got %s", consoleEntry.String())
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

	sink := buildSink(logFile)
	if _, err := sink.Write([]byte("seed")); err != nil {
		t.Fatalf("sink.Write: %v", err)
	}
	if err := sink.Sync(); err != nil {
		t.Fatalf("sink.Sync: %v", err)
	}
	if data, err := os.ReadFile(logFile); err != nil || string(data) != "seed" {
		t.Fatalf("expected sink to write to file, got data=%q err=%v", data, err)
	}

	cfg := settings.LogConfig{Level: "debug", Format: "json", Output: logFile}
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

	if sink := buildSink("stderr"); sink == nil {
		t.Fatal("expected stderr sink")
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
