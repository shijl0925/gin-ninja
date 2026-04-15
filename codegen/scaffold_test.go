package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteProjectScaffold(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")
	if err := WriteProjectScaffold(ProjectScaffoldConfig{
		Name:   "mysite",
		Module: "github.com/acme/mysite",
	}, outputDir); err != nil {
		t.Fatalf("WriteProjectScaffold: %v", err)
	}

	for _, rel := range []string{
		"go.mod",
		"main.go",
		"config.yaml",
		"app/models.go",
		"app/repos.go",
		"app/schemas.go",
		"app/apis.go",
		"app/routers.go",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}

	runScaffoldGoTest(t, outputDir)
}

func TestWriteProjectScaffoldStandardTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")
	if err := WriteProjectScaffold(ProjectScaffoldConfig{
		Name:      "mysite",
		Module:    "github.com/acme/mysite",
		AppDir:    "internal/app",
		Template:  "admin",
		WithTests: true,
	}, outputDir); err != nil {
		t.Fatalf("WriteProjectScaffold: %v", err)
	}

	for _, rel := range []string{
		"main.go",
		"cmd/server/main.go",
		"internal/server/server.go",
		"bootstrap/db.go",
		"bootstrap/logger.go",
		"bootstrap/cache.go",
		"settings/config.local.yaml.example",
		"settings/config.prod.yaml.example",
		"README.md",
		"Dockerfile",
		"docker-compose.yml",
		"internal/app/services.go",
		"internal/app/errors.go",
		"internal/app/auth.go",
		"internal/app/admin.go",
		"internal/app/permissions.go",
		"internal/app/scaffold_test.go",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}

	runScaffoldGoTest(t, outputDir)
}

func TestWriteAppScaffold(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := "module demo\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	outputDir := filepath.Join(dir, "blog")
	if err := WriteAppScaffold(AppScaffoldConfig{
		Name:        "blog",
		PackageName: "blog",
		ModelName:   "Post",
	}, outputDir); err != nil {
		t.Fatalf("WriteAppScaffold: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "routers.go"))
	if err != nil {
		t.Fatalf("read routers.go: %v", err)
	}
	if !strings.Contains(string(content), `ninja.NewRouter("/posts"`) {
		t.Fatalf("expected plural route base in routers.go\n%s", content)
	}

	runScaffoldGoTest(t, dir)
}

func TestWriteAppScaffoldStandardTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := "module demo\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	outputDir := filepath.Join(dir, "blog")
	if err := WriteAppScaffold(AppScaffoldConfig{
		Name:        "blog",
		PackageName: "blog",
		ModelName:   "Post",
		Template:    "admin",
		WithTests:   true,
	}, outputDir); err != nil {
		t.Fatalf("WriteAppScaffold: %v", err)
	}

	for _, rel := range []string{
		"services.go",
		"errors.go",
		"auth.go",
		"admin.go",
		"permissions.go",
		"scaffold_test.go",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}

	runScaffoldGoTest(t, dir)
}

func TestWriteAppScaffoldRejectsNonEmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "blog")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "existing.txt"), []byte("busy"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	err := WriteAppScaffold(AppScaffoldConfig{Name: "blog"}, outputDir)
	if err == nil || !strings.Contains(err.Error(), "already exists and is not empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteAppScaffoldForceAllowsNonEmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := "module demo\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	outputDir := filepath.Join(dir, "blog")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "existing.txt"), []byte("busy"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	if err := WriteAppScaffold(AppScaffoldConfig{
		Name:      "blog",
		Template:  "standard",
		WithTests: true,
		Force:     true,
	}, outputDir); err != nil {
		t.Fatalf("WriteAppScaffold: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "services.go")); err != nil {
		t.Fatalf("expected services.go: %v", err)
	}
}

func runScaffoldGoTest(t *testing.T, dir string) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))

	goModPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(content), "replace github.com/shijl0925/gin-ninja => ") {
		content = append(content, []byte("\nreplace github.com/shijl0925/gin-ninja => "+repoRoot+"\n")...)
		if err := os.WriteFile(goModPath, content, 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
	}

	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = dir
	if output, err := modTidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy scaffold: %v\n%s", err, output)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test scaffold: %v\n%s", err, output)
	}
}
