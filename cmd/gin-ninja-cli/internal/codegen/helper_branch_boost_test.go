package codegen

import (
	"errors"
	"go/ast"
	"os"
	"strings"
	"testing"
)

func TestCodegenHelperBranchBoost(t *testing.T) {
	t.Parallel()

	t.Run("crud helper edge branches", func(t *testing.T) {
		imports := map[string]string{"time": "time", "pkg": "example.com/pkg"}

		if lowerCamel("") != "" || lowerCamel("UserI") != "userI" {
			t.Fatalf("unexpected lowerCamel edge behavior")
		}
		if got := parseTagLiteral(&ast.BasicLit{Value: "`json:\"name\"`"}); got.Get("json") != "name" {
			t.Fatalf("unexpected parsed tag literal: %q", got.Get("json"))
		}
		if got := parseTagLiteral(&ast.BasicLit{Value: "bad"}); got != "" {
			t.Fatalf("expected invalid tag literal to be empty, got %q", got)
		}
		if toSnake("") != "" {
			t.Fatal("expected empty snake case result")
		}
		if defaultFilterOperator(&ast.Ident{Name: "string"}) != "eq" {
			t.Fatal("expected string filter operator to default to eq")
		}
		if !isListFilterType(&ast.StarExpr{X: &ast.Ident{Name: "string"}}, imports) {
			t.Fatal("expected pointer list filter type to be supported")
		}
		if isListFilterType(&ast.ArrayType{Elt: &ast.Ident{Name: "string"}}, imports) {
			t.Fatal("expected array list filter type to be rejected")
		}
		if isStringLikeExpr(&ast.ArrayType{Elt: &ast.Ident{Name: "string"}}) {
			t.Fatal("expected array type to be non-string-like")
		}
		if !containsRelationOut([]relationOutSpec{{TypeName: "UserOut"}}, "UserOut") || containsRelationOut([]relationOutSpec{{TypeName: "UserOut"}}, "TeamOut") {
			t.Fatal("unexpected containsRelationOut behavior")
		}
		if got := uniqueSortFields([]sortFieldSpec{{Alias: "name", Column: "name"}, {Alias: "name", Column: "name"}, {Alias: "email", Column: "email"}}); len(got) != 2 {
			t.Fatalf("expected duplicate sort fields to collapse, got %+v", got)
		}
		if !isRenderableFieldType(&ast.ArrayType{Elt: &ast.StructType{}}, imports) {
			t.Fatal("expected array of structs to be renderable")
		}
		if !isRenderableFieldType(&ast.InterfaceType{Methods: &ast.FieldList{}}, imports) {
			t.Fatal("expected empty interface to be renderable")
		}
		if isRenderableFieldType(&ast.SelectorExpr{X: &ast.SelectorExpr{}, Sel: &ast.Ident{Name: "Time"}}, imports) {
			t.Fatal("expected nested selector without ident root to be rejected")
		}
		if isRenderableFieldType(&ast.IndexExpr{X: &ast.Ident{Name: "Slice"}, Index: &ast.FuncType{}}, imports) {
			t.Fatal("expected unsupported index expr to be rejected")
		}
		if isRenderableFieldType(&ast.IndexListExpr{X: &ast.Ident{Name: "Pair"}, Indices: []ast.Expr{&ast.FuncType{}}}, imports) {
			t.Fatal("expected unsupported index list expr to be rejected")
		}
		if isEmbeddedGormModel(&ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Thing"}}, map[string]string{"pkg": "gorm.io/gorm"}) {
			t.Fatal("expected non-Model selector to be rejected")
		}
		if isEmbeddedGormModel(&ast.SelectorExpr{X: &ast.SelectorExpr{}, Sel: &ast.Ident{Name: "Model"}}, map[string]string{"pkg": "gorm.io/gorm"}) {
			t.Fatal("expected selector with non-ident root to be rejected")
		}
		if parseGORMSetting("column: ; foreignKey: OwnerID", "column") != "" || parseGORMSetting("", "column") != "" {
			t.Fatal("expected empty gorm settings to return empty values")
		}
		collector := newImportCollector(map[string]importSpec{
			"b": {Alias: "b", Path: "example.com/same"},
			"a": {Alias: "a", Path: "example.com/same"},
		})
		collector.addExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "b"}, Sel: &ast.Ident{Name: "Thing"}})
		collector.addExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "a"}, Sel: &ast.Ident{Name: "Thing"}})
		importList := collector.list()
		if len(importList) != 2 || importList[0].Alias != "a" || importList[1].Alias != "b" {
			t.Fatalf("expected import list to sort by alias when paths match, got %+v", importList)
		}
	})

	t.Run("scaffold helper edge branches", func(t *testing.T) {
		if scaffoldJoinTitle(nil) != "" {
			t.Fatal("expected empty title join result")
		}
		if scaffoldCapitalizeFirst("") != "" {
			t.Fatal("expected empty capitalize result")
		}
		if scaffoldNormalizeExportedName("123 api") != "App123Api" {
			t.Fatalf("unexpected normalized exported name")
		}
		if data, err := buildAppTemplateData(AppScaffoldConfig{Name: "post category"}, scaffoldOptions{}); err != nil {
			t.Fatalf("expected default app template data, got %v", err)
		} else if data.PackageName != "post_category" || data.ModelName != "PostCategory" || data.ModelPlural != "PostCategories" || data.RouteBase != "post-categories" {
			t.Fatalf("unexpected default app template data: %+v", data)
		}
		if _, err := buildAppTemplateData(AppScaffoldConfig{}, scaffoldOptions{}); err == nil || !strings.Contains(err.Error(), "name is required") {
			t.Fatalf("expected missing name error, got %v", err)
		}
		if _, err := resolveScaffoldOptions("standard", false, false, false, nil); err != nil {
			t.Fatalf("expected standard template to resolve: %v", err)
		}
		if err := WriteAppScaffold(AppScaffoldConfig{Name: "blog"}, " "); err == nil || !strings.Contains(err.Error(), "output directory is required") {
			t.Fatalf("expected missing output directory error, got %v", err)
		}
		if err := WriteAppScaffold(AppScaffoldConfig{Name: "blog", Template: "nope"}, t.TempDir()); err == nil || !strings.Contains(err.Error(), "unknown scaffold template") {
			t.Fatalf("expected invalid app template error, got %v", err)
		}
		if err := WriteAppScaffold(AppScaffoldConfig{Name: "!!!"}, t.TempDir()); err == nil || !strings.Contains(err.Error(), "does not contain any valid letters or digits") {
			t.Fatalf("expected invalid app name error, got %v", err)
		}
		if err := WriteProjectScaffold(ProjectScaffoldConfig{
			Module:   "github.com/acme/demo",
			Template: "nope",
		}, t.TempDir()); err == nil || !strings.Contains(err.Error(), "unknown scaffold template") {
			t.Fatalf("expected invalid project template error, got %v", err)
		}
		if err := WriteProjectScaffold(ProjectScaffoldConfig{
			Module: "github.com/acme/demo",
			AppDir: "/absolute",
		}, t.TempDir()); err == nil || !strings.Contains(err.Error(), "scaffold subdirectory must be relative") {
			t.Fatalf("expected invalid app dir error, got %v", err)
		}
		restrictedDir := t.TempDir()
		if err := os.Chmod(restrictedDir, 0o000); err == nil {
			defer os.Chmod(restrictedDir, 0o755)
			if err := ensureScaffoldDir(restrictedDir, false); err == nil || !strings.Contains(err.Error(), "read output directory") {
				t.Fatalf("expected read output directory error, got %v", err)
			}
		}
		if err := ensureScaffoldDir("\x00", false); err == nil || !strings.Contains(err.Error(), "stat output directory") {
			t.Fatalf("expected stat output directory error, got %v", err)
		}
		if _, err := executeGoTemplate("broken.go", `package demo
var _ = "{{call .Broken}}"`, struct{ Broken func() (string, error) }{
			Broken: func() (string, error) { return "", errors.New("boom") },
		}); err == nil || !strings.Contains(err.Error(), "execute template") {
			t.Fatalf("expected execute template error, got %v", err)
		}
	})
}
