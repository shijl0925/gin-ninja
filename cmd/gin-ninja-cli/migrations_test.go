package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/gorm"
)

type migrationTestBackend struct {
	name           string
	driver         string
	databaseConfig func() string
}

type externalMigrationEnv struct {
	driver    string
	dsn       string
	host      string
	port      int
	user      string
	password  string
	database  string
	charset   string
	parseTime bool
	loc       string
	sslMode   string
	timeZone  string
}

func TestRunMigrationCommands(t *testing.T) {
	t.Parallel()

	_, configPath := writeMigrationTestProject(t, sqliteMigrationTestBackend())
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

	assertDatabaseState(t, configPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
	})

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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", false)
	})
}

func TestRunMakeMigrations(t *testing.T) {
	t.Parallel()

	_, configPath := writeMigrationTestProject(t, sqliteMigrationTestBackend())
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
	runMigrationSchemaEvolutionScenario(t, sqliteMigrationTestBackend())
}

func TestRunMigrationCommandsHandleMultiStepSchemaEvolutionMySQL(t *testing.T) {
	requireIntegration(t)
	runMigrationSchemaEvolutionScenario(t, mysqlMigrationTestBackend(t))
}

func TestRunMigrationCommandsHandleMultiStepSchemaEvolutionPostgres(t *testing.T) {
	requireIntegration(t)
	runMigrationSchemaEvolutionScenario(t, postgresMigrationTestBackend(t))
}

func TestMigrationDialectSpecificHelpers(t *testing.T) {
	t.Parallel()

	if got := normalizeDialect("postgresql"); got != "postgres" {
		t.Fatalf("normalizeDialect(postgresql) = %q, want postgres", got)
	}
	if got := normalizeDialect("sqlite3"); got != "sqlite" {
		t.Fatalf("normalizeDialect(sqlite3) = %q, want sqlite", got)
	}
	if got := bindVar("postgres", 2); got != "$2" {
		t.Fatalf("bindVar(postgres, 2) = %q, want $2", got)
	}
	if got := bindVar("mysql", 2); got != "?" {
		t.Fatalf("bindVar(mysql, 2) = %q, want ?", got)
	}

	stmt, ok := reverseMigrationStatement("mysql", "CREATE INDEX `idx_users_email` ON `users` (`email`)")
	if !ok {
		t.Fatal("expected mysql index statement to be reversible")
	}
	if stmt != "DROP INDEX `idx_users_email` ON `users`" {
		t.Fatalf("mysql reverse index = %q", stmt)
	}

	stmt, ok = reverseMigrationStatement("mysql", "ALTER TABLE `audit_logs` ADD CONSTRAINT `fk_audit_logs_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)")
	if !ok {
		t.Fatal("expected mysql foreign key statement to be reversible")
	}
	if stmt != "ALTER TABLE `audit_logs` DROP FOREIGN KEY `fk_audit_logs_user`" {
		t.Fatalf("mysql reverse foreign key = %q", stmt)
	}

	stmt, ok = reverseMigrationStatement("postgres", "CREATE INDEX \"idx_users_email\" ON \"users\" (\"email\")")
	if !ok {
		t.Fatal("expected postgres index statement to be reversible")
	}
	if stmt != "DROP INDEX IF EXISTS \"idx_users_email\"" {
		t.Fatalf("postgres reverse index = %q", stmt)
	}
}

func TestRunMigrationCommandFailurePaths(t *testing.T) {
	t.Parallel()

	t.Run("invalid target", func(t *testing.T) {
		t.Parallel()

		_, configPath := writeMigrationTestProject(t, sqliteMigrationTestBackend())
		makeMigration(t, configPath)

		_, stderr, code := runCommand(t, "migrate", "missing-version", "-config", configPath)
		if code != 1 {
			t.Fatalf("migrate missing target code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, `resolve migration target: migration "missing-version" not found`) {
			t.Fatalf("expected missing target error, got %q", stderr)
		}
	})

	t.Run("irreversible rollback", func(t *testing.T) {
		t.Parallel()

		projectDir, configPath := writeMigrationTestProject(t, sqliteMigrationTestBackend())
		migrationID := "20260101010101_irreversible"
		content := buildMigrationFile(migrationID+".sql", "CREATE TABLE `users` (`id` integer primary key);", migrationIrreversible+"\n")
		if err := os.WriteFile(filepath.Join(projectDir, "migrations", migrationID+".sql"), []byte(content), 0o644); err != nil {
			t.Fatalf("write irreversible migration: %v", err)
		}

		stdout, stderr, code := runCommand(t, "migrate", "-config", configPath)
		if code != 0 {
			t.Fatalf("apply irreversible migration code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stdout, "applied "+migrationID+".sql") {
			t.Fatalf("expected irreversible migration to apply, got %q", stdout)
		}

		_, stderr, code = runCommand(t, "migrate", "zero", "-config", configPath)
		if code != 1 {
			t.Fatalf("rollback irreversible migration code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, "rollback migration "+migrationID+".sql: migration is irreversible") {
			t.Fatalf("expected irreversible rollback error, got %q", stderr)
		}
	})

	t.Run("migration table creation failure", func(t *testing.T) {
		t.Parallel()

		projectDir, configPath := writeMigrationTestProject(t, sqliteMigrationTestBackend())
		dbPath := filepath.Join(projectDir, "app.db")
		if err := os.WriteFile(dbPath, nil, 0o444); err != nil {
			t.Fatalf("seed readonly sqlite db: %v", err)
		}

		_, stderr, code := runCommand(t, "showmigrations", "-config", configPath)
		if code != 1 {
			t.Fatalf("showmigrations with conflicting view code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, "ensure migration table") {
			t.Fatalf("expected ensure migration table error, got %q", stderr)
		}
	})

	t.Run("mysql connection failure", func(t *testing.T) {
		t.Parallel()

		_, configPath := writeMigrationTestProject(t, migrationTestBackend{
			name:   "mysql-bad-connection",
			driver: "mysql",
			databaseConfig: func() string {
				return strings.TrimSpace(`database:
  driver: "mysql"
  dsn: "root:secret@tcp(127.0.0.1:1)/gin_ninja?parseTime=true"
`)
			},
		})

		_, stderr, code := runCommand(t, "showmigrations", "-config", configPath)
		if code != 1 {
			t.Fatalf("mysql showmigrations bad connection code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, "open database") {
			t.Fatalf("expected open database error, got %q", stderr)
		}
	})

	t.Run("postgres connection failure", func(t *testing.T) {
		t.Parallel()

		_, configPath := writeMigrationTestProject(t, migrationTestBackend{
			name:   "postgres-bad-connection",
			driver: "postgresql",
			databaseConfig: func() string {
				return strings.TrimSpace(`database:
  driver: "postgresql"
  dsn: "host=127.0.0.1 port=1 user=postgres dbname=gin_ninja sslmode=disable connect_timeout=1"
`)
			},
		})

		_, stderr, code := runCommand(t, "showmigrations", "-config", configPath)
		if code != 1 {
			t.Fatalf("postgres showmigrations bad connection code = %d stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, "open database") {
			t.Fatalf("expected open database error, got %q", stderr)
		}
	})
}

func runMigrationSchemaEvolutionScenario(t *testing.T, backend migrationTestBackend) {
	projectDir, configPath := writeMigrationTestProject(t, backend)
	resetMigrationTestDatabase(t, configPath)
	t.Cleanup(func() {
		resetMigrationTestDatabase(t, configPath)
	})

	initialID := makeMigration(t, configPath)

	stdout, stderr, code := runCommand(t, "migrate", "-config", configPath)
	if code != 0 {
		t.Fatalf("apply initial migration code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "applied "+initialID+".sql") {
		t.Fatalf("expected initial migration to apply, got %q", stdout)
	}

	insertUserRecord(t, configPath, "alice")
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
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
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
		assertTableExists(t, db, "users", true)
		assertColumnExists(t, db, "users", "email", true)
		assertTableExists(t, db, "audit_logs", true)
	})

	if backend.driver == "postgres" {
		project, err := loadMigrationProject(configPath, "", defaultMigrationsDir, false)
		if err != nil {
			t.Fatalf("load postgres migration project: %v", err)
		}
		if project.dialect != "postgres" {
			t.Fatalf("project.dialect = %q, want postgres", project.dialect)
		}
	}
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

func writeMigrationTestProject(t *testing.T, backend migrationTestBackend) (string, string) {
	t.Helper()
	projectDir := t.TempDir()
	repoRoot := repoRootForTests(t)
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
	configPath := filepath.Join(projectDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(buildMigrationTestConfig(backend)), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}
	return projectDir, configPath
}

func buildMigrationTestConfig(backend migrationTestBackend) string {
	return strings.Join([]string{
		"app:",
		"  name: \"Migration Test\"",
		"server:",
		"  port: 8080",
		backend.databaseConfig(),
		"log:",
		"  level: \"info\"",
		"  format: \"console\"",
		"  output: \"stdout\"",
		"",
	}, "\n")
}

func sqliteMigrationTestBackend() migrationTestBackend {
	return migrationTestBackend{
		name:   "sqlite",
		driver: "sqlite",
		databaseConfig: func() string {
			return strings.TrimSpace(`database:
  driver: "sqlite"
  dsn: "app.db"
`)
		},
	}
}

func mysqlMigrationTestBackend(t *testing.T) migrationTestBackend {
	t.Helper()
	cfg, ok := loadExternalMigrationEnv("GIN_NINJA_TEST_MYSQL")
	if !ok {
		t.Skip("set GIN_NINJA_TEST_MYSQL_DSN or GIN_NINJA_TEST_MYSQL_HOST/GIN_NINJA_TEST_MYSQL_DB to run MySQL migration integration tests")
	}
	cfg.driver = "mysql"
	if cfg.charset == "" {
		cfg.charset = "utf8mb4"
	}
	if cfg.loc == "" {
		cfg.loc = "UTC"
	}
	cfg.parseTime = true
	return migrationTestBackend{
		name:   "mysql",
		driver: "mysql",
		databaseConfig: func() string {
			return renderMySQLDatabaseConfig(cfg)
		},
	}
}

func postgresMigrationTestBackend(t *testing.T) migrationTestBackend {
	t.Helper()
	cfg, ok := loadExternalMigrationEnv("GIN_NINJA_TEST_POSTGRES")
	if !ok {
		t.Skip("set GIN_NINJA_TEST_POSTGRES_DSN or GIN_NINJA_TEST_POSTGRES_HOST/GIN_NINJA_TEST_POSTGRES_DB to run PostgreSQL migration integration tests")
	}
	if strings.TrimSpace(cfg.driver) == "" {
		cfg.driver = "postgresql"
	}
	if cfg.sslMode == "" {
		cfg.sslMode = "disable"
	}
	if cfg.timeZone == "" {
		cfg.timeZone = "UTC"
	}
	return migrationTestBackend{
		name:   "postgres",
		driver: "postgres",
		databaseConfig: func() string {
			return renderPostgresDatabaseConfig(cfg)
		},
	}
}

func loadExternalMigrationEnv(prefix string) (externalMigrationEnv, bool) {
	cfg := externalMigrationEnv{
		driver:   lookupEnvTrim(prefix + "_DRIVER"),
		dsn:      lookupEnvTrim(prefix + "_DSN"),
		host:     lookupEnvTrim(prefix + "_HOST"),
		port:     lookupEnvInt(prefix+"_PORT", 0),
		user:     lookupEnvTrim(prefix + "_USER"),
		password: os.Getenv(prefix + "_PASSWORD"),
		database: lookupEnvTrim(prefix + "_DB"),
		charset:  lookupEnvTrim(prefix + "_CHARSET"),
		loc:      lookupEnvTrim(prefix + "_LOC"),
		sslMode:  lookupEnvTrim(prefix + "_SSLMODE"),
		timeZone: lookupEnvTrim(prefix + "_TIME_ZONE"),
	}
	if cfg.dsn != "" {
		return cfg, true
	}
	if cfg.host != "" && cfg.database != "" {
		return cfg, true
	}
	return externalMigrationEnv{}, false
}

func renderMySQLDatabaseConfig(cfg externalMigrationEnv) string {
	if cfg.dsn != "" {
		return strings.TrimSpace(fmt.Sprintf("database:\n  driver: %s\n  dsn: %s\n", yamlString(cfg.driver), yamlString(cfg.dsn)))
	}
	port := cfg.port
	if port == 0 {
		port = 3306
	}
	return strings.TrimSpace(fmt.Sprintf(`database:
  driver: %s
  mysql:
    host: %s
    port: %d
    user: %s
    password: %s
    name: %s
    charset: %s
    parse_time: true
    loc: %s
`, yamlString(cfg.driver), yamlString(cfg.host), port, yamlString(cfg.user), yamlString(cfg.password), yamlString(cfg.database), yamlString(cfg.charset), yamlString(cfg.loc)))
}

func renderPostgresDatabaseConfig(cfg externalMigrationEnv) string {
	if cfg.dsn != "" {
		return strings.TrimSpace(fmt.Sprintf("database:\n  driver: %s\n  dsn: %s\n", yamlString(cfg.driver), yamlString(cfg.dsn)))
	}
	port := cfg.port
	if port == 0 {
		port = 5432
	}
	return strings.TrimSpace(fmt.Sprintf(`database:
  driver: %s
  postgres:
    host: %s
    port: %d
    user: %s
    password: %s
    name: %s
    sslmode: %s
    time_zone: %s
`, yamlString(cfg.driver), yamlString(cfg.host), port, yamlString(cfg.user), yamlString(cfg.password), yamlString(cfg.database), yamlString(cfg.sslMode), yamlString(cfg.timeZone)))
}

func yamlString(value string) string {
	return strconv.Quote(value)
}

func lookupEnvTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func lookupEnvInt(key string, fallback int) int {
	value := lookupEnvTrim(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
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

func insertUserRecord(t *testing.T, configPath, name string) {
	t.Helper()
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
		if err := db.Exec(`INSERT INTO users (name, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, name).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
	})
}

func assertDatabaseState(t *testing.T, configPath string, check func(*gorm.DB)) {
	t.Helper()
	cfg, err := settings.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	normalizeMigrationConfigPaths(configPath, cfg)
	db, err := ginbootstrap.InitDB(&cfg.Database)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer sqlDB.Close()
	check(db)
}

func resetMigrationTestDatabase(t *testing.T, configPath string) {
	t.Helper()
	assertDatabaseState(t, configPath, func(db *gorm.DB) {
		if err := db.Migrator().DropTable("audit_logs", "users", migrationTableName); err != nil {
			t.Fatalf("drop migration test tables: %v", err)
		}
	})
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
