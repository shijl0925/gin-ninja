package filter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shijl0925/go-toolkits/gormx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestParseNilInput(t *testing.T) {
	t.Parallel()

	clauses, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) error = %v", err)
	}
	if len(clauses) != 0 {
		t.Fatalf("expected no clauses, got %+v", clauses)
	}
}

func TestApplyAndBuildOptionEdgeCases(t *testing.T) {
	t.Parallel()

	query, _ := gormx.NewQuery[userRecord]()
	if err := applySingleClause(query, Clause{}); err == nil || !strings.Contains(err.Error(), "missing fields") {
		t.Fatalf("expected missing field error, got %v", err)
	}
	if _, err := buildOption(Clause{}); err == nil || !strings.Contains(err.Error(), "missing fields") {
		t.Fatalf("expected missing field error, got %v", err)
	}
	if err := applySingleClause(query, Clause{Field: "id", Op: Operator("bad"), Value: 1}); err == nil || !strings.Contains(err.Error(), "unsupported filter operator") {
		t.Fatalf("expected unsupported operator error, got %v", err)
	}
	if _, err := buildExpression("id", OpIn, []int{1, 2}); err != nil {
		t.Fatalf("buildExpression(IN): %v", err)
	}
	for _, op := range []Operator{OpNe, OpGt, OpGe, OpLt, OpLe} {
		if _, err := buildExpression("id", op, 1); err != nil {
			t.Fatalf("buildExpression(%s): %v", op, err)
		}
	}
	if isEmptyValue(reflect.ValueOf(map[string]int{"id": 1})) {
		t.Fatal("expected non-empty map to be detected")
	}
	if isEmptyValue(reflect.ValueOf(false)) {
		t.Fatal("expected false bool to be treated as a meaningful value")
	}
}

func TestApplyDBEdgeCases(t *testing.T) {
	t.Parallel()

	if _, err := applyDBClause(nil, Clause{}); err == nil || !strings.Contains(err.Error(), "missing fields") {
		t.Fatalf("expected missing field error, got %v", err)
	}
	if _, err := applyDBClause(nil, Clause{
		Fields:   []string{"name", "email"},
		Op:       OpLike,
		Value:    "alice",
		Combiner: "and",
	}); err == nil || !strings.Contains(err.Error(), "unsupported filter combiner") {
		t.Fatalf("expected unsupported combiner error, got %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := applyDBClause(db.Model(&userRecord{}), Clause{}); err == nil || !strings.Contains(err.Error(), "missing fields") {
		t.Fatalf("expected missing field error with db, got %v", err)
	}
}

func TestFilterHelperSuccessBranches(t *testing.T) {
	t.Parallel()

	type input struct {
		Search string `filter:"name|email,like"`
		IDs    []int  `filter:"id,in"`
		Active bool   `filter:"active,eq"`
	}

	clauses, err := Parse(input{Search: "ali", IDs: []int{1, 2}, Active: false})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %+v", clauses)
	}

	opts, err := BuildOptions(input{Search: "ali", IDs: []int{1, 2}})
	if err != nil {
		t.Fatalf("BuildOptions: %v", err)
	}
	if len(opts) != 3 {
		t.Fatalf("expected 3 built options, got %d", len(opts))
	}

	query, _ := gormx.NewQuery[userRecord]()
	if err := Apply(query, input{Active: false}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := Apply[userRecord](nil, input{Active: true}); err != nil {
		t.Fatalf("Apply nil query: %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	filtered, err := ApplyDB(db.Model(&userRecord{}), input{Search: "ali"})
	if err != nil {
		t.Fatalf("ApplyDB: %v", err)
	}
	if filtered == nil {
		t.Fatal("expected filtered db")
	}
	if fields, op, combiner, err := parseTag("name,email,extra", reflect.StructField{Name: "Search"}); err == nil {
		t.Fatalf("expected too-many-parts tag error, got fields=%v op=%v combiner=%v", fields, op, combiner)
	}
}

func TestFilterSwitchCoverage(t *testing.T) {
	t.Parallel()

	query, _ := gormx.NewQuery[userRecord]()
	for _, op := range []Operator{OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpLike, OpIn} {
		value := any(1)
		if op == OpLike {
			value = "alice"
		}
		if op == OpIn {
			value = []int{1, 2}
		}
		if err := applySingleClause(query, Clause{Field: "id", Op: op, Value: value}); err != nil {
			t.Fatalf("applySingleClause(%s): %v", op, err)
		}
		if _, err := buildOption(Clause{Field: "id", Op: op, Value: value}); err != nil {
			t.Fatalf("buildOption(%s): %v", op, err)
		}
	}

	if _, err := buildExpression("name", OpLike, 123); err == nil || !strings.Contains(err.Error(), "requires a string value") {
		t.Fatalf("expected like type error, got %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, clause := range []Clause{
		{Field: "id", Op: OpEq, Value: 1},
		{Field: "id", Op: OpNe, Value: 2},
		{Field: "id", Op: OpGt, Value: 0},
		{Field: "id", Op: OpGe, Value: 0},
		{Field: "id", Op: OpLt, Value: 10},
		{Field: "id", Op: OpLe, Value: 10},
		{Field: "name", Op: OpLike, Value: "ali"},
		{Field: "id", Op: OpIn, Value: []int{1, 2}},
		{Fields: []string{"name", "email"}, Op: OpLike, Value: "ali", Combiner: CombinerOr},
	} {
		if _, err := applyDBClause(db, clause); err != nil {
			t.Fatalf("applyDBClause(%+v): %v", clause, err)
		}
	}

	cases := []reflect.Value{
		reflect.ValueOf(""),
		reflect.ValueOf("value"),
		reflect.ValueOf([]int{}),
		reflect.ValueOf([]int{1}),
		reflect.ValueOf(0),
		reflect.ValueOf(2),
		reflect.ValueOf(0.0),
		reflect.ValueOf(1.5),
		reflect.ValueOf(struct{ Name string }{}),
		reflect.ValueOf(struct{ Name string }{Name: "alice"}),
	}
	results := []bool{true, false, true, false, true, false, true, false, true, false}
	for i, value := range cases {
		if got := isEmptyValue(value); got != results[i] {
			t.Fatalf("isEmptyValue(%v) = %v, want %v", value.Interface(), got, results[i])
		}
	}
}
