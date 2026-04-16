package filter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shijl0925/go-toolkits/gormx"
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
}
