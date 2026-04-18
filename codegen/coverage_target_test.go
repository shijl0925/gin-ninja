package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/template"
)

func TestScaffoldFallbackAndWriteCoverage(t *testing.T) {
	t.Parallel()

	t.Run("write project scaffold validates inputs and naming fallbacks", func(t *testing.T) {
		if err := WriteProjectScaffold(ProjectScaffoldConfig{}, ""); err == nil || !strings.Contains(err.Error(), "module is required") {
			t.Fatalf("expected missing module error, got %v", err)
		}
		if err := WriteProjectScaffold(ProjectScaffoldConfig{Module: "github.com/acme/demo"}, "   "); err == nil || !strings.Contains(err.Error(), "output directory is required") {
			t.Fatalf("expected missing output directory error, got %v", err)
		}

		dir := t.TempDir()
		outputDir := filepath.Join(dir, "---")
		if err := WriteProjectScaffold(ProjectScaffoldConfig{
			Module: "github.com/acme/my-service",
		}, outputDir); err != nil {
			t.Fatalf("WriteProjectScaffold(module fallback): %v", err)
		}

		configContent, err := os.ReadFile(filepath.Join(outputDir, "config.yaml"))
		if err != nil {
			t.Fatalf("read config.yaml: %v", err)
		}
		if !strings.Contains(string(configContent), `dsn: "my_service.db"`) {
			t.Fatalf("expected module-name database fallback, got:\n%s", configContent)
		}

		outputBaseDir := filepath.Join(dir, "story-blog")
		if err := WriteProjectScaffold(ProjectScaffoldConfig{
			Module: "github.com/acme/ignored",
		}, outputBaseDir); err != nil {
			t.Fatalf("WriteProjectScaffold(output basename fallback): %v", err)
		}
		outputBaseConfig, err := os.ReadFile(filepath.Join(outputBaseDir, "config.yaml"))
		if err != nil {
			t.Fatalf("read output basename config.yaml: %v", err)
		}
		if !strings.Contains(string(outputBaseConfig), `dsn: "story_blog.db"`) {
			t.Fatalf("expected output basename database fallback, got:\n%s", outputBaseConfig)
		}

		appFallbackDir := filepath.Join(dir, "___")
		if err := WriteProjectScaffold(ProjectScaffoldConfig{
			Module:   "!!!",
			Template: "minimal",
			Force:    true,
		}, appFallbackDir); err != nil {
			t.Fatalf("WriteProjectScaffold(app fallback): %v", err)
		}

		appFallbackConfig, err := os.ReadFile(filepath.Join(appFallbackDir, "config.yaml"))
		if err != nil {
			t.Fatalf("read app fallback config.yaml: %v", err)
		}
		if !strings.Contains(string(appFallbackConfig), `dsn: "app.db"`) {
			t.Fatalf("expected app database fallback, got:\n%s", appFallbackConfig)
		}
	})

	t.Run("write scaffold files reports directory and write errors", func(t *testing.T) {
		root := t.TempDir()

		blocker := filepath.Join(root, "blocker")
		if err := os.WriteFile(blocker, []byte("busy"), 0o644); err != nil {
			t.Fatalf("write blocker: %v", err)
		}
		if err := writeScaffoldFiles(root, map[string][]byte{
			filepath.Join("blocker", "file.txt"): []byte("x"),
		}); err == nil || !strings.Contains(err.Error(), "create parent directory") {
			t.Fatalf("expected parent directory error, got %v", err)
		}

		existingDir := filepath.Join(root, "existing")
		if err := os.MkdirAll(existingDir, 0o755); err != nil {
			t.Fatalf("mkdir existing dir: %v", err)
		}
		if err := writeScaffoldFiles(root, map[string][]byte{
			"existing": []byte("x"),
		}); err == nil || !strings.Contains(err.Error(), "write existing") {
			t.Fatalf("expected write error, got %v", err)
		}
	})
}

func TestScaffoldTemplateSelectionCoverage(t *testing.T) {
	t.Parallel()

	minimalOpts, err := resolveScaffoldOptions("minimal", false, false, false, boolPtr(false))
	if err != nil {
		t.Fatalf("resolveScaffoldOptions(minimal): %v", err)
	}
	standardOpts, err := resolveScaffoldOptions("admin", true, false, false, boolPtr(true))
	if err != nil {
		t.Fatalf("resolveScaffoldOptions(admin): %v", err)
	}

	minimalApp, err := buildAppTemplateData(AppScaffoldConfig{
		Name:        "123 blog",
		PackageName: "123 blog",
		ModelName:   "123 post",
	}, minimalOpts)
	if err != nil {
		t.Fatalf("buildAppTemplateData(minimal): %v", err)
	}
	if minimalApp.PackageName != "app_123_blog" || minimalApp.ModelName != "App123Post" {
		t.Fatalf("unexpected normalized app data: %+v", minimalApp)
	}

	standardApp, err := buildAppTemplateData(AppScaffoldConfig{
		Name:        "admin blog",
		PackageName: "admin_blog",
		ModelName:   "Entry",
	}, standardOpts)
	if err != nil {
		t.Fatalf("buildAppTemplateData(standard): %v", err)
	}

	minimalFiles, err := appFiles(minimalApp)
	if err != nil {
		t.Fatalf("appFiles(minimal): %v", err)
	}
	for _, name := range []string{"models.go", "migrations.go", "schemas.go", "routers.go", "repos.go", "apis.go"} {
		if _, ok := minimalFiles[name]; !ok {
			t.Fatalf("expected minimal app file %q", name)
		}
	}
	for _, name := range []string{"services.go", "errors.go", "auth.go", "admin.go", "permissions.go", "scaffold_test.go"} {
		if _, ok := minimalFiles[name]; ok {
			t.Fatalf("did not expect minimal app file %q", name)
		}
	}
	if strings.Contains(string(minimalFiles["repos.go"]), "gormx") {
		t.Fatalf("expected minimal native repos template, got:\n%s", minimalFiles["repos.go"])
	}
	if strings.Contains(string(minimalFiles["apis.go"]), "gormx") {
		t.Fatalf("expected minimal native apis template, got:\n%s", minimalFiles["apis.go"])
	}

	standardFiles, err := appFiles(standardApp)
	if err != nil {
		t.Fatalf("appFiles(standard): %v", err)
	}
	for _, name := range []string{"models.go", "migrations.go", "schemas.go", "routers.go", "repos.go", "apis.go", "services.go", "errors.go", "auth.go", "admin.go", "permissions.go", "scaffold_test.go"} {
		if _, ok := standardFiles[name]; !ok {
			t.Fatalf("expected standard app file %q", name)
		}
	}
	if !strings.Contains(string(standardFiles["repos.go"]), "gormx") || !strings.Contains(string(standardFiles["services.go"]), "gormx") {
		t.Fatalf("expected gormx-backed standard templates")
	}

	minimalProjectFiles, err := projectFiles(projectTemplateData{
		Module:        "github.com/acme/blog",
		AppName:       "Blog",
		DatabaseFile:  "blog.db",
		AppDir:        "app",
		AppImportPath: "github.com/acme/blog/app",
		App:           minimalApp,
		Options:       minimalOpts,
	})
	if err != nil {
		t.Fatalf("projectFiles(minimal): %v", err)
	}
	if _, ok := minimalProjectFiles["main.go"]; !ok {
		t.Fatal("expected minimal main.go")
	}
	if _, ok := minimalProjectFiles[filepath.Join("cmd", "server", "main.go")]; ok {
		t.Fatal("did not expect standard cmd/server/main.go in minimal project")
	}

	standardProjectFiles, err := projectFiles(projectTemplateData{
		Module:        "github.com/acme/blog",
		AppName:       "Blog",
		DatabaseFile:  "blog.db",
		AppDir:        "internal/app",
		AppImportPath: "github.com/acme/blog/internal/app",
		App:           standardApp,
		Options:       standardOpts,
	})
	if err != nil {
		t.Fatalf("projectFiles(standard): %v", err)
	}
	for _, name := range []string{
		"main.go",
		filepath.Join("cmd", "server", "main.go"),
		filepath.Join("internal", "server", "server.go"),
		filepath.Join("bootstrap", "db.go"),
		filepath.Join("bootstrap", "logger.go"),
		filepath.Join("bootstrap", "cache.go"),
		"README.md",
		"Dockerfile",
		"docker-compose.yml",
		filepath.Join("internal", "app", "services.go"),
	} {
		if _, ok := standardProjectFiles[name]; !ok {
			t.Fatalf("expected standard project file %q", name)
		}
	}
}

func TestCRUDModelSpecCoverage(t *testing.T) {
	t.Parallel()

	t.Run("load model spec captures relations imports and overrides", func(t *testing.T) {
		dir := t.TempDir()
		modelFile := filepath.Join(dir, "models.go")
		if err := os.WriteFile(modelFile, []byte(`package demo

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Team struct {
	gorm.Model
	Name string `+"`json:\"name\"`"+`
}

type User struct {
	gorm.Model
	TeamID    uint           `+"`json:\"team_id\" crud:\"filter,sort\"`"+`
	Team      Team           `+"`gorm:\"foreignKey:TeamID\" json:\"team\"`"+`
	Email     string         `+"`json:\"email\" binding:\"required,email\" crud:\"filter:like,search\"`"+`
	Nickname  sql.NullString `+"`json:\"nickname\"`"+`
	CreatedAt time.Time      `+"`json:\"created_at\"`"+`
	Meta      map[string]any `+"`json:\"meta\"`"+`
	Ignored   chan int       `+"`json:\"ignored\"`"+`
}
`), 0o644); err != nil {
			t.Fatalf("write model file: %v", err)
		}

		spec, err := loadModelSpec(CRUDConfig{
			ModelFile:   modelFile,
			Model:       "User",
			PackageName: "custompkg",
			Tag:         "members",
			WithGormX:   boolPtr(false),
		})
		if err != nil {
			t.Fatalf("loadModelSpec: %v", err)
		}

		if spec.packageName != "custompkg" || spec.tag != "members" || spec.useGormX {
			t.Fatalf("unexpected model spec overrides: %+v", spec)
		}
		if spec.idField != "ID" || spec.idTypeExpr != "uint" || spec.idColumn != "id" {
			t.Fatalf("unexpected ID spec: field=%q type=%q column=%q", spec.idField, spec.idTypeExpr, spec.idColumn)
		}
		if spec.useByIDMethods {
			t.Fatalf("expected uint IDs to avoid by-id helpers, got %+v", spec)
		}

		if len(spec.relations) != 1 {
			t.Fatalf("expected one relation, got %+v", spec.relations)
		}
		relation := spec.relations[0]
		if relation.FieldName != "Team" || relation.TargetModel != "Team" || relation.InputName != "TeamID" || relation.UseAssociationInput || !relation.ExistingInputField {
			t.Fatalf("unexpected relation spec: %+v", relation)
		}
		if relation.ExistingCreateField != "TeamID" || relation.ExistingUpdateField != "TeamID" {
			t.Fatalf("expected relation to reuse TeamID field, got %+v", relation)
		}

		if len(spec.relationOuts) != 1 || spec.relationOuts[0].TypeName != "UserTeamOut" || spec.relationOuts[0].OutputFieldsValue != "id,name" {
			t.Fatalf("unexpected relation outs: %+v", spec.relationOuts)
		}

		if len(spec.listFields) != 2 {
			t.Fatalf("expected filterable list fields, got %+v", spec.listFields)
		}
		if spec.listFields[0].FilterTag != "team_id,eq" || spec.listFields[1].FilterTag != "email,like" {
			t.Fatalf("unexpected filter tags: %+v", spec.listFields)
		}
		if len(spec.sortFields) != 1 || spec.sortFields[0].Alias != "team_id" || spec.sortFields[0].Column != "team_id" {
			t.Fatalf("unexpected sort fields: %+v", spec.sortFields)
		}
		if !reflect.DeepEqual(spec.searchFields, []string{"email"}) {
			t.Fatalf("unexpected search fields: %+v", spec.searchFields)
		}

		if containsImportPath(spec.imports, "github.com/shijl0925/go-toolkits/gormx") {
			t.Fatalf("did not expect gormx import: %+v", spec.imports)
		}
		for _, path := range []string{
			"github.com/shijl0925/gin-ninja/filter",
			"github.com/shijl0925/gin-ninja/order",
			"database/sql",
			"gorm.io/gorm",
		} {
			if !containsImportPath(spec.imports, path) {
				t.Fatalf("expected import %q in %+v", path, spec.imports)
			}
		}

		if contains(spec.outputFields, "ignored") {
			t.Fatalf("expected non-renderable field to be skipped, got %+v", spec.outputFields)
		}
	})

	t.Run("load model spec validates required inputs and parse errors", func(t *testing.T) {
		if _, err := loadModelSpec(CRUDConfig{}); err == nil || !strings.Contains(err.Error(), "model file is required") {
			t.Fatalf("expected missing model file error, got %v", err)
		}

		dir := t.TempDir()
		modelFile := filepath.Join(dir, "models.go")
		if err := os.WriteFile(modelFile, []byte("package demo\n"), 0o644); err != nil {
			t.Fatalf("write model file: %v", err)
		}
		if _, err := loadModelSpec(CRUDConfig{ModelFile: modelFile}); err == nil || !strings.Contains(err.Error(), "model name is required") {
			t.Fatalf("expected missing model name error, got %v", err)
		}

		badFile := filepath.Join(dir, "bad.go")
		if err := os.WriteFile(badFile, []byte("package demo\nfunc {\n"), 0o644); err != nil {
			t.Fatalf("write bad model file: %v", err)
		}
		if _, err := loadModelSpec(CRUDConfig{ModelFile: badFile, Model: "User"}); err == nil || !strings.Contains(err.Error(), "parse model file") {
			t.Fatalf("expected parse model file error, got %v", err)
		}
	})

	t.Run("write crud file reports write failures", func(t *testing.T) {
		dir := t.TempDir()
		modelFile := filepath.Join(dir, "models.go")
		if err := os.WriteFile(modelFile, []byte("package demo\ntype User struct{ ID uint }\n"), 0o644); err != nil {
			t.Fatalf("write model file: %v", err)
		}

		outputDir := filepath.Join(dir, "generated")
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("mkdir output dir: %v", err)
		}
		if err := WriteCRUDFile(CRUDConfig{ModelFile: modelFile, Model: "User"}, outputDir); err == nil || !strings.Contains(err.Error(), "write generated file") {
			t.Fatalf("expected write generated file error, got %v", err)
		}
	})
}

func TestBuildStructModelSpecCoverage(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "models.go", `package demo

import "gorm.io/gorm"

type User struct {
	ID uint `+"`json:\"id\"`"+`
}

type Role struct {
	gorm.Model
	Name   string            `+"`json:\"name\"`"+`
	Secret string            `+"`json:\"-\"`"+`
	Meta   map[string]string `+"`json:\"meta\"`"+`
	Users  []User            `+"`gorm:\"many2many:role_users\" json:\"users\"`"+`
}
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	structTypes := map[string]*ast.StructType{}
	imports := map[string]string{"gorm": "gorm.io/gorm"}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				structTypes[typeSpec.Name.Name] = structType
			}
		}
	}

	spec, ok := buildStructModelSpec(fset, structTypes["Role"], structTypes, imports)
	if !ok {
		t.Fatal("expected buildStructModelSpec to succeed")
	}
	if spec.idField != "ID" || spec.idTypeExpr != "uint" || spec.idColumn != "id" {
		t.Fatalf("unexpected ID spec: %+v", spec)
	}
	if !reflect.DeepEqual(spec.outputFields, []string{"id", "name", "meta"}) {
		t.Fatalf("unexpected output fields: %+v", spec.outputFields)
	}
}

func containsImportPath(imports []importSpec, path string) bool {
	for _, imp := range imports {
		if imp.Path == path {
			return true
		}
	}
	return false
}

func TestGenerateCRUDErrorCoverage(t *testing.T) {
	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

type User struct {
	ID uint `+"`json:\"id\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	originalTemplate := crudTemplate
	t.Cleanup(func() {
		crudTemplate = originalTemplate
	})

	crudTemplate = template.Must(template.New("crud").Parse(`{{call .PackageName}}`))
	if _, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User"}); err == nil || !strings.Contains(err.Error(), "execute template") {
		t.Fatalf("expected execute template error, got %v", err)
	}

	crudTemplate = template.Must(template.New("crud").Parse("package demo\nfunc {"))
	if _, err := GenerateCRUD(CRUDConfig{ModelFile: modelFile, Model: "User"}); err == nil || !strings.Contains(err.Error(), "format generated source") {
		t.Fatalf("expected format generated source error, got %v", err)
	}
}
