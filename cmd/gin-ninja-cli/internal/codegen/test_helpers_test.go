package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root")
	}

	dir := filepath.Dir(thisFile)
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("resolve repo root")
		}
		dir = parent
	}
}

func localModuleReplaces(t *testing.T, fromDir string) string {
	t.Helper()

	root := repoRoot(t)
	modules := []string{
		"github.com/shijl0925/gin-ninja",
		"github.com/shijl0925/gin-ninja/admin",
		"github.com/shijl0925/gin-ninja/bootstrap",
		"github.com/shijl0925/gin-ninja/cache/redis",
		"github.com/shijl0925/gin-ninja/filter",
		"github.com/shijl0925/gin-ninja/middleware",
		"github.com/shijl0925/gin-ninja/order",
		"github.com/shijl0925/gin-ninja/orm",
		"github.com/shijl0925/gin-ninja/pkg/logger",
		"github.com/shijl0925/gin-ninja/settings",
	}

	var b strings.Builder
	for _, module := range modules {
		rel := strings.TrimPrefix(module, "github.com/shijl0925/gin-ninja")
		target := filepath.Join(root, filepath.FromSlash(rel))
		if rel == "" {
			target = root
		}
		if fromDir != "" {
			if relative, err := filepath.Rel(fromDir, target); err == nil {
				target = relative
			}
		}
		fmt.Fprintf(&b, "replace %s => %s\n", module, filepath.ToSlash(target))
	}
	return b.String()
}
