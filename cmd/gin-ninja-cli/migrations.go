package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
)

const (
	defaultConfigPath      = "config.yaml"
	defaultMigrationsDir   = "migrations"
	defaultMigrationAppDir = "app"
	migrationTableName     = "gin_ninja_migrations"
	migrationIrreversible  = "-- gin-ninja:irreversible"
	migrationSectionUp     = "-- Up"
	migrationSectionDown   = "-- Down"
)

var (
	errIrreversibleMigration  = errors.New("migration is irreversible")
	moduleLinePattern         = regexp.MustCompile(`(?m)^module\s+(\S+)\s*$`)
	indexPattern              = regexp.MustCompile(`(?i)^CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([` + "`\"" + `]?[^\s(` + "`\"" + `]+[` + "`\"" + `]?)\s+ON\s+([` + "`\"" + `]?[^\s(` + "`\"" + `]+[` + "`\"" + `]?)`)
	tablePattern              = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([` + "`\"" + `]?[^\s(` + "`\"" + `]+[` + "`\"" + `]?)`)
	alterAddColumnPattern     = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+([` + "`\"" + `]?[^\s` + "`\"" + `]+[` + "`\"" + `]?)\s+ADD(?:\s+COLUMN)?\s+([` + "`\"" + `]?[^\s` + "`\"" + `]+[` + "`\"" + `]?)(?:\s|$)`)
	alterAddConstraintPattern = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+([` + "`\"" + `]?[^\s` + "`\"" + `]+[` + "`\"" + `]?)\s+ADD\s+CONSTRAINT\s+([` + "`\"" + `]?[^\s` + "`\"" + `]+[` + "`\"" + `]?)(?:\s|$)`)
)

type migrationProject struct {
	rootDir       string
	configPath    string
	appDir        string
	migrationsDir string
	modulePath    string
	appImportPath string
	database      settings.DatabaseConfig
	dialect       string
}

type helperResult struct {
	Statements []string `json:"statements"`
}

type migrationFile struct {
	Version      string
	Name         string
	FileName     string
	Path         string
	RawUp        string
	RawDown      string
	Irreversible bool
}

type migrationRecord struct {
	Version   string
	Name      string
	AppliedAt time.Time
}

func runMakeMigrations(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("makemigrations", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli makemigrations [-config <path>] [-app-dir <path>] [-migrations-dir <path>] [-name <name>]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "Project config file path")
	appDir := fs.String("app-dir", "", "Relative app package directory containing MigrationModels()")
	migrationsDir := fs.String("migrations-dir", defaultMigrationsDir, "Relative migrations directory")
	name := fs.String("name", "", "Optional migration name suffix")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "makemigrations does not accept positional arguments")
		return 2
	}

	project, err := loadMigrationProject(*configPath, *appDir, *migrationsDir, true)
	if err != nil {
		fmt.Fprintf(stderr, "load migration project: %v\n", err)
		return 1
	}
	statements, err := collectMigrationStatements(project)
	if err != nil {
		fmt.Fprintf(stderr, "collect migration statements: %v\n", err)
		return 1
	}
	if len(statements) == 0 {
		fmt.Fprintln(stdout, "No changes detected")
		return 0
	}
	migrationName := strings.TrimSpace(*name)
	if migrationName == "" {
		migrationName = "auto"
	}
	fileName := newMigrationFileName(migrationName)
	upSQL := formatSQLStatements(statements)
	downStatements, reversible := reverseMigrationStatements(project.dialect, statements)
	downSQL := formatSQLStatements(downStatements)
	if !reversible {
		if downSQL != "" {
			downSQL += "\n"
		}
		downSQL += migrationIrreversible + "\n"
	}
	content := buildMigrationFile(fileName, upSQL, downSQL)
	fullPath := filepath.Join(project.rootDir, project.migrationsDir, fileName)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		fmt.Fprintf(stderr, "create migrations dir: %v\n", err)
		return 1
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "write migration file: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created migration %s\n", fullPath)
	return 0
}

func runMigrate(stdout, stderr io.Writer, args []string) int {
	targetArg, args := consumeLeadingName(args)

	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli migrate [target|zero] [-config <path>] [-migrations-dir <path>]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "Project config file path")
	migrationsDir := fs.String("migrations-dir", defaultMigrationsDir, "Relative migrations directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if targetArg == "" && fs.NArg() > 1 {
		fmt.Fprintln(stderr, "migrate accepts at most one target")
		return 2
	}
	if targetArg != "" && fs.NArg() != 0 {
		fmt.Fprintln(stderr, "migrate accepts at most one target")
		return 2
	}
	target := strings.TrimSpace(targetArg)
	if target == "" && fs.NArg() == 1 {
		target = strings.TrimSpace(fs.Arg(0))
	}

	project, err := loadMigrationProject(*configPath, "", *migrationsDir, false)
	if err != nil {
		fmt.Fprintf(stderr, "load migration project: %v\n", err)
		return 1
	}
	files, err := loadMigrationFiles(filepath.Join(project.rootDir, project.migrationsDir))
	if err != nil {
		fmt.Fprintf(stderr, "load migration files: %v\n", err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(stdout, "No migration files found")
		return 0
	}
	sqlDB, err := openMigrationDB(project)
	if err != nil {
		fmt.Fprintf(stderr, "open database: %v\n", err)
		return 1
	}
	defer sqlDB.Close()
	if err := ensureMigrationTable(sqlDB); err != nil {
		fmt.Fprintf(stderr, "ensure migration table: %v\n", err)
		return 1
	}
	applied, err := loadAppliedMigrations(sqlDB)
	if err != nil {
		fmt.Fprintf(stderr, "load applied migrations: %v\n", err)
		return 1
	}
	currentIndex, err := currentMigrationIndex(files, applied)
	if err != nil {
		fmt.Fprintf(stderr, "resolve current migration state: %v\n", err)
		return 1
	}
	targetIndex, err := resolveTargetIndex(files, target)
	if err != nil {
		fmt.Fprintf(stderr, "resolve migration target: %v\n", err)
		return 1
	}
	if target == "" {
		targetIndex = len(files) - 1
	}
	if targetIndex == currentIndex {
		fmt.Fprintln(stdout, "No migrations to apply")
		return 0
	}
	if targetIndex > currentIndex {
		for i := currentIndex + 1; i <= targetIndex; i++ {
			if err := applyMigration(sqlDB, project.dialect, files[i]); err != nil {
				fmt.Fprintf(stderr, "apply migration %s: %v\n", files[i].FileName, err)
				return 1
			}
			fmt.Fprintf(stdout, "applied %s\n", files[i].FileName)
		}
		return 0
	}
	for i := currentIndex; i > targetIndex; i-- {
		if err := rollbackMigration(sqlDB, project.dialect, files[i]); err != nil {
			fmt.Fprintf(stderr, "rollback migration %s: %v\n", files[i].FileName, err)
			return 1
		}
		fmt.Fprintf(stdout, "rolled back %s\n", files[i].FileName)
	}
	return 0
}

func runShowMigrations(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("showmigrations", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli showmigrations [-config <path>] [-migrations-dir <path>]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "Project config file path")
	migrationsDir := fs.String("migrations-dir", defaultMigrationsDir, "Relative migrations directory")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "showmigrations does not accept positional arguments")
		return 2
	}
	project, err := loadMigrationProject(*configPath, "", *migrationsDir, false)
	if err != nil {
		fmt.Fprintf(stderr, "load migration project: %v\n", err)
		return 1
	}
	files, err := loadMigrationFiles(filepath.Join(project.rootDir, project.migrationsDir))
	if err != nil {
		fmt.Fprintf(stderr, "load migration files: %v\n", err)
		return 1
	}
	sqlDB, err := openMigrationDB(project)
	if err != nil {
		fmt.Fprintf(stderr, "open database: %v\n", err)
		return 1
	}
	defer sqlDB.Close()
	if err := ensureMigrationTable(sqlDB); err != nil {
		fmt.Fprintf(stderr, "ensure migration table: %v\n", err)
		return 1
	}
	applied, err := loadAppliedMigrations(sqlDB)
	if err != nil {
		fmt.Fprintf(stderr, "load applied migrations: %v\n", err)
		return 1
	}
	for _, file := range files {
		marker := "[ ]"
		if _, ok := applied[file.Version]; ok {
			marker = "[x]"
		}
		fmt.Fprintf(stdout, "%s %s\n", marker, file.FileName)
	}
	if len(files) == 0 {
		fmt.Fprintln(stdout, "(no migrations)")
	}
	return 0
}

func runSQLMigrate(stdout, stderr io.Writer, args []string) int {
	migrationArg, args := consumeLeadingName(args)

	fs := flag.NewFlagSet("sqlmigrate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  gin-ninja-cli sqlmigrate <migration> [-config <path>] [-migrations-dir <path>] [-direction <up|down|all>]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	configPath := fs.String("config", defaultConfigPath, "Project config file path")
	migrationsDir := fs.String("migrations-dir", defaultMigrationsDir, "Relative migrations directory")
	direction := fs.String("direction", "up", "Which SQL to print: up, down, or all")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if migrationArg == "" && fs.NArg() != 1 {
		fmt.Fprintln(stderr, "sqlmigrate requires exactly one migration identifier")
		return 2
	}
	if migrationArg != "" && fs.NArg() != 0 {
		fmt.Fprintln(stderr, "sqlmigrate accepts only one migration identifier")
		return 2
	}
	target := strings.TrimSpace(migrationArg)
	if target == "" {
		target = strings.TrimSpace(fs.Arg(0))
	}
	if target == "" {
		fmt.Fprintln(stderr, "migration identifier must not be empty")
		return 2
	}
	project, err := loadMigrationProject(*configPath, "", *migrationsDir, false)
	if err != nil {
		fmt.Fprintf(stderr, "load migration project: %v\n", err)
		return 1
	}
	files, err := loadMigrationFiles(filepath.Join(project.rootDir, project.migrationsDir))
	if err != nil {
		fmt.Fprintf(stderr, "load migration files: %v\n", err)
		return 1
	}
	file, err := findMigration(files, target)
	if err != nil {
		fmt.Fprintf(stderr, "find migration: %v\n", err)
		return 1
	}
	switch strings.ToLower(strings.TrimSpace(*direction)) {
	case "up":
		fmt.Fprint(stdout, strings.TrimSpace(file.RawUp))
	case "down":
		fmt.Fprint(stdout, strings.TrimSpace(file.RawDown))
	case "all":
		fmt.Fprintf(stdout, "%s\n\n%s", strings.TrimSpace(file.RawUp), strings.TrimSpace(file.RawDown))
	default:
		fmt.Fprintln(stderr, "-direction must be one of: up, down, all")
		return 2
	}
	fmt.Fprintln(stdout)
	return 0
}

func loadMigrationProject(configPath, appDir, migrationsDir string, requireRegistry bool) (migrationProject, error) {
	absConfig, err := filepath.Abs(strings.TrimSpace(configPath))
	if err != nil {
		return migrationProject{}, fmt.Errorf("resolve config path: %w", err)
	}
	projectRoot, err := findProjectRoot(filepath.Dir(absConfig))
	if err != nil {
		return migrationProject{}, err
	}
	modulePath, err := readModulePath(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return migrationProject{}, err
	}
	cfg, err := settings.Load(absConfig)
	if err != nil {
		return migrationProject{}, err
	}
	normalizeMigrationConfigPaths(absConfig, cfg)
	resolvedAppDir := strings.TrimSpace(appDir)
	if requireRegistry {
		if resolvedAppDir == "" {
			resolvedAppDir, err = detectMigrationAppDir(projectRoot)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					resolvedAppDir = defaultMigrationAppDir
				} else {
					return migrationProject{}, err
				}
			}
		}
		resolvedAppDir = filepath.Clean(resolvedAppDir)
		if resolvedAppDir == "." || filepath.IsAbs(resolvedAppDir) {
			return migrationProject{}, fmt.Errorf("app-dir must be a relative path")
		}
		if err := ensureMigrationRegistry(projectRoot, resolvedAppDir); err != nil {
			return migrationProject{}, err
		}
	}
	resolvedMigrationsDir := strings.TrimSpace(migrationsDir)
	if resolvedMigrationsDir == "" {
		resolvedMigrationsDir = defaultMigrationsDir
	}
	resolvedMigrationsDir = filepath.Clean(resolvedMigrationsDir)
	if resolvedMigrationsDir == "." || filepath.IsAbs(resolvedMigrationsDir) {
		return migrationProject{}, fmt.Errorf("migrations-dir must be a relative path")
	}
	return migrationProject{
		rootDir:       projectRoot,
		configPath:    absConfig,
		appDir:        resolvedAppDir,
		migrationsDir: resolvedMigrationsDir,
		modulePath:    modulePath,
		appImportPath: modulePath + "/" + filepath.ToSlash(resolvedAppDir),
		database:      cfg.Database,
		dialect:       normalizeDialect(cfg.Database.Driver),
	}, nil
}

func collectMigrationStatements(project migrationProject) ([]string, error) {
	helperDir, err := os.MkdirTemp("", "gin-ninja-migration-helper-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(helperDir)
	helperPath := filepath.Join(helperDir, "main.go")
	if err := os.WriteFile(helperPath, []byte(buildMigrationHelper(project.appImportPath)), 0o644); err != nil {
		return nil, err
	}
	cmd := exec.Command("go", "run", helperPath, project.configPath)
	cmd.Dir = project.rootDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("go run helper: %s", message)
	}
	var result helperResult
	payload := bytes.TrimSpace(stdout.Bytes())
	if idx := bytes.LastIndexByte(payload, '{'); idx >= 0 {
		payload = payload[idx:]
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode helper output: %w", err)
	}
	return uniqueMigrationStatements(result.Statements), nil
}

func buildMigrationHelper(importPath string) string {
	return fmt.Sprintf(`package main

import (
"context"
"encoding/json"
"os"
"path/filepath"
"strings"
"time"

app %q
ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
"github.com/shijl0925/gin-ninja/settings"
"gorm.io/gorm"
gormlogger "gorm.io/gorm/logger"
)

type collector struct {
statements []string
}

func (c *collector) LogMode(level gormlogger.LogLevel) gormlogger.Interface { return c }
func (c *collector) Info(context.Context, string, ...interface{})            {}
func (c *collector) Warn(context.Context, string, ...interface{})            {}
func (c *collector) Error(context.Context, string, ...interface{})           {}
func (c *collector) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
sql, _ := fc()
sql = strings.TrimSpace(sql)
if sql == "" || !isDDL(sql) {
return
}
c.statements = append(c.statements, sql)
}

func isDDL(sql string) bool {
upper := strings.ToUpper(strings.TrimSpace(sql))
return strings.HasPrefix(upper, "CREATE ") || strings.HasPrefix(upper, "ALTER ") || strings.HasPrefix(upper, "DROP ")
}

func main() {
if len(os.Args) != 2 {
panic("expected config path")
}
cfgPath := os.Args[1]
cfg, err := settings.Load(cfgPath)
if err != nil {
panic(err)
}
if cfg.Database.Driver == "sqlite" || cfg.Database.Driver == "sqlite3" {
if cfg.Database.DSN != "" && !strings.HasPrefix(cfg.Database.DSN, "file:") && !filepath.IsAbs(cfg.Database.DSN) {
cfg.Database.DSN = filepath.Join(filepath.Dir(cfgPath), cfg.Database.DSN)
}
}
models := app.MigrationModels()
if len(models) == 0 {
panic("MigrationModels() returned no models")
}
db, err := ginbootstrap.InitDB(&cfg.Database)
if err != nil {
panic(err)
}
collector := &collector{}
dryRunDB := db.Session(&gorm.Session{DryRun: true, Logger: collector})
if err := dryRunDB.AutoMigrate(models...); err != nil {
panic(err)
}
if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"statements": collector.statements}); err != nil {
panic(err)
}
}
`, importPath)
}

func uniqueMigrationStatements(statements []string) []string {
	seen := make(map[string]struct{}, len(statements))
	out := make([]string, 0, len(statements))
	for _, stmt := range statements {
		normalized := normalizeSQLStatement(stmt)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeSQLStatement(stmt string) string {
	trimmed := strings.TrimSpace(stmt)
	trimmed = strings.TrimSuffix(trimmed, ";")
	trimmed = strings.TrimSpace(trimmed)
	return trimmed
}

func formatSQLStatements(statements []string) string {
	if len(statements) == 0 {
		return ""
	}
	formatted := make([]string, 0, len(statements))
	for _, stmt := range statements {
		trimmed := normalizeSQLStatement(stmt)
		if trimmed == "" {
			continue
		}
		formatted = append(formatted, trimmed+";")
	}
	return strings.Join(formatted, "\n")
}

func reverseMigrationStatements(dialect string, statements []string) ([]string, bool) {
	reversed := make([]string, 0, len(statements))
	reversible := true
	for i := len(statements) - 1; i >= 0; i-- {
		reverse, ok := reverseMigrationStatement(dialect, statements[i])
		if !ok {
			reversible = false
			continue
		}
		reversed = append(reversed, reverse)
	}
	return reversed, reversible
}

func reverseMigrationStatement(dialect, statement string) (string, bool) {
	stmt := normalizeSQLStatement(statement)
	upper := strings.ToUpper(stmt)
	if matches := tablePattern.FindStringSubmatch(stmt); len(matches) == 2 {
		return fmt.Sprintf("DROP TABLE IF EXISTS %s", matches[1]), true
	}
	if matches := indexPattern.FindStringSubmatch(stmt); len(matches) == 3 {
		if dialect == "mysql" {
			return fmt.Sprintf("DROP INDEX %s ON %s", matches[1], matches[2]), true
		}
		return fmt.Sprintf("DROP INDEX IF EXISTS %s", matches[1]), true
	}
	if matches := alterAddConstraintPattern.FindStringSubmatch(stmt); len(matches) == 3 {
		if dialect == "mysql" && strings.Contains(upper, "FOREIGN KEY") {
			return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", matches[1], matches[2]), true
		}
		return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", matches[1], matches[2]), true
	}
	if matches := alterAddColumnPattern.FindStringSubmatch(stmt); len(matches) == 3 {
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", matches[1], matches[2]), true
	}
	return "", false
}

func buildMigrationFile(fileName, upSQL, downSQL string) string {
	var buf strings.Builder
	buf.WriteString("-- gin-ninja migration: ")
	buf.WriteString(fileName)
	buf.WriteString("\n\n")
	buf.WriteString(migrationSectionUp)
	buf.WriteString("\n")
	if strings.TrimSpace(upSQL) != "" {
		buf.WriteString(strings.TrimSpace(upSQL))
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
	buf.WriteString(migrationSectionDown)
	buf.WriteString("\n")
	if strings.TrimSpace(downSQL) != "" {
		buf.WriteString(strings.TrimSpace(downSQL))
		buf.WriteString("\n")
	}
	return buf.String()
}

func newMigrationFileName(name string) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	slug := slugifyMigrationName(name)
	if slug == "" {
		slug = "auto"
	}
	return fmt.Sprintf("%s_%s.sql", timestamp, slug)
}

func slugifyMigrationName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func findProjectRoot(start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod from %s", start)
		}
		dir = parent
	}
}

func readModulePath(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	matches := moduleLinePattern.FindStringSubmatch(string(content))
	if len(matches) != 2 {
		return "", fmt.Errorf("could not determine module path from %s", goModPath)
	}
	return matches[1], nil
}

func normalizeMigrationConfigPaths(configPath string, cfg *settings.Config) {
	if cfg == nil {
		return
	}
	if cfg.Database.Driver != "sqlite" && cfg.Database.Driver != "sqlite3" {
		return
	}
	if cfg.Database.DSN == "" || strings.HasPrefix(cfg.Database.DSN, "file:") || filepath.IsAbs(cfg.Database.DSN) {
		return
	}
	cfg.Database.DSN = filepath.Join(filepath.Dir(configPath), cfg.Database.DSN)
}

func detectMigrationAppDir(projectRoot string) (string, error) {
	matches := make([]string, 0, 2)
	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "tmp" || name == defaultMigrationsDir {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "func MigrationModels(") {
			rel, err := filepath.Rel(projectRoot, filepath.Dir(path))
			if err != nil {
				return err
			}
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fs.ErrNotExist
	}
	sort.Strings(matches)
	matches = dedupeStrings(matches)
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple MigrationModels() packages found (%s); please set -app-dir", strings.Join(matches, ", "))
	}
	return matches[0], nil
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for i := 1; i < len(values); i++ {
		if values[i] != values[i-1] {
			out = append(out, values[i])
		}
	}
	return out
}

func ensureMigrationRegistry(projectRoot, appDir string) error {
	targetDir := filepath.Join(projectRoot, appDir)
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("read app-dir %s: %w", appDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(targetDir, entry.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "func MigrationModels(") {
			return nil
		}
	}
	return fmt.Errorf("%s must define func MigrationModels() []any", appDir)
}

func normalizeDialect(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite3":
		return "sqlite"
	case "postgresql":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func openMigrationDB(project migrationProject) (*sql.DB, error) {
	cfg, err := settings.Load(project.configPath)
	if err != nil {
		return nil, err
	}
	normalizeMigrationConfigPaths(project.configPath, cfg)
	db, err := ginbootstrap.InitDB(&cfg.Database)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return sqlDB, nil
}

func ensureMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + migrationTableName + ` (
version TEXT PRIMARY KEY,
name TEXT NOT NULL,
applied_at TIMESTAMP NOT NULL
)`)
	return err
}

func loadAppliedMigrations(db *sql.DB) (map[string]migrationRecord, error) {
	rows, err := db.Query(`SELECT version, name, applied_at FROM ` + migrationTableName + ` ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]migrationRecord)
	for rows.Next() {
		var record migrationRecord
		if err := rows.Scan(&record.Version, &record.Name, &record.AppliedAt); err != nil {
			return nil, err
		}
		out[record.Version] = record
	}
	return out, rows.Err()
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		parsed, err := parseMigrationFile(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, parsed)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].FileName < files[j].FileName })
	return files, nil
}

func parseMigrationFile(path string) (migrationFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return migrationFile{}, err
	}
	fileName := filepath.Base(path)
	version := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	name := version
	if idx := strings.Index(version, "_"); idx >= 0 {
		name = version[idx+1:]
		version = version[:idx]
	}
	up, down, err := splitMigrationSections(string(content))
	if err != nil {
		return migrationFile{}, fmt.Errorf("parse %s: %w", fileName, err)
	}
	return migrationFile{
		Version:      version,
		Name:         name,
		FileName:     fileName,
		Path:         path,
		RawUp:        strings.TrimSpace(up),
		RawDown:      strings.TrimSpace(down),
		Irreversible: strings.Contains(down, migrationIrreversible),
	}, nil
}

func splitMigrationSections(content string) (string, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var up, down strings.Builder
	section := ""
	for scanner.Scan() {
		line := scanner.Text()
		switch strings.TrimSpace(line) {
		case migrationSectionUp:
			section = "up"
			continue
		case migrationSectionDown:
			section = "down"
			continue
		}
		switch section {
		case "up":
			up.WriteString(line)
			up.WriteByte('\n')
		case "down":
			down.WriteString(line)
			down.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(up.String()) == "" && strings.TrimSpace(down.String()) == "" {
		return "", "", fmt.Errorf("missing %s/%s sections", migrationSectionUp, migrationSectionDown)
	}
	return up.String(), down.String(), nil
}

func splitSQLStatements(section string) []string {
	trimmed := strings.TrimSpace(section)
	if trimmed == "" {
		return nil
	}
	var (
		statements []string
		current    strings.Builder
		inSingle   bool
		inDouble   bool
		inBacktick bool
	)
	flush := func() {
		stmt := normalizeSQLStatement(current.String())
		if stmt != "" && !strings.HasPrefix(stmt, "--") {
			statements = append(statements, stmt)
		}
		current.Reset()
	}
	for i, r := range trimmed {
		switch r {
		case '\'':
			if !inDouble && !inBacktick && (i == 0 || trimmed[i-1] != '\\') {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBacktick && (i == 0 || trimmed[i-1] != '\\') {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingle && !inDouble && !inBacktick {
				flush()
				continue
			}
		}
		current.WriteRune(r)
	}
	flush()
	return statements
}

func currentMigrationIndex(files []migrationFile, applied map[string]migrationRecord) (int, error) {
	index := -1
	for i, file := range files {
		_, ok := applied[file.Version]
		if ok {
			index = i
			continue
		}
		for j := i + 1; j < len(files); j++ {
			if _, laterApplied := applied[files[j].Version]; laterApplied {
				return -1, fmt.Errorf("migration history is not linear; %s is missing while %s is applied", file.FileName, files[j].FileName)
			}
		}
		break
	}
	return index, nil
}

func resolveTargetIndex(files []migrationFile, target string) (int, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return len(files) - 1, nil
	}
	if target == "zero" {
		return -1, nil
	}
	for i, file := range files {
		if file.Version == target || strings.TrimSuffix(file.FileName, ".sql") == target || file.FileName == target || file.Name == target {
			return i, nil
		}
	}
	return -1, fmt.Errorf("migration %q not found", target)
}

func findMigration(files []migrationFile, target string) (migrationFile, error) {
	idx, err := resolveTargetIndex(files, target)
	if err != nil {
		return migrationFile{}, err
	}
	if idx < 0 {
		return migrationFile{}, fmt.Errorf("migration %q not found", target)
	}
	return files[idx], nil
}

func applyMigration(db *sql.DB, dialect string, file migrationFile) error {
	statements := splitSQLStatements(file.RawUp)
	if len(statements) == 0 {
		return recordAppliedMigration(db, dialect, file)
	}
	if err := execMigrationStatements(db, statements); err != nil {
		return err
	}
	return recordAppliedMigration(db, dialect, file)
}

func rollbackMigration(db *sql.DB, dialect string, file migrationFile) error {
	if file.Irreversible {
		return errIrreversibleMigration
	}
	statements := splitSQLStatements(file.RawDown)
	if len(statements) == 0 {
		return fmt.Errorf("migration %s has no down statements", file.FileName)
	}
	if err := execMigrationStatements(db, statements); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM `+migrationTableName+` WHERE version = `+bindVar(dialect, 1), file.Version)
	return err
}

func execMigrationStatements(db *sql.DB, statements []string) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec %q: %w", stmt, err)
		}
	}
	return tx.Commit()
}

func recordAppliedMigration(db *sql.DB, dialect string, file migrationFile) error {
	_, err := db.Exec(
		`INSERT INTO `+migrationTableName+` (version, name, applied_at) VALUES (`+bindVar(dialect, 1)+`, `+bindVar(dialect, 2)+`, `+bindVar(dialect, 3)+`)`,
		file.Version,
		file.Name,
		time.Now().UTC(),
	)
	return err
}

func bindVar(dialect string, idx int) string {
	if dialect == "postgres" {
		return fmt.Sprintf("$%d", idx)
	}
	return "?"
}
