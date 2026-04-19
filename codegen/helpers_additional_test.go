package codegen

import (
	"errors"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	if lowerCamel("user") != "user" {
		t.Fatalf("expected lowerCamel to preserve lower-case words")
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
	if defaultFilterOperator(&ast.Ident{Name: "int"}) != "eq" {
		t.Fatal("expected non-string filter operator fallback to eq")
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

func TestCRUDAndScaffoldErrorCoverage(t *testing.T) {
	t.Parallel()

	t.Run("write crud file and scaffold validation errors", func(t *testing.T) {
		dir := t.TempDir()
		modelFile := filepath.Join(dir, "models.go")
		if err := os.WriteFile(modelFile, []byte("package demo\ntype User struct{ ID uint }\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("busy"), 0o644); err != nil {
			t.Fatalf("WriteFile(blocker): %v", err)
		}
		if err := WriteCRUDFile(CRUDConfig{ModelFile: modelFile, Model: "User"}, filepath.Join(blocker, "user_crud.go")); err == nil || !strings.Contains(err.Error(), "create output dir") {
			t.Fatalf("expected create output dir error, got %v", err)
		}

		if _, err := resolveScaffoldOptions("unknown", false, false, false, nil, nil); err == nil {
			t.Fatal("expected unknown scaffold template error")
		}
		for _, tc := range []struct {
			value string
		}{
			{value: "."},
			{value: "/absolute"},
			{value: "../escape"},
		} {
			if _, err := normalizeScaffoldSubdir(tc.value, "app"); err == nil {
				t.Fatalf("expected normalizeScaffoldSubdir(%q) to fail", tc.value)
			}
		}
		if _, err := buildAppTemplateData(AppScaffoldConfig{Name: "!!!"}, scaffoldOptions{}); err == nil {
			t.Fatal("expected invalid app name error")
		}
		if err := ensureScaffoldDir(blocker, false); err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("expected non-directory scaffold error, got %v", err)
		}
	})

	t.Run("template execution errors", func(t *testing.T) {
		if _, err := executeTextTemplate("{{", nil); err == nil || !strings.Contains(err.Error(), "parse template") {
			t.Fatalf("expected template parse error, got %v", err)
		}
		if _, err := executeGoTemplate("broken.go", "package demo\nfunc {", nil); err == nil || !strings.Contains(err.Error(), "format broken.go") {
			t.Fatalf("expected go format error, got %v", err)
		}
	})

	t.Run("ast helper edge branches", func(t *testing.T) {
		imports := map[string]string{"gorm": "gorm.io/gorm", "time": "time"}
		structTypes := map[string]*ast.StructType{
			"User": {Fields: &ast.FieldList{}},
		}
		if name, collection, pointer, ok := relationTargetModel(&ast.StarExpr{X: &ast.ArrayType{Elt: &ast.Ident{Name: "User"}}}, structTypes); !ok || name != "User" || !collection || !pointer {
			t.Fatalf("unexpected relationTargetModel result: %q %v %v %v", name, collection, pointer, ok)
		}
		if _, _, _, ok := relationTargetModel(&ast.MapType{}, structTypes); ok {
			t.Fatal("expected unsupported relation target model")
		}
		if got := resolveBelongsToForeignKey("Owner", "foreignKey:AccountID"); got != "AccountID" {
			t.Fatalf("resolveBelongsToForeignKey() = %q", got)
		}
		if got := resolveBelongsToForeignKey("Owner", ""); got != "OwnerID" {
			t.Fatalf("resolveBelongsToForeignKey default = %q", got)
		}
		if got := resolveColumnName("OwnerID", "column: owner_id "); got != "owner_id" {
			t.Fatalf("resolveColumnName() = %q", got)
		}
		if !allowCreateWritableField("ID", "id", "string", "") {
			t.Fatal("expected string id field to be creatable")
		}
		if allowCreateWritableField("ID", "id", "uint", "") {
			t.Fatal("expected integer id field to be skipped for create")
		}
		if !allowCreateWritableField("ID", "id", "uint", "autoincrement:false") {
			t.Fatal("expected explicit non-autoincrement id to be creatable")
		}
		if !allowUpdateWritableField("Name", "name", "") || allowUpdateWritableField("ID", "id", "") {
			t.Fatal("unexpected update writable field behavior")
		}
		if !isRenderableFieldType(&ast.ParenExpr{X: &ast.Ident{Name: "string"}}, imports) {
			t.Fatal("expected paren expr to be renderable")
		}
		if isRenderableFieldType(&ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "Read"}}}}}}, imports) {
			t.Fatal("expected non-empty interface to be rejected")
		}
		if isEmbeddedGormModel(&ast.Ident{Name: "Model"}, imports) {
			t.Fatal("expected non-selector embedded model to be rejected")
		}

		spec, ok := buildStructModelSpec(token.NewFileSet(), nil, structTypes, imports)
		if ok || !reflect.DeepEqual(spec, structModelSpec{}) {
			t.Fatalf("expected nil struct spec to fail, got %+v ok=%v", spec, ok)
		}
	})

	t.Run("execute text template propagates execution error", func(t *testing.T) {
		_, err := executeTextTemplate(`{{if call .Broken}}ok{{end}}`, struct{ Broken func() (bool, error) }{
			Broken: func() (bool, error) { return false, errors.New("boom") },
		})
		if err == nil || !strings.Contains(err.Error(), "execute template") {
			t.Fatalf("expected execute template error, got %v", err)
		}
	})
}
