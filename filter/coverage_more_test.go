package filter

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterParseAndBuilderErrorBranches(t *testing.T) {
	if set, err := Parse((*struct {
		Name string `filter:"name"`
	})(nil)); err != nil || len(set) != 0 {
		t.Fatalf("Parse nil pointer = %+v, %v", set, err)
	}
	if _, err := Parse(12); err == nil || !strings.Contains(err.Error(), "struct") {
		t.Fatalf("expected non-struct parse error, got %v", err)
	}

	type badPrivate struct {
		name string `filter:"name"`
	}
	if set, err := Parse(badPrivate{name: "alice"}); err != nil || len(set) != 0 {
		t.Fatalf("unexported filter field = %+v, %v", set, err)
	}

	field, _ := reflect.TypeOf(struct {
		Name string `filter:"name"`
	}{}).FieldByName("Name")
	if _, _, _, err := parseTag("a,b,c", field); err == nil || !strings.Contains(err.Error(), "must be in the form") {
		t.Fatalf("expected malformed tag error, got %v", err)
	}
	if _, _, _, err := parseTag("", field); err == nil || !strings.Contains(err.Error(), "must include") {
		t.Fatalf("expected empty field tag error, got %v", err)
	}
	if _, _, _, err := parseTag("name|,eq", field); err == nil || !strings.Contains(err.Error(), "empty field") {
		t.Fatalf("expected empty multi field error, got %v", err)
	}

	if _, err := applyDBClause(nil, Clause{}); err == nil {
		t.Fatal("expected missing fields apply error")
	}
	if _, err := applyDBClause(nil, Clause{Fields: []string{"name", "email"}, Combiner: "and"}); err == nil || !strings.Contains(err.Error(), "combiner") {
		t.Fatalf("expected unsupported combiner error, got %v", err)
	}
	if _, err := buildExpression("name", OpLike, 12); err == nil || !strings.Contains(err.Error(), "requires a string") {
		t.Fatalf("expected like type error, got %v", err)
	}
	type badLike struct {
		Name int `filter:"name,like"`
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if _, err := ApplyDB(db, 12); err == nil || !strings.Contains(err.Error(), "struct") {
		t.Fatalf("expected ApplyDB parse error, got %v", err)
	}
	if _, err := ApplyDB(db, badLike{Name: 12}); err == nil || !strings.Contains(err.Error(), "requires a string") {
		t.Fatalf("expected ApplyDB clause error, got %v", err)
	}
	if _, err := buildExpression("name", Operator("bad"), "alice"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported op error, got %v", err)
	}

	values := toValues([2]int{1, 2})
	if !reflect.DeepEqual(values, []any{1, 2}) {
		t.Fatalf("toValues(array) = %+v", values)
	}
	if got := toColumn("public.users.name"); got.Table != "" || got.Name != "public.users.name" {
		t.Fatalf("expected dotted field with more than one dot to stay whole, got %+v", got)
	}
}
