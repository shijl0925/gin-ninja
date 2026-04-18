package logger

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
)

func TestBuildSinkCreatesMissingDirectoriesAndGlobalFallback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "app.log")
	sink := buildSink(settings.LogConfig{Output: dir})
	if sink == nil {
		t.Fatal("expected sink")
	}
	if _, err := sink.Write([]byte("hello")); err != nil {
		t.Fatalf("sink.Write: %v", err)
	}
	if err := sink.Sync(); err != nil {
		t.Fatalf("sink.Sync: %v", err)
	}
	if data, err := os.ReadFile(dir); err != nil || string(data) != "hello" {
		t.Fatalf("expected sink to create nested file, got data=%q err=%v", data, err)
	}

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

	globalMu.Lock()
	global = nil
	globalMu.Unlock()
	if Global() == nil {
		t.Fatal("expected fallback global logger")
	}
}

func TestFatalExitsProcess(t *testing.T) {
	if os.Getenv("GIN_NINJA_LOGGER_FATAL_HELPER") == "1" {
		Fatal("boom")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatalExitsProcess")
	cmd.Env = append(os.Environ(), "GIN_NINJA_LOGGER_FATAL_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected subprocess to exit with failure")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit from Fatal, got %v", err)
	}
}
