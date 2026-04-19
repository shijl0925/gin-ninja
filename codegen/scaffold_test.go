package codegen

import (
	"os"
	"os/exec"
	"path"
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
		"app/migrations.go",
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
		".air.toml",
		"settings/config.local.yaml.example",
		"settings/config.prod.yaml.example",
		"README.md",
		"Dockerfile",
		"docker-compose.yml",
		"internal/app/migrations.go",
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

	adminContent, err := os.ReadFile(filepath.Join(outputDir, "internal", "app", "admin.go"))
	if err != nil {
		t.Fatalf("read internal/app/admin.go: %v", err)
	}
	if !strings.Contains(string(adminContent), "admin.MountUI(api.Engine(), admin.DefaultUIConfig())") {
		t.Fatalf("expected generated admin scaffold to mount the default admin UI, got:\n%s", adminContent)
	}

	readmeContent, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	if !strings.Contains(string(readmeContent), "`/admin`") {
		t.Fatalf("expected generated README to mention the default admin UI, got:\n%s", readmeContent)
	}

	for _, rel := range []string{
		"config.yaml",
		"settings/config.local.yaml.example",
		"settings/config.prod.yaml.example",
	} {
		content, err := os.ReadFile(filepath.Join(outputDir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(content)
		for _, snippet := range []string{
			`max_size_mb: 100`,
			`max_age_days: 7`,
			`max_backups: 3`,
			`compress: false`,
		} {
			if !strings.Contains(text, snippet) {
				t.Fatalf("expected %s to contain %q, got:\n%s", rel, snippet, text)
			}
		}
	}

	airConfig, err := os.ReadFile(filepath.Join(outputDir, ".air.toml"))
	if err != nil {
		t.Fatalf("read .air.toml: %v", err)
	}
	airText := string(airConfig)
	if !strings.Contains(airText, "go build -o ./bin/app .") ||
		!strings.Contains(airText, `bin = "./bin/app"`) ||
		!strings.Contains(airText, `root = "."`) {
		t.Fatalf("expected .air.toml to define the hot reload build, got:\n%s", airConfig)
	}

	makefile, err := os.ReadFile(filepath.Join(outputDir, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makeText := string(makefile)
	if !strings.Contains(makeText, "dev:") ||
		!strings.Contains(makeText, "install-air:") ||
		!strings.Contains(makeText, "command -v air") ||
		!strings.Contains(makeText, "make install-air") ||
		!strings.Contains(makeText, "go install github.com/air-verse/air@latest") {
		t.Fatalf("expected Makefile hot reload targets, got:\n%s", makefile)
	}
	assertGeneratedMakefileParses(t, outputDir)

	runScaffoldGoTest(t, outputDir)
}

func TestWriteProjectScaffoldWithoutGormx(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")
	if err := WriteProjectScaffold(ProjectScaffoldConfig{
		Name:      "mysite",
		Module:    "github.com/acme/mysite",
		Template:  "standard",
		WithGormx: boolPtr(false),
	}, outputDir); err != nil {
		t.Fatalf("WriteProjectScaffold: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "app", "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm repo scaffold, got:\n%s", content)
	}
	if !strings.Contains(string(content), "type IExampleRepo interface") {
		t.Fatalf("expected repo interface scaffold, got:\n%s", content)
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
	if _, err := os.Stat(filepath.Join(outputDir, "migrations.go")); err != nil {
		t.Fatalf("expected migrations.go: %v", err)
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

	adminContent, err := os.ReadFile(filepath.Join(outputDir, "admin.go"))
	if err != nil {
		t.Fatalf("read admin.go: %v", err)
	}
	if !strings.Contains(string(adminContent), "admin.MountUI(api.Engine(), admin.DefaultUIConfig())") {
		t.Fatalf("expected generated admin scaffold to mount the default admin UI, got:\n%s", adminContent)
	}

	runScaffoldGoTest(t, dir)
}

func TestWriteAppScaffoldWithoutGormx(t *testing.T) {
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
		Template:    "standard",
		WithTests:   true,
		WithGormx:   boolPtr(false),
	}, outputDir); err != nil {
		t.Fatalf("WriteAppScaffold: %v", err)
	}

	for _, rel := range []string{"repos.go", "apis.go"} {
		content, err := os.ReadFile(filepath.Join(outputDir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if strings.Contains(string(content), "gormx") {
			t.Fatalf("expected %s to avoid gormx, got:\n%s", rel, content)
		}
	}
	for _, rel := range []string{"services.go", "errors.go"} {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); !os.IsNotExist(err) {
			t.Fatalf("did not expect %s for plain standard scaffold, err=%v", rel, err)
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

	if _, err := os.Stat(filepath.Join(outputDir, "apis.go")); err != nil {
		t.Fatalf("expected apis.go: %v", err)
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

func assertGeneratedMakefileParses(t *testing.T, dir string) {
	t.Helper()

	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not available")
	}

	for _, target := range []string{"install-air", "run", "build", "test", "lint", "tidy"} {
		cmd := exec.Command("make", "-n", target)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("make -n %s in %s: %v\n%s", target, path.Base(dir), err, output)
		}
	}
}
