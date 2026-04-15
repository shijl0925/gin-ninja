package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/jinzhu/inflection"
)

type ScaffoldTemplate string

const (
	ScaffoldTemplateMinimal  ScaffoldTemplate = "minimal"
	ScaffoldTemplateStandard ScaffoldTemplate = "standard"
	ScaffoldTemplateAuth     ScaffoldTemplate = "auth"
	ScaffoldTemplateAdmin    ScaffoldTemplate = "admin"
)

// ProjectScaffoldConfig defines the inputs for a new project scaffold.
type ProjectScaffoldConfig struct {
	Name      string
	Module    string
	AppDir    string
	Template  string
	WithTests bool
	WithAuth  bool
	WithAdmin bool
	Force     bool
}

// AppScaffoldConfig defines the inputs for a new app scaffold.
type AppScaffoldConfig struct {
	Name        string
	PackageName string
	ModelName   string
	Template    string
	WithTests   bool
	WithAuth    bool
	WithAdmin   bool
	Force       bool
}

type scaffoldOptions struct {
	Template  ScaffoldTemplate
	WithTests bool
	WithAuth  bool
	WithAdmin bool
	Standard  bool
}

// WriteProjectScaffold creates a new project scaffold in outputDir.
func WriteProjectScaffold(cfg ProjectScaffoldConfig, outputDir string) error {
	module := strings.TrimSpace(cfg.Module)
	if module == "" {
		return fmt.Errorf("module is required")
	}
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("output directory is required")
	}

	opts, err := resolveScaffoldOptions(cfg.Template, cfg.WithTests, cfg.WithAuth, cfg.WithAdmin)
	if err != nil {
		return err
	}
	if err := ensureScaffoldDir(outputDir, cfg.Force); err != nil {
		return err
	}

	projectName := strings.TrimSpace(cfg.Name)
	if projectName == "" {
		projectName = filepath.Base(outputDir)
	}
	words := scaffoldSplitWords(projectName)
	if len(words) == 0 {
		words = scaffoldSplitWords(filepath.Base(module))
	}
	if len(words) == 0 {
		words = []string{"app"}
	}

	appDir, err := normalizeScaffoldSubdir(cfg.AppDir, "app")
	if err != nil {
		return err
	}
	appPackage := scaffoldNormalizePackageName(filepath.Base(appDir))
	appData, err := buildAppTemplateData(AppScaffoldConfig{
		Name:        "example",
		PackageName: appPackage,
		ModelName:   "Example",
	}, opts)
	if err != nil {
		return err
	}

	data := projectTemplateData{
		Module:        module,
		AppName:       scaffoldJoinTitle(words),
		DatabaseFile:  scaffoldToSeparated(words, "_", true) + ".db",
		AppDir:        filepath.ToSlash(appDir),
		AppImportPath: module + "/" + filepath.ToSlash(appDir),
		App:           appData,
		Options:       opts,
	}
	if data.DatabaseFile == ".db" {
		data.DatabaseFile = "app.db"
	}

	files, err := projectFiles(data)
	if err != nil {
		return err
	}
	return writeScaffoldFiles(outputDir, files)
}

// WriteAppScaffold creates a new app scaffold in outputDir.
func WriteAppScaffold(cfg AppScaffoldConfig, outputDir string) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("output directory is required")
	}
	opts, err := resolveScaffoldOptions(cfg.Template, cfg.WithTests, cfg.WithAuth, cfg.WithAdmin)
	if err != nil {
		return err
	}
	if err := ensureScaffoldDir(outputDir, cfg.Force); err != nil {
		return err
	}
	data, err := buildAppTemplateData(cfg, opts)
	if err != nil {
		return err
	}
	files, err := appFiles(data)
	if err != nil {
		return err
	}
	return writeScaffoldFiles(outputDir, files)
}

type projectTemplateData struct {
	Module        string
	AppName       string
	DatabaseFile  string
	AppDir        string
	AppImportPath string
	App           appTemplateData
	Options       scaffoldOptions
}

type appTemplateData struct {
	PackageName string
	ModelName   string
	ModelPlural string
	ModelLower  string
	RepoName    string
	ServiceName string
	OutName     string
	ListName    string
	GetName     string
	CreateName  string
	UpdateName  string
	DeleteName  string
	RouteBase   string
	RouteTag    string
	Options     scaffoldOptions
}

func resolveScaffoldOptions(templateName string, withTests, withAuth, withAdmin bool) (scaffoldOptions, error) {
	templateName = strings.ToLower(strings.TrimSpace(templateName))
	if templateName == "" {
		templateName = string(ScaffoldTemplateMinimal)
	}

	templateKind := ScaffoldTemplate(templateName)
	switch templateKind {
	case ScaffoldTemplateMinimal, ScaffoldTemplateStandard, ScaffoldTemplateAuth, ScaffoldTemplateAdmin:
	default:
		return scaffoldOptions{}, fmt.Errorf("unknown scaffold template %q", templateName)
	}

	opts := scaffoldOptions{
		Template:  templateKind,
		WithTests: withTests,
		WithAuth:  withAuth,
		WithAdmin: withAdmin,
	}
	if templateKind == ScaffoldTemplateAuth {
		opts.WithAuth = true
	}
	if templateKind == ScaffoldTemplateAdmin {
		opts.WithAuth = true
		opts.WithAdmin = true
	}
	opts.Standard = templateKind != ScaffoldTemplateMinimal || opts.WithTests || opts.WithAuth || opts.WithAdmin
	return opts, nil
}

func normalizeScaffoldSubdir(value, fallback string) (string, error) {
	dir := strings.TrimSpace(value)
	if dir == "" {
		dir = fallback
	}
	dir = filepath.Clean(dir)
	if dir == "." || dir == "" {
		return "", fmt.Errorf("scaffold subdirectory must not be empty")
	}
	if filepath.IsAbs(dir) {
		return "", fmt.Errorf("scaffold subdirectory must be relative")
	}
	if dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("scaffold subdirectory must stay within the project root")
	}
	return dir, nil
}

func buildAppTemplateData(cfg AppScaffoldConfig, opts scaffoldOptions) (appTemplateData, error) {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return appTemplateData{}, fmt.Errorf("name is required")
	}
	words := scaffoldSplitWords(name)
	if len(words) == 0 {
		return appTemplateData{}, fmt.Errorf("name %q does not contain any valid letters or digits", name)
	}

	packageName := strings.TrimSpace(cfg.PackageName)
	if packageName == "" {
		packageName = scaffoldToSeparated(words, "_", true)
	}
	packageName = scaffoldNormalizePackageName(packageName)

	modelName := strings.TrimSpace(cfg.ModelName)
	if modelName == "" {
		modelName = scaffoldToExported(words)
	}
	modelName = scaffoldNormalizeExportedName(modelName)

	modelPlural := inflection.Plural(modelName)
	pluralWords := scaffoldSplitWords(modelPlural)
	if len(pluralWords) == 0 {
		pluralWords = append([]string(nil), words...)
		pluralWords = append(pluralWords, "items")
	}

	return appTemplateData{
		PackageName: packageName,
		ModelName:   modelName,
		ModelPlural: modelPlural,
		ModelLower:  strings.ToLower(modelName),
		RepoName:    modelName + "Repo",
		ServiceName: modelName + "Service",
		OutName:     modelName + "Out",
		ListName:    "List" + modelPlural + "Input",
		GetName:     "Get" + modelName + "Input",
		CreateName:  "Create" + modelName + "Input",
		UpdateName:  "Update" + modelName + "Input",
		DeleteName:  "Delete" + modelName + "Input",
		RouteBase:   scaffoldToSeparated(pluralWords, "-", true),
		RouteTag:    scaffoldJoinTitle(pluralWords),
		Options:     opts,
	}, nil
}

func projectFiles(data projectTemplateData) (map[string][]byte, error) {
	files := map[string][]byte{
		"go.mod":      []byte(fmt.Sprintf("module %s\n\ngo 1.26\n", data.Module)),
		"config.yaml": []byte(executeTextTemplate(projectConfigTemplate, data)),
		".gitignore":  []byte(executeTextTemplate(projectGitignoreTemplate, data)),
	}

	if data.Options.Standard {
		mainGo, err := executeGoTemplate("main.go", projectMainWrapperTemplate, data)
		if err != nil {
			return nil, err
		}
		files["main.go"] = mainGo

		serverGo, err := executeGoTemplate("internal/server/server.go", projectInternalServerTemplate, data)
		if err != nil {
			return nil, err
		}
		files[filepath.Join("internal", "server", "server.go")] = serverGo

		cmdServerGo, err := executeGoTemplate("cmd/server/main.go", projectCmdServerTemplate, data)
		if err != nil {
			return nil, err
		}
		files[filepath.Join("cmd", "server", "main.go")] = cmdServerGo

		for name, tpl := range map[string]string{
			filepath.Join("bootstrap", "db.go"):                    projectBootstrapDBTemplate,
			filepath.Join("bootstrap", "logger.go"):                projectBootstrapLoggerTemplate,
			filepath.Join("bootstrap", "cache.go"):                 projectBootstrapCacheTemplate,
			filepath.Join("settings", "config.local.yaml.example"): projectConfigLocalTemplate,
			filepath.Join("settings", "config.prod.yaml.example"):  projectConfigProdTemplate,
			"README.md":                             projectREADMETemplate,
			".env.example":                          projectEnvTemplate,
			"Makefile":                              projectMakefileTemplate,
			"Dockerfile":                            projectDockerfileTemplate,
			"docker-compose.yml":                    projectDockerComposeTemplate,
			filepath.Join("migrations", ".gitkeep"): "",
			filepath.Join("scripts", ".gitkeep"):    "",
		} {
			if strings.HasSuffix(name, ".go") {
				content, err := executeGoTemplate(name, tpl, data)
				if err != nil {
					return nil, err
				}
				files[name] = content
				continue
			}
			files[name] = []byte(executeTextTemplate(tpl, data))
		}
	} else {
		mainGo, err := executeGoTemplate("main.go", projectMainTemplate, data)
		if err != nil {
			return nil, err
		}
		files["main.go"] = mainGo
	}

	appFiles, err := appFiles(data.App)
	if err != nil {
		return nil, err
	}
	for name, content := range appFiles {
		files[filepath.Join(filepath.FromSlash(data.AppDir), name)] = content
	}
	return files, nil
}

func appFiles(data appTemplateData) (map[string][]byte, error) {
	templates := map[string]string{
		"models.go":  appModelsTemplate,
		"repos.go":   appReposTemplate,
		"schemas.go": appSchemasTemplate,
		"routers.go": appRoutersTemplate,
	}
	if data.Options.Standard {
		templates["services.go"] = appServicesTemplate
		templates["errors.go"] = appErrorsTemplate
		templates["apis.go"] = appAPIsWithServicesTemplate
	} else {
		templates["apis.go"] = appAPIsTemplate
	}
	if data.Options.WithAuth {
		templates["auth.go"] = appAuthTemplate
	}
	if data.Options.WithAdmin {
		templates["admin.go"] = appAdminTemplate
		templates["permissions.go"] = appPermissionsTemplate
	}
	if data.Options.WithTests {
		templates["scaffold_test.go"] = appTestsTemplate
	}

	files := make(map[string][]byte, len(templates))
	for name, tpl := range templates {
		content, err := executeGoTemplate(name, tpl, data)
		if err != nil {
			return nil, err
		}
		files[name] = content
	}
	return files, nil
}

func ensureScaffoldDir(dir string, force bool) error {
	info, err := os.Stat(dir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("%s already exists and is not a directory", dir)
		}
		if !force {
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				return fmt.Errorf("read output directory: %w", readErr)
			}
			if len(entries) > 0 {
				return fmt.Errorf("%s already exists and is not empty", dir)
			}
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	default:
		return fmt.Errorf("stat output directory: %w", err)
	}
	return nil
}

func writeScaffoldFiles(root string, files map[string][]byte) error {
	for rel, content := range files {
		fullPath := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}

func executeGoTemplate(name, source string, data any) ([]byte, error) {
	rendered := executeTextTemplate(source, data)
	formatted, err := format.Source([]byte(rendered))
	if err != nil {
		return nil, fmt.Errorf("format %s: %w", name, err)
	}
	return formatted, nil
}

func executeTextTemplate(source string, data any) string {
	tpl := template.Must(template.New("scaffold").Funcs(template.FuncMap{
		"bt":    func() string { return "`" },
		"lower": strings.ToLower,
	}).Parse(source))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func scaffoldSplitWords(input string) []string {
	var words []string
	var current []rune
	var prev rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, string(current))
		current = current[:0]
	}

	for i, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if len(current) > 0 && shouldSplitWord(prev, r, peekRune(input, i)) {
				flush()
			}
			current = append(current, r)
			prev = r
			continue
		}
		flush()
		prev = 0
	}
	flush()
	return words
}

func shouldSplitWord(prev, curr, next rune) bool {
	if prev == 0 {
		return false
	}
	if unicode.IsLower(prev) && unicode.IsUpper(curr) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsDigit(curr) {
		return true
	}
	if unicode.IsDigit(prev) && unicode.IsLetter(curr) {
		return true
	}
	if unicode.IsUpper(prev) && unicode.IsUpper(curr) && unicode.IsLower(next) {
		return true
	}
	return false
}

func peekRune(input string, index int) rune {
	seen := false
	for i, r := range input {
		if !seen {
			if i == index {
				seen = true
			}
			continue
		}
		return r
	}
	return 0
}

func scaffoldToSeparated(words []string, sep string, lower bool) string {
	if len(words) == 0 {
		return ""
	}
	out := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" {
			continue
		}
		if lower {
			out = append(out, strings.ToLower(word))
		} else {
			out = append(out, word)
		}
	}
	return strings.Join(out, sep)
}

func scaffoldJoinTitle(words []string) string {
	if len(words) == 0 {
		return ""
	}
	out := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" {
			continue
		}
		out = append(out, scaffoldCapitalizeFirst(word))
	}
	return strings.Join(out, " ")
}

func scaffoldToExported(words []string) string {
	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		b.WriteString(scaffoldCapitalizeFirst(word))
	}
	name := b.String()
	if name == "" {
		return "App"
	}
	if first := rune(name[0]); !unicode.IsLetter(first) {
		return "App" + name
	}
	return name
}

func scaffoldNormalizePackageName(name string) string {
	words := scaffoldSplitWords(name)
	normalized := scaffoldToSeparated(words, "_", true)
	if normalized == "" {
		return "app"
	}
	if first := rune(normalized[0]); !unicode.IsLetter(first) && first != '_' {
		return "app_" + normalized
	}
	return normalized
}

func scaffoldNormalizeExportedName(name string) string {
	out := scaffoldToExported(scaffoldSplitWords(name))
	if out == "" {
		return "App"
	}
	return out
}

func scaffoldCapitalizeFirst(word string) string {
	if word == "" {
		return ""
	}
	lower := strings.ToLower(word)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

const projectMainTemplate = `package main

import (
"fmt"
"log"
"net/http"
"path/filepath"
"runtime"

ginpkg "github.com/gin-gonic/gin"
ninja "github.com/shijl0925/gin-ninja"
ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
"{{ .AppImportPath }}"
"github.com/shijl0925/gin-ninja/middleware"
"github.com/shijl0925/gin-ninja/orm"
"github.com/shijl0925/gin-ninja/pkg/logger"
"github.com/shijl0925/gin-ninja/settings"
"go.uber.org/zap"
"gorm.io/gorm"
)

var runMain = run
var fatalMain = func(v ...any) { log.Fatal(v...) }

func initDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
db, err := ginbootstrap.InitDB(cfg)
if err != nil {
return nil, fmt.Errorf("init db: %w", err)
}
if err := db.AutoMigrate(&app.{{ .App.ModelName }}{}); err != nil {
return nil, fmt.Errorf("auto migrate: %w", err)
}
orm.Init(db)
return db, nil
}

func buildAPI(cfg settings.Config, db *gorm.DB, log_ *zap.Logger) *ninja.NinjaAPI {
api := ninja.New(ninja.Config{
Title:             cfg.App.Name,
Version:           cfg.App.Version,
Prefix:            "/api/v1",
DisableGinDefault: true,
})

api.UseGin(
ginpkg.Logger(),
middleware.RequestID(),
middleware.Recovery(log_),
middleware.Logger(log_),
middleware.CORS(nil),
orm.Middleware(db),
)

app.RegisterRoutes(api)
api.Engine().GET("/health", func(c *ginpkg.Context) {
c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
})
return api
}

func run(cfg settings.Config, log_ *zap.Logger) error {
db, err := initDB(&cfg.Database)
if err != nil {
return err
}
api := buildAPI(cfg, db, log_)
log.Printf("Starting %s v%s on http://%s", cfg.App.Name, cfg.App.Version, cfg.Server.Addr())
log.Printf("Swagger UI: http://%s/docs", cfg.Server.Addr())
return api.Run(cfg.Server.Addr())
}

func main() {
_, file, _, ok := runtime.Caller(0)
if !ok {
fatalMain("resolve config path")
}

cfg := settings.MustLoad(filepath.Join(filepath.Dir(file), "config.yaml"))
log_ := ginbootstrap.InitLogger(&cfg.Log)
defer logger.Sync()

if err := runMain(*cfg, log_); err != nil {
fatalMain(err)
}
}
`

const projectMainWrapperTemplate = `package main

import "{{ .Module }}/internal/server"

func main() {
server.Main()
}
`

const projectCmdServerTemplate = `package main

import "{{ .Module }}/internal/server"

func main() {
server.Main()
}
`

const projectInternalServerTemplate = `package server

import (
"fmt"
"log"
"net/http"
"path/filepath"
"runtime"

ginpkg "github.com/gin-gonic/gin"
projectbootstrap "{{ .Module }}/bootstrap"
"{{ .AppImportPath }}"
ninja "github.com/shijl0925/gin-ninja"
"github.com/shijl0925/gin-ninja/middleware"
"github.com/shijl0925/gin-ninja/orm"
"github.com/shijl0925/gin-ninja/pkg/logger"
"github.com/shijl0925/gin-ninja/settings"
"go.uber.org/zap"
"gorm.io/gorm"
)

var runMain = run
var fatalMain = func(v ...any) { log.Fatal(v...) }

func initDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
db, err := projectbootstrap.InitDB(cfg)
if err != nil {
return nil, fmt.Errorf("init db: %w", err)
}
if err := db.AutoMigrate(&app.{{ .App.ModelName }}{}); err != nil {
return nil, fmt.Errorf("auto migrate: %w", err)
}
orm.Init(db)
return db, nil
}

func buildAPI(cfg settings.Config, db *gorm.DB, log_ *zap.Logger) *ninja.NinjaAPI {
apiCfg := ninja.Config{
Title:             cfg.App.Name,
Version:           cfg.App.Version,
Prefix:            "/api/v1",
DisableGinDefault: true,
{{- if .Options.WithAuth }}
SecuritySchemes: map[string]ninja.SecurityScheme{
"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
},
{{- end }}
}
api := ninja.New(apiCfg)

api.UseGin(
ginpkg.Logger(),
middleware.RequestID(),
middleware.Recovery(log_),
middleware.Logger(log_),
middleware.CORS(nil),
orm.Middleware(db),
)

app.RegisterRoutes(api)
{{- if .Options.WithAuth }}
app.RegisterAuthRoutes(api)
{{- end }}
{{- if .Options.WithAdmin }}
app.RegisterAdminRoutes(api)
{{- end }}
api.Engine().GET("/health", func(c *ginpkg.Context) {
c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
})
return api
}

func run(cfg settings.Config, log_ *zap.Logger) error {
db, err := initDB(&cfg.Database)
if err != nil {
return err
}
api := buildAPI(cfg, db, log_)
log.Printf("Starting %s v%s on http://%s", cfg.App.Name, cfg.App.Version, cfg.Server.Addr())
log.Printf("Swagger UI: http://%s/docs", cfg.Server.Addr())
return api.Run(cfg.Server.Addr())
}

func projectRoot() string {
_, file, _, ok := runtime.Caller(0)
if !ok {
fatalMain("resolve config path")
}
return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func Main() {
root := projectRoot()
cfg := settings.MustLoadWithOverrides(
filepath.Join(root, "config.yaml"),
filepath.Join(root, "settings", "config.local.yaml"),
)
log_ := projectbootstrap.InitLogger(&cfg.Log)
defer logger.Sync()

if err := runMain(*cfg, log_); err != nil {
fatalMain(err)
}
}
`

const projectBootstrapDBTemplate = `package bootstrap

import (
ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
"github.com/shijl0925/gin-ninja/settings"
"gorm.io/gorm"
)

func InitDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
return ginbootstrap.InitDB(cfg)
}
`

const projectBootstrapLoggerTemplate = `package bootstrap

import (
ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
"github.com/shijl0925/gin-ninja/settings"
"go.uber.org/zap"
)

func InitLogger(cfg *settings.LogConfig) *zap.Logger {
return ginbootstrap.InitLogger(cfg)
}
`

const projectBootstrapCacheTemplate = `package bootstrap

import ninja "github.com/shijl0925/gin-ninja"

func InitCacheStore() ninja.ResponseCacheStore {
return ninja.NewMemoryCacheStore()
}
`

const projectGitignoreTemplate = `*.db
*.db-journal
.env
/tmp/
{{- if .Options.Standard }}
/bin/
/settings/config.local.yaml
{{- end }}
`

const projectConfigTemplate = `app:
  name: "{{ .AppName }}"
  env: "development"
  debug: true
  version: "1.0.0"

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 60
  write_timeout: 60

database:
  driver: "sqlite"
  dsn: "{{ .DatabaseFile }}"
  max_idle_conns: 5
  max_open_conns: 10
{{- if .Options.WithAuth }}

jwt:
  secret: "change-me-in-production"
  expire_hours: 24
  issuer: "{{ .Module }}"
{{- end }}

log:
  level: "info"
  format: "console"
  output: "stdout"
`

const projectConfigLocalTemplate = `app:
  env: "development"
  debug: true

server:
  port: 8080

log:
  level: "debug"
  format: "console"
`

const projectConfigProdTemplate = `app:
  env: "production"
  debug: false

server:
  host: "0.0.0.0"
  port: 8080

log:
  level: "info"
  format: "json"
{{- if .Options.WithAuth }}

jwt:
  secret: "replace-me"
{{- end }}
`

const projectEnvTemplate = `APP__NAME={{ .AppName }}
APP__ENV=development
APP__SERVER__PORT=8080
APP__DATABASE__DSN={{ .DatabaseFile }}
{{- if .Options.WithAuth }}
APP__JWT__SECRET=change-me-in-production
APP__JWT__ISSUER={{ .Module }}
{{- end }}
`

const projectMakefileTemplate = `.PHONY: run build test lint tidy

run:
go run .

build:
go build ./...

test:
go test ./...

lint:
go vet ./...

tidy:
go mod tidy
`

const projectDockerfileTemplate = `FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN go mod download && go build -o /out/app .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /out/app /app/app
COPY config.yaml /app/config.yaml
EXPOSE 8080
CMD ["/app/app"]
`

const projectDockerComposeTemplate = `services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./:/workspace
    working_dir: /workspace
    command: go run .
`

const projectREADMETemplate = `# {{ .AppName }}

Generated with gin-ninja scaffold template {{ .Options.Template }}.

## Quick start

~~~bash
go mod tidy
go run .
~~~

{{- if .Options.Standard }}
Alternative entrypoint:

~~~bash
go run ./cmd/server
~~~

Configuration overrides can be placed in {{ bt }}settings/config.local.yaml{{ bt }} (copy from {{ bt }}settings/config.local.yaml.example{{ bt }}).
{{- end }}

## Generated structure

- {{ bt }}config.yaml{{ bt }}
- {{ bt }}{{ .AppDir }}{{ bt }}
{{- if .Options.Standard }}
- {{ bt }}internal/server{{ bt }}
- {{ bt }}bootstrap{{ bt }}
- {{ bt }}settings/*.yaml.example{{ bt }}
- {{ bt }}Dockerfile{{ bt }} / {{ bt }}docker-compose.yml{{ bt }}
{{- end }}
{{- if .Options.WithAuth }}
- JWT login scaffold at {{ bt }}POST /api/v1/auth/login{{ bt }}
{{- end }}
{{- if .Options.WithAdmin }}
- Admin resource scaffold at {{ bt }}/api/v1/admin{{ bt }}
{{- end }}
`

const appModelsTemplate = `package {{ .PackageName }}

import "gorm.io/gorm"

type {{ .ModelName }} struct {
gorm.Model
Name string {{ bt }}gorm:"column:name;not null" json:"name"{{ bt }}
}
`

const appReposTemplate = `package {{ .PackageName }}

import "github.com/shijl0925/go-toolkits/gormx"

type I{{ .RepoName }} interface {
gormx.IBaseRepo[{{ .ModelName }}]
}

type {{ .RepoName | printf "%sImpl" }} struct {
gormx.BaseRepo[{{ .ModelName }}]
}

func New{{ .RepoName }}() I{{ .RepoName }} {
return &{{ .RepoName }}Impl{}
}
`

const appSchemasTemplate = `package {{ .PackageName }}

import "github.com/shijl0925/gin-ninja/pagination"

type {{ .OutName }} struct {
ID   uint   {{ bt }}json:"id"{{ bt }}
Name string {{ bt }}json:"name"{{ bt }}
}

type {{ .ListName }} struct {
pagination.PageInput
Search string {{ bt }}form:"search"{{ bt }}
}

type {{ .GetName }} struct {
ID uint {{ bt }}path:"id" binding:"required"{{ bt }}
}

type {{ .CreateName }} struct {
Name string {{ bt }}json:"name" binding:"required"{{ bt }}
}

type {{ .UpdateName }} struct {
ID   uint    {{ bt }}path:"id" binding:"required"{{ bt }}
Name *string {{ bt }}json:"name" binding:"omitempty"{{ bt }}
}

type {{ .DeleteName }} struct {
ID uint {{ bt }}path:"id" binding:"required"{{ bt }}
}
`

const appErrorsTemplate = `package {{ .PackageName }}

import ninja "github.com/shijl0925/gin-ninja"

const {{ .ModelName }}NameRequiredCode = 10001

func New{{ .ModelName }}NameRequiredError() error {
return ninja.NewBusinessError({{ .ModelName }}NameRequiredCode, "{{ lower .ModelName }} name is required")
}
`

const appServicesTemplate = `package {{ .PackageName }}

import (
"errors"
"strings"

ninja "github.com/shijl0925/gin-ninja"
"github.com/shijl0925/gin-ninja/orm"
"github.com/shijl0925/gin-ninja/pagination"
"github.com/shijl0925/go-toolkits/gormx"
"gorm.io/gorm"
)

type {{ .ServiceName }} struct {
repo I{{ .RepoName }}
}

func New{{ .ServiceName }}() *{{ .ServiceName }} {
return &{{ .ServiceName }}{repo: New{{ .RepoName }}()}
}

func repoDB(ctx *ninja.Context) *gorm.DB {
if ctx != nil && ctx.Context != nil {
return orm.WithContext(ctx.Context)
}
return gormx.GetDb()
}

func to{{ .ModelName }}Out(item {{ .ModelName }}) {{ .OutName }} {
return {{ .OutName }}{
ID:   item.ID,
Name: item.Name,
}
}

func (s *{{ .ServiceName }}) List(ctx *ninja.Context, in *{{ .ListName }}) (*pagination.Page[{{ .OutName }}], error) {
db := repoDB(ctx)
query, model := gormx.NewQuery[{{ .ModelName }}]()
if in.Search != "" {
query.Like(&model.Name, in.Search)
}

opts := append([]gormx.DBOption{gormx.UseDB(db)}, query.ToOptions()...)
items, total, err := s.repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
if err != nil {
return nil, err
}

out := make([]{{ .OutName }}, len(items))
for i, item := range items {
out[i] = to{{ .ModelName }}Out(item)
}
return pagination.NewPage(out, total, in.PageInput), nil
}

func (s *{{ .ServiceName }}) Get(ctx *ninja.Context, in *{{ .GetName }}) (*{{ .OutName }}, error) {
item, err := s.repo.SelectOneById(int(in.ID), gormx.UseDB(repoDB(ctx)))
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}
out := to{{ .ModelName }}Out(item)
return &out, nil
}

func (s *{{ .ServiceName }}) Create(ctx *ninja.Context, in *{{ .CreateName }}) (*{{ .OutName }}, error) {
name := strings.TrimSpace(in.Name)
if name == "" {
return nil, New{{ .ModelName }}NameRequiredError()
}
item := &{{ .ModelName }}{Name: name}
if err := s.repo.Insert(item, gormx.UseDB(repoDB(ctx))); err != nil {
return nil, err
}
out := to{{ .ModelName }}Out(*item)
return &out, nil
}

func (s *{{ .ServiceName }}) Update(ctx *ninja.Context, in *{{ .UpdateName }}) (*{{ .OutName }}, error) {
db := repoDB(ctx)
item, err := s.repo.SelectOneById(int(in.ID), gormx.UseDB(db))
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}

updates := map[string]interface{}{}
if in.Name != nil {
name := strings.TrimSpace(*in.Name)
if name == "" {
return nil, New{{ .ModelName }}NameRequiredError()
}
updates["name"] = name
item.Name = name
}
if len(updates) > 0 {
if err := s.repo.UpdateById(int(in.ID), updates, gormx.UseDB(db)); err != nil {
return nil, err
}
}

out := to{{ .ModelName }}Out(item)
return &out, nil
}

func (s *{{ .ServiceName }}) Delete(ctx *ninja.Context, in *{{ .DeleteName }}) error {
err := s.repo.DeleteById(int(in.ID), gormx.UseDB(repoDB(ctx)))
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
`

const appAPIsTemplate = `package {{ .PackageName }}

import (
"errors"

ninja "github.com/shijl0925/gin-ninja"
"github.com/shijl0925/gin-ninja/orm"
"github.com/shijl0925/gin-ninja/pagination"
"github.com/shijl0925/go-toolkits/gormx"
"gorm.io/gorm"
)

func repoDB(ctx *ninja.Context) *gorm.DB {
if ctx != nil && ctx.Context != nil {
return orm.WithContext(ctx.Context)
}
return gormx.GetDb()
}

func to{{ .ModelName }}Out(item {{ .ModelName }}) {{ .OutName }} {
return {{ .OutName }}{
ID:   item.ID,
Name: item.Name,
}
}

func List{{ .ModelPlural }}(ctx *ninja.Context, in *{{ .ListName }}) (*pagination.Page[{{ .OutName }}], error) {
db := repoDB(ctx)
repo := New{{ .RepoName }}()
query, model := gormx.NewQuery[{{ .ModelName }}]()
if in.Search != "" {
query.Like(&model.Name, in.Search)
}

opts := append([]gormx.DBOption{gormx.UseDB(db)}, query.ToOptions()...)
items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
if err != nil {
return nil, err
}

out := make([]{{ .OutName }}, len(items))
for i, item := range items {
out[i] = to{{ .ModelName }}Out(item)
}
return pagination.NewPage(out, total, in.PageInput), nil
}

func Get{{ .ModelName }}(ctx *ninja.Context, in *{{ .GetName }}) (*{{ .OutName }}, error) {
item, err := New{{ .RepoName }}().SelectOneById(int(in.ID), gormx.UseDB(repoDB(ctx)))
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}
out := to{{ .ModelName }}Out(item)
return &out, nil
}

func Create{{ .ModelName }}(ctx *ninja.Context, in *{{ .CreateName }}) (*{{ .OutName }}, error) {
repo := New{{ .RepoName }}()
item := &{{ .ModelName }}{
Name: in.Name,
}
if err := repo.Insert(item, gormx.UseDB(repoDB(ctx))); err != nil {
return nil, err
}
out := to{{ .ModelName }}Out(*item)
return &out, nil
}

func Update{{ .ModelName }}(ctx *ninja.Context, in *{{ .UpdateName }}) (*{{ .OutName }}, error) {
db := repoDB(ctx)
repo := New{{ .RepoName }}()
item, err := repo.SelectOneById(int(in.ID), gormx.UseDB(db))
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ninja.NotFoundError()
}
return nil, err
}

updates := map[string]interface{}{}
if in.Name != nil {
updates["name"] = *in.Name
item.Name = *in.Name
}
if len(updates) > 0 {
if err := repo.UpdateById(int(in.ID), updates, gormx.UseDB(db)); err != nil {
return nil, err
}
}

out := to{{ .ModelName }}Out(item)
return &out, nil
}

func Delete{{ .ModelName }}(ctx *ninja.Context, in *{{ .DeleteName }}) error {
err := New{{ .RepoName }}().DeleteById(int(in.ID), gormx.UseDB(repoDB(ctx)))
if errors.Is(err, gorm.ErrRecordNotFound) {
return ninja.NotFoundError()
}
return err
}
`

const appAPIsWithServicesTemplate = `package {{ .PackageName }}

import (
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
)

var {{ .ModelLower }}Service = New{{ .ServiceName }}()

func List{{ .ModelPlural }}(ctx *ninja.Context, in *{{ .ListName }}) (*pagination.Page[{{ .OutName }}], error) {
return {{ .ModelLower }}Service.List(ctx, in)
}

func Get{{ .ModelName }}(ctx *ninja.Context, in *{{ .GetName }}) (*{{ .OutName }}, error) {
return {{ .ModelLower }}Service.Get(ctx, in)
}

func Create{{ .ModelName }}(ctx *ninja.Context, in *{{ .CreateName }}) (*{{ .OutName }}, error) {
return {{ .ModelLower }}Service.Create(ctx, in)
}

func Update{{ .ModelName }}(ctx *ninja.Context, in *{{ .UpdateName }}) (*{{ .OutName }}, error) {
return {{ .ModelLower }}Service.Update(ctx, in)
}

func Delete{{ .ModelName }}(ctx *ninja.Context, in *{{ .DeleteName }}) error {
return {{ .ModelLower }}Service.Delete(ctx, in)
}
`

const appRoutersTemplate = `package {{ .PackageName }}

import (
ninja "github.com/shijl0925/gin-ninja"
{{- if .Options.WithAuth }}
"github.com/shijl0925/gin-ninja/middleware"
{{- end }}
)

func RegisterRoutes(api *ninja.NinjaAPI) {
router := ninja.NewRouter("/{{ .RouteBase }}", ninja.WithTags("{{ .RouteTag }}"){{ if .Options.WithAuth }}, ninja.WithBearerAuth(){{ end }})
{{- if .Options.WithAuth }}
router.UseGin(middleware.JWTAuth())
{{- end }}
ninja.Get(router, "/", List{{ .ModelPlural }}, ninja.Summary("List {{ .RouteTag }}"))
ninja.Get(router, "/:id", Get{{ .ModelName }}, ninja.Summary("Get {{ .ModelName }}"))
ninja.Post(router, "/", Create{{ .ModelName }}, ninja.Summary("Create {{ .ModelName }}"), ninja.WithTransaction())
ninja.Put(router, "/:id", Update{{ .ModelName }}, ninja.Summary("Update {{ .ModelName }}"), ninja.WithTransaction())
ninja.Delete(router, "/:id", Delete{{ .ModelName }}, ninja.Summary("Delete {{ .ModelName }}"), ninja.WithTransaction())
api.AddRouter(router)
}
`

const appAuthTemplate = `package {{ .PackageName }}

import (
ninja "github.com/shijl0925/gin-ninja"
"github.com/shijl0925/gin-ninja/middleware"
)

type LoginInput struct {
UserID   uint   {{ bt }}json:"user_id" binding:"required"{{ bt }}
Username string {{ bt }}json:"username" binding:"required"{{ bt }}
}

type LoginOutput struct {
Token string {{ bt }}json:"token"{{ bt }}
}

func Login(ctx *ninja.Context, in *LoginInput) (*LoginOutput, error) {
token, err := middleware.GenerateToken(in.UserID, in.Username)
if err != nil {
return nil, err
}
return &LoginOutput{Token: token}, nil
}

func RegisterAuthRoutes(api *ninja.NinjaAPI) {
router := ninja.NewRouter("/auth", ninja.WithTags("Auth"))
ninja.Post(router, "/login", Login, ninja.Summary("Issue a JWT token"))
api.AddRouter(router)
}
`

const appAdminTemplate = `package {{ .PackageName }}

import (
admin "github.com/shijl0925/gin-ninja/admin"
ninja "github.com/shijl0925/gin-ninja"
{{- if .Options.WithAuth }}
"github.com/shijl0925/gin-ninja/middleware"
{{- end }}
)

func NewAdminSite() *admin.Site {
{{- if .Options.WithAuth }}
site := admin.NewSite(admin.WithPermissionChecker(AllowAdminAccess))
{{- else }}
site := admin.NewSite()
{{- end }}
site.MustRegisterModel(&admin.ModelResource{
Model:        {{ .ModelName }}{},
ListFields:   []string{"id", "name", "createdAt", "updatedAt"},
DetailFields: []string{"id", "name", "createdAt", "updatedAt"},
CreateFields: []string{"name"},
UpdateFields: []string{"name"},
FilterFields: []string{"name", "createdAt"},
SortFields:   []string{"id", "name", "createdAt", "updatedAt"},
SearchFields: []string{"name"},
})
return site
}

func RegisterAdminRoutes(api *ninja.NinjaAPI) {
router := ninja.NewRouter("/admin", ninja.WithTags("Admin"), ninja.WithVersion("v1"){{ if .Options.WithAuth }}, ninja.WithBearerAuth(){{ end }})
{{- if .Options.WithAuth }}
router.UseGin(middleware.JWTAuth())
{{- end }}
NewAdminSite().Mount(router)
api.AddRouter(router)
}
`

const appPermissionsTemplate = `package {{ .PackageName }}

import (
admin "github.com/shijl0925/gin-ninja/admin"
ninja "github.com/shijl0925/gin-ninja"
)

func AllowAdminAccess(ctx *ninja.Context, action admin.Action, resource *admin.Resource) error {
if ctx.GetUserID() == 0 {
return ninja.UnauthorizedError()
}
return nil
}
`

const appTestsTemplate = `package {{ .PackageName }}

import (
"testing"
{{- if .Options.WithAuth }}
"github.com/shijl0925/gin-ninja/settings"
{{- end }}
)

func TestScaffoldConstructors(t *testing.T) {
if New{{ .RepoName }}() == nil {
t.Fatal("expected repository constructor to return a value")
}
{{- if .Options.Standard }}
if New{{ .ServiceName }}() == nil {
t.Fatal("expected service constructor to return a value")
}
{{- end }}
}

{{- if .Options.WithAuth }}
func TestLoginIssuesToken(t *testing.T) {
prev := settings.Global.JWT
t.Cleanup(func() { settings.Global.JWT = prev })
settings.Global.JWT.Secret = "test-secret"
settings.Global.JWT.ExpireHours = 1
out, err := Login(nil, &LoginInput{UserID: 1, Username: "demo"})
if err != nil {
t.Fatalf("Login: %v", err)
}
if out == nil || out.Token == "" {
t.Fatal("expected a token in login output")
}
}
{{- end }}
`
