package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestRunGenerateCRUDWithRichRelations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import "gorm.io/gorm"

type Tag struct {
	ID   uint   `+"`json:\"id\"`"+`
	Name string `+"`json:\"name\"`"+`
}

type Record struct {
	gorm.Model
	Email    string    `+"`json:\"email\" binding:\"required,email\"`"+`
	Tags     []Tag     `+"`gorm:\"many2many:record_tags;\" json:\"-\"`"+`
	TagIDs   []uint    `+"`gorm:\"-\" json:\"tag_ids\"`"+`
	Comments []Comment `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
}

type Comment struct {
	gorm.Model
	RecordID uint      `+"`json:\"record_id\" binding:\"required\"`"+`
	ParentID *uint     `+"`json:\"parent_id\"`"+`
	Body     string    `+"`json:\"body\" binding:\"required\"`"+`
	Record   Record    `+"`gorm:\"foreignKey:RecordID\" json:\"-\"`"+`
	Parent   *Comment  `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
	Children []Comment `+"`gorm:\"foreignKey:ParentID\" json:\"-\"`"+`
}
`), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"generate", "crud", "-model", "Comment", "-model-file", modelFile})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	outputFile := filepath.Join(dir, "comment_crud_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	text := string(content)
	for _, snippet := range []string{
		"func RegisterCommentCRUDRoutes",
		`RecordID uint ` + "`json:\"record_id\" binding:\"required\"`",
		`ParentID *uint ` + "`json:\"parent_id\"`",
		`query.Preload("Record")`,
		`query.Preload("Parent")`,
		`query.Preload("Children")`,
		`func syncCommentChildrenRelations`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected generated content to contain %q\n%s", snippet, text)
		}
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

func TestRunStartProjectWithoutGormx(t *testing.T) {
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
		"-with-gormx=false",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "app", "repos.go"))
	if err != nil {
		t.Fatalf("read repos.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm scaffold, got:\n%s", content)
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

func TestRunStartAppWithoutGormx(t *testing.T) {
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
		"-with-gormx=false",
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "services.go"))
	if err != nil {
		t.Fatalf("read services.go: %v", err)
	}
	if strings.Contains(string(content), "gormx") {
		t.Fatalf("expected native gorm service scaffold, got:\n%s", content)
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

func TestRunStartProjectGeneratedCRUDRoutesIntegration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputDir := filepath.Join(dir, "rich-project")

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"startproject",
		"rich-project",
		"-module", "github.com/acme/rich-project",
		"-output", outputDir,
	})
	if code != 0 {
		t.Fatalf("run exit code = %d stderr=%s", code, stderr.String())
	}

	injectRichCRUDScaffold(t, filepath.Join(outputDir, "app"), "app")
	runScaffoldModuleGoTest(t, outputDir)
}

func TestRunStartAppGeneratedCRUDRoutesIntegration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	outputDir := filepath.Join(dir, "blog")
	var stdout, stderr bytes.Buffer
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

	injectRichCRUDScaffold(t, outputDir, "blog")
	runScaffoldModuleGoTest(t, dir)
}

func injectRichCRUDScaffold(t *testing.T, appDir, packageName string) {
	t.Helper()

	modelFile := filepath.Join(appDir, "complex_models.go")
	if err := os.WriteFile(modelFile, []byte(richScaffoldModelSource(packageName)), 0o644); err != nil {
		t.Fatalf("write complex models: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"generate", "crud",
		"-model", "Comment",
		"-model-file", modelFile,
		"-output", filepath.Join(appDir, "comment_crud_gen.go"),
	})
	if code != 0 {
		t.Fatalf("generate crud exit code = %d stderr=%s", code, stderr.String())
	}

	for path, content := range map[string]string{
		filepath.Join(appDir, "comment_routes_testhook.go"): richScaffoldRouteHookSource(packageName),
		filepath.Join(appDir, "comment_runtime_test.go"):    richScaffoldIntegrationTestSource(packageName),
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", filepath.Base(path), err)
		}
	}
}

func richScaffoldModelSource(packageName string) string {
	return `package ` + packageName + `

import "gorm.io/gorm"

type Record struct {
	gorm.Model
	Email    string    ` + "`json:\"email\" binding:\"required,email\"`" + `
	Comments []Comment ` + "`gorm:\"foreignKey:RecordID\" json:\"-\"`" + `
}

type Comment struct {
	gorm.Model
	RecordID uint      ` + "`json:\"record_id\" binding:\"required\"`" + `
	ParentID *uint     ` + "`json:\"parent_id\"`" + `
	Body     string    ` + "`json:\"body\" binding:\"required\"`" + `
	Record   Record    ` + "`gorm:\"foreignKey:RecordID\" json:\"-\"`" + `
	Parent   *Comment  ` + "`gorm:\"foreignKey:ParentID\" json:\"-\"`" + `
	Children []Comment ` + "`gorm:\"foreignKey:ParentID\" json:\"-\"`" + `
}
`
}

func richScaffoldRouteHookSource(packageName string) string {
	return `package ` + packageName + `

import ninja "github.com/shijl0925/gin-ninja"

func RegisterScaffoldCommentRoutes(api *ninja.NinjaAPI) {
	router := ninja.NewRouter("/comments", ninja.WithTags("Comments"))
	RegisterCommentCRUDRoutes(router)
	api.AddRouter(router)
}
`
}

func richScaffoldIntegrationTestSource(packageName string) string {
	return `package ` + packageName + `

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRichScaffoldAPI(t *testing.T) (*ninja.NinjaAPI, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&Record{}, &Comment{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{DisableGinDefault: true})
	api.UseGin(orm.Middleware(db))
	RegisterRoutes(api)
	RegisterScaffoldCommentRoutes(api)
	return api, db
}

func performRichScaffoldJSON(t *testing.T, api *ninja.NinjaAPI, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	return rec
}

func TestScaffoldSupportsGeneratedCommentCRUD(t *testing.T) {
	api, db := newRichScaffoldAPI(t)

	record := Record{Email: "scaffold@example.com"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("db.Create record: %v", err)
	}

	rootResp := performRichScaffoldJSON(t, api, http.MethodPost, "/comments/", map[string]any{
		"record_id": record.ID,
		"body":      "root",
	})
	if rootResp.Code != http.StatusCreated {
		t.Fatalf("create root status=%d body=%s", rootResp.Code, rootResp.Body.String())
	}
	var root map[string]any
	if err := json.Unmarshal(rootResp.Body.Bytes(), &root); err != nil {
		t.Fatalf("json.Unmarshal root: %v", err)
	}
	rootID := uint(root["id"].(float64))

	childResp := performRichScaffoldJSON(t, api, http.MethodPost, "/comments/", map[string]any{
		"record_id": record.ID,
		"parent_id": rootID,
		"body":      "child",
	})
	if childResp.Code != http.StatusCreated {
		t.Fatalf("create child status=%d body=%s", childResp.Code, childResp.Body.String())
	}
	var child map[string]any
	if err := json.Unmarshal(childResp.Body.Bytes(), &child); err != nil {
		t.Fatalf("json.Unmarshal child: %v", err)
	}
	childID := uint(child["id"].(float64))

	detailResp := performRichScaffoldJSON(t, api, http.MethodGet, "/comments/"+strconv.FormatUint(uint64(childID), 10), nil)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detail); err != nil {
		t.Fatalf("json.Unmarshal detail: %v", err)
	}
	if got := uint(detail["record_id"].(float64)); got != record.ID {
		t.Fatalf("expected record_id %d, got %+v", record.ID, detail)
	}
	if got := uint(detail["parent_id"].(float64)); got != rootID {
		t.Fatalf("expected parent_id %d, got %+v", rootID, detail)
	}

	updateResp := performRichScaffoldJSON(t, api, http.MethodPatch, "/comments/"+strconv.FormatUint(uint64(rootID), 10), map[string]any{
		"body": "root updated",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}

	listResp := performRichScaffoldJSON(t, api, http.MethodGet, "/comments/?page=1&size=10", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	var page struct {
		Items []map[string]any ` + "`json:\"items\"`" + `
		Total int64            ` + "`json:\"total\"`" + `
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &page); err != nil {
		t.Fatalf("json.Unmarshal page: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("expected two generated comments, got %+v", page)
	}
}
`
}

func runScaffoldModuleGoTest(t *testing.T, dir string) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

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
