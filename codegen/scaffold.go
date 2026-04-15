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

// ProjectScaffoldConfig defines the inputs for a new project scaffold.
type ProjectScaffoldConfig struct {
	Name   string
	Module string
}

// AppScaffoldConfig defines the inputs for a new app scaffold.
type AppScaffoldConfig struct {
	Name        string
	PackageName string
	ModelName   string
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
	if err := ensureScaffoldDir(outputDir); err != nil {
		return err
	}

	projectName := strings.TrimSpace(cfg.Name)
	if projectName == "" {
		projectName = filepath.Base(outputDir)
	}
	words := splitWords(projectName)
	if len(words) == 0 {
		words = splitWords(filepath.Base(module))
	}
	if len(words) == 0 {
		words = []string{"app"}
	}

	appData, err := buildAppTemplateData(AppScaffoldConfig{
		Name:        "example",
		PackageName: "app",
		ModelName:   "Example",
	})
	if err != nil {
		return err
	}

	data := projectTemplateData{
		Module:       module,
		AppName:      joinTitle(words),
		DatabaseFile: toSeparated(words, "_", true) + ".db",
		App:          appData,
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
	if err := ensureScaffoldDir(outputDir); err != nil {
		return err
	}
	data, err := buildAppTemplateData(cfg)
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
	Module       string
	AppName      string
	DatabaseFile string
	App          appTemplateData
}

type appTemplateData struct {
	PackageName string
	ModelName   string
	ModelPlural string
	RepoName    string
	OutName     string
	ListName    string
	GetName     string
	CreateName  string
	UpdateName  string
	DeleteName  string
	RouteBase   string
	RouteTag    string
}

func buildAppTemplateData(cfg AppScaffoldConfig) (appTemplateData, error) {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return appTemplateData{}, fmt.Errorf("name is required")
	}
	words := splitWords(name)
	if len(words) == 0 {
		return appTemplateData{}, fmt.Errorf("name %q does not contain any valid letters or digits", name)
	}

	packageName := strings.TrimSpace(cfg.PackageName)
	if packageName == "" {
		packageName = toSeparated(words, "_", true)
	}
	packageName = normalizePackageName(packageName)

	modelName := strings.TrimSpace(cfg.ModelName)
	if modelName == "" {
		modelName = toExported(words)
	}
	modelName = normalizeExportedName(modelName)

	modelPlural := inflection.Plural(modelName)
	pluralWords := splitWords(modelPlural)
	if len(pluralWords) == 0 {
		pluralWords = append([]string(nil), words...)
		pluralWords = append(pluralWords, "items")
	}

	return appTemplateData{
		PackageName: packageName,
		ModelName:   modelName,
		ModelPlural: modelPlural,
		RepoName:    modelName + "Repo",
		OutName:     modelName + "Out",
		ListName:    "List" + modelPlural + "Input",
		GetName:     "Get" + modelName + "Input",
		CreateName:  "Create" + modelName + "Input",
		UpdateName:  "Update" + modelName + "Input",
		DeleteName:  "Delete" + modelName + "Input",
		RouteBase:   toSeparated(pluralWords, "-", true),
		RouteTag:    joinTitle(pluralWords),
	}, nil
}

func projectFiles(data projectTemplateData) (map[string][]byte, error) {
	files := map[string][]byte{
		"go.mod":      []byte(fmt.Sprintf("module %s\n\ngo 1.26\n", data.Module)),
		"config.yaml": []byte(executeTextTemplate(projectConfigTemplate, data)),
		".gitignore":  []byte("*.db\n*.db-journal\n.env\n/tmp/\n"),
	}

	mainGo, err := executeGoTemplate("project_main", projectMainTemplate, data)
	if err != nil {
		return nil, err
	}
	files["main.go"] = mainGo

	appFiles, err := appFiles(data.App)
	if err != nil {
		return nil, err
	}
	for name, content := range appFiles {
		files[filepath.Join("app", name)] = content
	}
	return files, nil
}

func appFiles(data appTemplateData) (map[string][]byte, error) {
	templates := map[string]string{
		"models.go":  appModelsTemplate,
		"repos.go":   appReposTemplate,
		"schemas.go": appSchemasTemplate,
		"apis.go":    appAPIsTemplate,
		"routers.go": appRoutersTemplate,
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

func ensureScaffoldDir(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("%s already exists and is not a directory", dir)
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return fmt.Errorf("read output directory: %w", readErr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("%s already exists and is not empty", dir)
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
		"bt": func() string { return "`" },
	}).Parse(source))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func splitWords(input string) []string {
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

func toSeparated(words []string, sep string, lower bool) string {
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

func joinTitle(words []string) string {
	if len(words) == 0 {
		return ""
	}
	out := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" {
			continue
		}
		lower := strings.ToLower(word)
		out = append(out, strings.ToUpper(lower[:1])+lower[1:])
	}
	return strings.Join(out, " ")
}

func toExported(words []string) string {
	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		lower := strings.ToLower(word)
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
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

func normalizePackageName(name string) string {
	words := splitWords(name)
	normalized := toSeparated(words, "_", true)
	if normalized == "" {
		return "app"
	}
	if first := rune(normalized[0]); !unicode.IsLetter(first) && first != '_' {
		return "app_" + normalized
	}
	return normalized
}

func normalizeExportedName(name string) string {
	out := toExported(splitWords(name))
	if out == "" {
		return "App"
	}
	return out
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
	"github.com/shijl0925/gin-ninja/bootstrap"
	"{{ .Module }}/app"
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
	db, err := bootstrap.InitDB(cfg)
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
	log_ := bootstrap.InitLogger(&cfg.Log)
	defer logger.Sync()

	if err := runMain(*cfg, log_); err != nil {
		fatalMain(err)
	}
}
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

log:
  level: "info"
  format: "console"
  output: "stdout"
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

const appRoutersTemplate = `package {{ .PackageName }}

import ninja "github.com/shijl0925/gin-ninja"

func RegisterRoutes(api *ninja.NinjaAPI) {
	router := ninja.NewRouter("/{{ .RouteBase }}", ninja.WithTags("{{ .RouteTag }}"))
	ninja.Get(router, "/", List{{ .ModelPlural }}, ninja.Summary("List {{ .RouteTag }}"))
	ninja.Get(router, "/:id", Get{{ .ModelName }}, ninja.Summary("Get {{ .ModelName }}"))
	ninja.Post(router, "/", Create{{ .ModelName }}, ninja.Summary("Create {{ .ModelName }}"), ninja.WithTransaction())
	ninja.Put(router, "/:id", Update{{ .ModelName }}, ninja.Summary("Update {{ .ModelName }}"), ninja.WithTransaction())
	ninja.Delete(router, "/:id", Delete{{ .ModelName }}, ninja.Summary("Delete {{ .ModelName }}"), ninja.WithTransaction())
	api.AddRouter(router)
}
`
