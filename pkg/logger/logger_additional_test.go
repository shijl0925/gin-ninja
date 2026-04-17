package logger

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestBuildSinkFallbackAndGlobalFallback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "app.log")
	if sink := buildSink(dir); sink == nil {
		t.Fatal("expected fallback sink")
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
