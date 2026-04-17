package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunMigrationCommands(t *testing.T) {
	t.Parallel()

	_, configPath, dbPath := writeMigrationTestProject(t)
	migrationID := makeMigration(t, configPath)

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"showmigrations", "-config", configPath})
	if code != 0 {
		t.Fatalf("showmigrations code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[ ] "+migrationID+".sql") {
		t.Fatalf("expected pending migration, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(&stdout, &stderr, []string{"sqlmigrate", migrationID, "-config", configPath})
	if code != 0 {
		t.Fatalf("sqlmigrate code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CREATE TABLE") {
		t.Fatalf("expected CREATE TABLE in sqlmigrate output, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(&stdout, &stderr, []string{"migrate", "-config", configPath})
	if code != 0 {
		t.Fatalf("migrate code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "applied "+migrationID+".sql") {
		t.Fatalf("expected applied migration output, got %q", stdout.String())
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if !db.Migrator().HasTable("users") {
		t.Fatal("expected users table after migrate")
	}

	stdout.Reset()
	stderr.Reset()
	code = run(&stdout, &stderr, []string{"showmigrations", "-config", configPath})
	if code != 0 {
		t.Fatalf("showmigrations after migrate code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[x] "+migrationID+".sql") {
		t.Fatalf("expected applied migration, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(&stdout, &stderr, []string{"makemigrations", "-config", configPath})
	if code != 0 {
		t.Fatalf("makemigrations after migrate code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No changes detected") {
		t.Fatalf("expected no changes output, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(&stdout, &stderr, []string{"migrate", "zero", "-config", configPath})
	if code != 0 {
		t.Fatalf("migrate zero code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rolled back "+migrationID+".sql") {
		t.Fatalf("expected rollback output, got %q", stdout.String())
	}
	if db.Migrator().HasTable("users") {
		t.Fatal("expected users table to be removed after rollback")
	}
}

func TestRunMakeMigrations(t *testing.T) {
	t.Parallel()

	_, configPath, _ := writeMigrationTestProject(t)
	migrationID := makeMigration(t, configPath)
	migrationFile := filepath.Join(filepath.Dir(configPath), "migrations", migrationID+".sql")
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, migrationSectionUp) || !strings.Contains(text, migrationSectionDown) {
		t.Fatalf("expected up/down sections, got %s", text)
	}
	if !strings.Contains(text, "CREATE TABLE `users`") {
		t.Fatalf("expected CREATE TABLE statement, got %s", text)
	}
	if !strings.Contains(text, "DROP TABLE IF EXISTS `users`") {
		t.Fatalf("expected DROP TABLE statement, got %s", text)
	}
}

func makeMigration(t *testing.T, configPath string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"makemigrations", "-config", configPath, "-name", "init users"})
	if code != 0 {
		t.Fatalf("makemigrations code = %d stderr=%s", code, stderr.String())
	}
	fields := strings.Fields(stdout.String())
	if len(fields) == 0 {
		t.Fatalf("unexpected makemigrations output %q", stdout.String())
	}
	createdPath := fields[len(fields)-1]
	return strings.TrimSuffix(filepath.Base(createdPath), ".sql")
}

func writeMigrationTestProject(t *testing.T) (string, string, string) {
	t.Helper()
	projectDir := t.TempDir()
	repoRoot := repoRootForTests(t)
	dbPath := filepath.Join(projectDir, "app.db")
	goMod := "module example.com/migrationtest\n\ngo 1.26\n\nrequire github.com/shijl0925/gin-ninja v0.0.0\n\nreplace github.com/shijl0925/gin-ninja => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app"), 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "migrations"), 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	keepDeps := `package migrationtest

import (
	_ "github.com/shijl0925/gin-ninja/bootstrap"
	_ "github.com/shijl0925/gin-ninja/settings"
)
`
	if err := os.WriteFile(filepath.Join(projectDir, "deps.go"), []byte(keepDeps), 0o644); err != nil {
		t.Fatalf("write deps.go: %v", err)
	}
	models := `package app

import "gorm.io/gorm"

type User struct {
gorm.Model
Name string ` + "`gorm:\"column:name;not null;uniqueIndex\" json:\"name\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "models.go"), []byte(models), 0o644); err != nil {
		t.Fatalf("write models.go: %v", err)
	}
	migrations := `package app

func MigrationModels() []any {
return []any{&User{}}
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "migrations.go"), []byte(migrations), 0o644); err != nil {
		t.Fatalf("write migrations.go: %v", err)
	}
	config := "app:\n  name: \"Migration Test\"\nserver:\n  port: 8080\ndatabase:\n  driver: \"sqlite\"\n  dsn: \"app.db\"\nlog:\n  level: \"info\"\n  format: \"console\"\n  output: \"stdout\"\n"
	configPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}
	return projectDir, configPath, dbPath
}

func repoRootForTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
