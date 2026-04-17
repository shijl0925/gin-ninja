package codegen

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCRUDFileAndHelperCoverage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modelFile := filepath.Join(dir, "models.go")
	if err := os.WriteFile(modelFile, []byte(`package demo

import "time"

type User struct {
	ID        uint
	Name      string
	CreatedAt time.Time
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	output := filepath.Join(dir, DefaultOutputName("UserProfile"))
	if err := WriteCRUDFile(CRUDConfig{
		Model:     "User",
		ModelFile: modelFile,
	}, output); err != nil {
		t.Fatalf("WriteCRUDFile: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("expected generated CRUD file: %v", err)
	}

	if DefaultOutputName("UserProfile") != "user_profile_crud_gen.go" {
		t.Fatalf("unexpected DefaultOutputName result")
	}
	if optionalType("string") != "*string" || optionalType("*string") != "*string" {
		t.Fatalf("unexpected optionalType behavior")
	}
	if pluralizeName("category") != "categories" || pluralizeName("box") != "boxes" || pluralizeName("car") != "cars" {
		t.Fatalf("unexpected pluralizeName behavior")
	}
	if !isVowel('a') || isVowel('b') {
		t.Fatalf("unexpected isVowel behavior")
	}
	if lowerCamel("HTTPRequestID") != "httpRequestId" {
		t.Fatalf("unexpected lowerCamel output")
	}
	if toSnake("HTTPRequestID") != "http_request_id" {
		t.Fatalf("unexpected toSnake output")
	}
}

func TestCRUDASTHelperCoverage(t *testing.T) {
	t.Parallel()

	imports := map[string]string{
		"time": "time",
		"gorm": "gorm.io/gorm",
	}

	if defaultFilterOperator(nil) != "eq" {
		t.Fatal("expected eq default filter operator")
	}
	if !isListFilterType(&ast.Ident{Name: "string"}, nil) {
		t.Fatal("expected string list filter type to be allowed")
	}
	if isListFilterType(&ast.Ident{Name: "interface{}"}, nil) || isListFilterType(&ast.Ident{Name: "any"}, nil) {
		t.Fatal("expected interface{} and any list filters to be rejected")
	}
	if !isListFilterType(&ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}, imports) {
		t.Fatal("expected imported selector list filter type")
	}
	if !isStringLikeExpr(&ast.StarExpr{X: &ast.Ident{Name: "string"}}) || isStringLikeExpr(&ast.Ident{Name: "int"}) {
		t.Fatal("unexpected isStringLikeExpr behavior")
	}
	if !isRenderableFieldType(&ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: &ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}},
	}, imports) {
		t.Fatal("expected map with imported selector to be renderable")
	}
	if !isRenderableFieldType(&ast.IndexExpr{
		X:     &ast.Ident{Name: "Slice"},
		Index: &ast.Ident{Name: "int"},
	}, imports) {
		t.Fatal("expected index expr to be renderable")
	}
	if !isRenderableFieldType(&ast.IndexListExpr{
		X:       &ast.Ident{Name: "Pair"},
		Indices: []ast.Expr{&ast.Ident{Name: "string"}, &ast.Ident{Name: "int"}},
	}, imports) {
		t.Fatal("expected index list expr to be renderable")
	}
	if isRenderableFieldType(&ast.SelectorExpr{X: &ast.Ident{Name: "missing"}, Sel: &ast.Ident{Name: "Time"}}, imports) {
		t.Fatal("expected unknown selector expr to be rejected")
	}
	if !isEmbeddedGormModel(&ast.SelectorExpr{X: &ast.Ident{Name: "gorm"}, Sel: &ast.Ident{Name: "Model"}}, imports) {
		t.Fatal("expected embedded gorm model to be detected")
	}
}

func TestScaffoldNamingHelperCoverage(t *testing.T) {
	t.Parallel()

	if !shouldSplitWord('a', 'B', 0) || !shouldSplitWord('A', '1', 0) || !shouldSplitWord('1', 'A', 0) || !shouldSplitWord('A', 'B', 'c') {
		t.Fatal("expected shouldSplitWord to cover transition cases")
	}
	if shouldSplitWord(0, 'A', 0) {
		t.Fatal("expected leading rune to avoid split")
	}
	if peekRune("abc", 0) != 'b' || peekRune("abc", 2) != 0 {
		t.Fatal("unexpected peekRune behavior")
	}
	if scaffoldToSeparated([]string{"Hello", "", "World"}, "-", true) != "hello-world" {
		t.Fatal("unexpected scaffoldToSeparated result")
	}
	if scaffoldJoinTitle([]string{"hello", "", "WORLD"}) != "Hello World" {
		t.Fatal("unexpected scaffoldJoinTitle result")
	}
	if scaffoldToExported(nil) != "App" || scaffoldToExported([]string{"123", "demo"}) != "App123Demo" {
		t.Fatal("unexpected scaffoldToExported result")
	}
	if scaffoldNormalizePackageName("") != "app" || scaffoldNormalizePackageName("123 Demo") != "app_123_demo" {
		t.Fatal("unexpected scaffoldNormalizePackageName result")
	}
	if scaffoldNormalizeExportedName("") != "App" || scaffoldNormalizeExportedName("hello world") != "HelloWorld" {
		t.Fatal("unexpected scaffoldNormalizeExportedName result")
	}
	if scaffoldCapitalizeFirst("hELLO") != "Hello" {
		t.Fatal("unexpected scaffoldCapitalizeFirst result")
	}
}
