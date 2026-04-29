package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestBuildSinkCreatesMissingDirectories(t *testing.T) {
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
}
