package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGenerateCRUD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte("package demo\n\ntype User struct {\n\tID uint `json:\"id\"`\n\tName string `json:\"name\"`\n}\n"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud", "-model", "User", "-model-file", modelFile})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	outputFile := filepath.Join(dir, "user_crud_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if !strings.Contains(string(content), "func RegisterUserCRUDRoutes") {
		t.Fatalf("unexpected generated content\n%s", content)
	}
	if strings.Contains(string(content), "gormx.") {
		t.Fatalf("expected native GORM output by default\n%s", content)
	}
	if !strings.Contains(string(content), `query := db.Model(&User{})`) {
		t.Fatalf("expected native GORM query code by default\n%s", content)
	}
	if !strings.Contains(stdout.String(), outputFile) {
		t.Fatalf("stdout missing generated path: %s", stdout.String())
	}
}

func TestRunGenerateCRUDWithNativeGORM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte("package demo\n\ntype User struct {\n\tID uint `json:\"id\"`\n\tName string `json:\"name\" crud:\"filter,sort,search\"`\n}\n"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud", "-model", "User", "-model-file", modelFile, "-with-gormx=false"})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	outputFile := filepath.Join(dir, "user_crud_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if strings.Contains(string(content), "gormx.") {
		t.Fatalf("expected native GORM output\n%s", content)
	}
	if !strings.Contains(string(content), `query := db.Model(&User{})`) {
		t.Fatalf("expected native GORM query code\n%s", content)
	}
	if !strings.Contains(stdout.String(), outputFile) {
		t.Fatalf("stdout missing generated path: %s", stdout.String())
	}
}

func TestRunGenerateCRUDRequiresFlags(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud"})
	if code != 2 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "-model and -model-file are required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunStartProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"mysite",
		"-module", "github.com/acme/mysite",
		"-output", outputDir,
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	checks := []string{
		filepath.Join(outputDir, "go.mod"),
		filepath.Join(outputDir, "main.go"),
		filepath.Join(outputDir, "config.yaml"),
		filepath.Join(outputDir, "app", "models.go"),
		filepath.Join(outputDir, "app", "migrations.go"),
		filepath.Join(outputDir, "app", "repos.go"),
		filepath.Join(outputDir, "app", "schemas.go"),
		filepath.Join(outputDir, "app", "apis.go"),
		filepath.Join(outputDir, "app", "routers.go"),
	}
	for _, path := range checks {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), outputDir) {
		t.Fatalf("stdout missing scaffold path: %s", stdout.String())
	}
	content, err := os.ReadFile(filepath.Join(outputDir, "app", "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm scaffold by default, got:\n%s", content)
	}
	if !strings.Contains(string(content), "type IExampleRepo interface") {
		t.Fatalf("expected repo interface scaffold, got:\n%s", content)
	}
}

func TestRunStartProjectStandardTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"mysite",
		"-module", "github.com/acme/mysite",
		"-output", outputDir,
		"-template", "admin",
		"-app-dir", "internal/app",
		"-with-tests",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(outputDir, "cmd", "server", "main.go"),
		filepath.Join(outputDir, "internal", "server", "server.go"),
		filepath.Join(outputDir, "bootstrap", "db.go"),
		filepath.Join(outputDir, "settings", "config.local.yaml.example"),
		filepath.Join(outputDir, "internal", "app", "migrations.go"),
		filepath.Join(outputDir, "internal", "app", "services.go"),
		filepath.Join(outputDir, "internal", "app", "auth.go"),
		filepath.Join(outputDir, "internal", "app", "admin.go"),
		filepath.Join(outputDir, "internal", "app", "scaffold_test.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}

	for _, path := range []string{
		filepath.Join(outputDir, "config.yaml"),
		filepath.Join(outputDir, "settings", "config.local.yaml.example"),
		filepath.Join(outputDir, "settings", "config.prod.yaml.example"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		for _, snippet := range []string{
			`max_size_mb: 100`,
			`max_age_days: 7`,
			`max_backups: 3`,
			`compress: false`,
		} {
			if !strings.Contains(text, snippet) {
				t.Fatalf("expected %s to contain %q, got:\n%s", path, snippet, text)
			}
		}
	}
}

func TestRunStartProjectWithGormx(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "mysite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"mysite",
		"-module", "github.com/acme/mysite",
		"-output", outputDir,
		"-template", "standard",
		"-with-gormx",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "app", "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if !strings.Contains(string(content), "gormx") {
		t.Fatalf("expected gormx scaffold, got:\n%s", content)
	}
}

func TestRunStartProjectWithConfigPreset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "preset-project")
	configPath := filepath.Join(dir, "scaffold.yaml")
	if err := os.WriteFile(configPath, []byte(`
name: preset-project
module: github.com/acme/preset-project
output: `+outputDir+`
app_dir: internal/app
template: admin
with_tests: true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"startproject", "-config", configPath})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(outputDir, "internal", "app", "auth.go"),
		filepath.Join(outputDir, "internal", "app", "admin.go"),
		filepath.Join(outputDir, "internal", "app", "scaffold_test.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
}

func TestRunStartProjectConfigOverriddenByCLI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "cli-project")
	configPath := filepath.Join(dir, "scaffold.yaml")
	if err := os.WriteFile(configPath, []byte(`
name: preset-project
module: github.com/acme/preset-project
output: `+filepath.Join(dir, "preset-project")+`
template: minimal
with_tests: false
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"cli-project",
		"-config", configPath,
		"-module", "github.com/acme/cli-project",
		"-output", outputDir,
		"-template", "standard",
		"-with-tests",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(outputDir, "cmd", "server", "main.go")); err != nil {
		t.Fatalf("expected standard scaffold file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "app", "scaffold_test.go")); err != nil {
		t.Fatalf("expected tests from CLI override: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "preset-project")); !os.IsNotExist(err) {
		t.Fatalf("unexpected preset output dir created, err=%v", err)
	}
}

func TestRunStartApp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "blog")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startapp",
		"blog",
		"-output", outputDir,
		"-package", "blog",
		"-model", "Post",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, file := range []string{"models.go", "migrations.go", "repos.go", "schemas.go", "apis.go", "routers.go"} {
		path := filepath.Join(outputDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), outputDir) {
		t.Fatalf("stdout missing scaffold path: %s", stdout.String())
	}
	content, err := os.ReadFile(filepath.Join(outputDir, "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm scaffold by default, got:\n%s", content)
	}
	if !strings.Contains(string(content), "type IPostRepo interface") {
		t.Fatalf("expected repo interface scaffold, got:\n%s", content)
	}
}

func TestRunStartAppWithForceAndTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "blog")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write keep.txt: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startapp",
		"blog",
		"-output", outputDir,
		"-package", "blog",
		"-model", "Post",
		"-template", "auth",
		"-with-tests",
		"-force",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, file := range []string{"services.go", "errors.go", "auth.go", "scaffold_test.go"} {
		path := filepath.Join(outputDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
}

func TestRunStartAppWithGormx(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "blog")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startapp",
		"blog",
		"-output", outputDir,
		"-package", "blog",
		"-model", "Post",
		"-template", "standard",
		"-with-gormx",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "apis.go"))
	if err != nil {
		t.Fatalf("read apis.go: %v", err)
	}
	if !strings.Contains(string(content), "gormx") {
		t.Fatalf("expected gormx scaffold, got:\n%s", content)
	}
}

func TestRunStartAppWithConfigPreset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "accounts")
	configPath := filepath.Join(dir, "scaffold.yaml")
	if err := os.WriteFile(configPath, []byte(`
name: accounts
output: `+outputDir+`
package: accounts
model: Account
template: auth
with_tests: true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"startapp", "-config", configPath})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(outputDir, "auth.go"),
		filepath.Join(outputDir, "services.go"),
		filepath.Join(outputDir, "scaffold_test.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
}

func TestRunInitProjectWizard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "wizard-project")
	input := strings.NewReader(strings.Join([]string{
		"project",
		"wizard-project",
		"github.com/acme/wizard-project",
		outputDir,
		"internal/app",
		"standard",
		"yes",
		"no",
		"",
	}, "\n"))

	var stdout, stderr bytes.Buffer
	code := runWithInput(input, &stdout, &stderr, []string{"init"})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(outputDir, "cmd", "server", "main.go"),
		filepath.Join(outputDir, "internal", "app", "scaffold_test.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "internal", "app", "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm scaffold from wizard choice, got:\n%s", content)
	}
}

func TestRunInitAppWizard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "wizard-app")
	input := strings.NewReader(strings.Join([]string{
		"app",
		"wizard-app",
		outputDir,
		"wizardapp",
		"Widget",
		"auth",
		"yes",
		"yes",
		"",
	}, "\n"))

	var stdout, stderr bytes.Buffer
	code := runWithInput(input, &stdout, &stderr, []string{"init"})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(outputDir, "auth.go"),
		filepath.Join(outputDir, "scaffold_test.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
}
