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

func TestRunMigrationCommandsHandleMultiStepSchemaEvolution(t *testing.T) {
	t.Parallel()

	projectDir, configPath, dbPath := writeMigrationTestProject(t)
	initialID := makeMigration(t, configPath)

	stdout, stderr, code := runCommand(t, "migrate", "-config", configPath)
	if code != 0 {
		t.Fatalf("apply initial migration code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+initialID+".sql") {
		t.Fatalf("expected initial migration to apply, got %q", stdout)
	}

	insertUserRecord(t, dbPath, "alice")
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", false)
		assertTableRowCount(t, db, "users", 1)
	})

	writeMigrationModels(t, projectDir, migrationTestModelsWithEmail())
	addColumnID := makeMigration(t, configPath)

	stdout, stderr, code = runCommand(t, "sqlmigrate", addColumnID, "-config", configPath)
	if code != 0 {
		t.Fatalf("sqlmigrate add-column code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "ALTER TABLE") || !strings.Contains(stdout, "email") {
		t.Fatalf("expected add-column SQL, got %q", stdout)
	}

	stdout, stderr, code = runCommand(t, "migrate", "-config", configPath)
	if code != 0 {
		t.Fatalf("apply add-column migration code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+addColumnID+".sql") {
		t.Fatalf("expected add-column migration to apply, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableRowCount(t, db, "users", 1)
	})

	writeMigrationModels(t, projectDir, migrationTestModelsWithAuditLog())
	writeMigrationRegistry(t, projectDir, migrationTestRegistryWithAuditLog())
	addTableID := makeMigration(t, configPath)

	stdout, stderr, code = runCommand(t, "sqlmigrate", addTableID, "-config", configPath)
	if code != 0 {
		t.Fatalf("sqlmigrate add-table code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "CREATE TABLE") || !strings.Contains(stdout, "audit_logs") {
		t.Fatalf("expected add-table SQL, got %q", stdout)
	}

	stdout, stderr, code = runCommand(t, "migrate", "-config", configPath)
	if code != 0 {
		t.Fatalf("apply add-table migration code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+addTableID+".sql") {
		t.Fatalf("expected add-table migration to apply, got %q", stdout)
	}

	stdout, stderr, code = runCommand(t, "showmigrations", "-config", configPath)
	if code != 0 {
		t.Fatalf("showmigrations latest code = %d stderr=%s", code, stderr)
	}
	for _, marker := range []string{
		"[x] " + initialID + ".sql",
		"[x] " + addColumnID + ".sql",
		"[x] " + addTableID + ".sql",
	} {
		if !strings.Contains(stdout, marker) {
			t.Fatalf("expected %q in showmigrations output, got %q", marker, stdout)
		}
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableExists(t, db, "audit_logs", true)
		assertTableRowCount(t, db, "users", 1)
	})

	stdout, stderr, code = runCommand(t, "migrate", addColumnID, "-config", configPath)
	if code != 0 {
		t.Fatalf("rollback to add-column target code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "rolled back "+addTableID+".sql") {
		t.Fatalf("expected add-table rollback output, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableExists(t, db, "audit_logs", false)
		assertTableRowCount(t, db, "users", 1)
	})

	stdout, stderr, code = runCommand(t, "showmigrations", "-config", configPath)
	if code != 0 {
		t.Fatalf("showmigrations after table rollback code = %d stderr=%s", code, stderr)
	}
	for _, marker := range []string{
		"[x] " + initialID + ".sql",
		"[x] " + addColumnID + ".sql",
		"[ ] " + addTableID + ".sql",
	} {
		if !strings.Contains(stdout, marker) {
			t.Fatalf("expected %q after table rollback, got %q", marker, stdout)
		}
	}

	stdout, stderr, code = runCommand(t, "migrate", initialID, "-config", configPath)
	if code != 0 {
		t.Fatalf("rollback to initial target code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "rolled back "+addColumnID+".sql") {
		t.Fatalf("expected add-column rollback output, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", false)
		assertTableExists(t, db, "audit_logs", false)
		assertTableRowCount(t, db, "users", 1)
	})

	stdout, stderr, code = runCommand(t, "migrate", "zero", "-config", configPath)
	if code != 0 {
		t.Fatalf("rollback to zero code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "rolled back "+initialID+".sql") {
		t.Fatalf("expected initial rollback output, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", false)
		assertTableExists(t, db, "audit_logs", false)
	})

	stdout, stderr, code = runCommand(t, "migrate", addColumnID, "-config", configPath)
	if code != 0 {
		t.Fatalf("apply up to add-column target code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+initialID+".sql") || !strings.Contains(stdout, "applied "+addColumnID+".sql") {
		t.Fatalf("expected initial and add-column migrations to apply, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableExists(t, db, "audit_logs", false)
		assertTableRowCount(t, db, "users", 0)
	})

	stdout, stderr, code = runCommand(t, "migrate", addColumnID, "-config", configPath)
	if code != 0 {
		t.Fatalf("re-run migrate to same target code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "No migrations to apply") {
		t.Fatalf("expected no-op migrate output, got %q", stdout)
	}

	stdout, stderr, code = runCommand(t, "migrate", "-config", configPath)
	if code != 0 {
		t.Fatalf("apply latest from add-column target code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+addTableID+".sql") {
		t.Fatalf("expected add-table migration to reapply, got %q", stdout)
	}
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableExists(t, db, "audit_logs", true)
	})
}

func makeMigration(t *testing.T, configPath string) string {
	t.Helper()
	stdout, stderr, code := runCommand(t, "makemigrations", "-config", configPath, "-name", "init users")
	if code != 0 {
		t.Fatalf("makemigrations code = %d stderr=%s", code, stderr)
	}
	fields := strings.Fields(stdout)
	if len(fields) == 0 {
		t.Fatalf("unexpected makemigrations output %q", stdout)
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
	writeMigrationModels(t, projectDir, migrationTestBaseModels())
	writeMigrationRegistry(t, projectDir, migrationTestBaseRegistry())
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

func writeMigrationModels(t *testing.T, projectDir, models string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(projectDir, "app", "models.go"), []byte(models), 0o644); err != nil {
		t.Fatalf("write models.go: %v", err)
	}
}

func writeMigrationRegistry(t *testing.T, projectDir, registry string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(projectDir, "app", "migrations.go"), []byte(registry), 0o644); err != nil {
		t.Fatalf("write migrations.go: %v", err)
	}
}

func migrationTestBaseModels() string {
	return `package app

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name string ` + "`gorm:\"column:name;not null;uniqueIndex\" json:\"name\"`" + `
}
`
}

func migrationTestBaseRegistry() string {
	return `package app

func MigrationModels() []any {
	return []any{&User{}}
}
`
}

func migrationTestModelsWithEmail() string {
	return `package app

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string ` + "`gorm:\"column:name;not null;uniqueIndex\" json:\"name\"`" + `
	Email string ` + "`gorm:\"column:email;default:''\" json:\"email\"`" + `
}
`
}

func migrationTestModelsWithAuditLog() string {
	return `package app

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string ` + "`gorm:\"column:name;not null;uniqueIndex\" json:\"name\"`" + `
	Email string ` + "`gorm:\"column:email;default:''\" json:\"email\"`" + `
}

type AuditLog struct {
	gorm.Model
	Action string ` + "`gorm:\"column:action;not null\" json:\"action\"`" + `
}
`
}

func migrationTestRegistryWithAuditLog() string {
	return `package app

func MigrationModels() []any {
	return []any{&User{}, &AuditLog{}}
}
`
}

func runCommand(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, args)
	return stdout.String(), stderr.String(), code
}

func insertUserRecord(t *testing.T, dbPath, name string) {
	t.Helper()
	assertSQLiteState(t, dbPath, func(db *gorm.DB) {
		if err := db.Exec(`INSERT INTO users (name, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, name).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
	})
}

func assertSQLiteState(t *testing.T, dbPath string, check func(*gorm.DB)) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open raw sqlite db: %v", err)
	}
	defer sqlDB.Close()
	check(db)
}

func assertTableExists(t *testing.T, db *gorm.DB, table string, want bool) {
	t.Helper()
	if got := db.Migrator().HasTable(table); got != want {
		t.Fatalf("table %s exists = %v, want %v", table, got, want)
	}
}

func assertColumnExists(t *testing.T, db *gorm.DB, table, column string, want bool) {
	t.Helper()
	if got := db.Migrator().HasColumn(table, column); got != want {
		t.Fatalf("column %s.%s exists = %v, want %v", table, column, got, want)
	}
}

func assertTableRowCount(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("row count for %s = %d, want %d", table, count, want)
	}
}

func repoRootForTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
