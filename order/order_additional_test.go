package order

import (
	"testing"

	"github.com/shijl0925/go-toolkits/gormx"
)

func TestSortSchemaAndResolveEdges(t *testing.T) {
	t.Parallel()

	var nilSchema *SortSchema
	if nilSchema.Allow("name") != nil {
		t.Fatal("expected Allow on nil schema to stay nil")
	}

	schema := (&SortSchema{}).Allow("alias")
	if schema.allowed["alias"] != "alias" {
		t.Fatalf("expected alias to map to itself, got %+v", schema.allowed)
	}

	if _, err := ResolveSort("name", nil); err == nil {
		t.Fatal("expected ResolveSort to require a schema")
	}
}

func TestResolveOrderAdditionalErrors(t *testing.T) {
	t.Parallel()

	t.Run("non string target", func(t *testing.T) {
		input := &struct {
			Sort int `order:"name"`
		}{Sort: 1}
		if _, err := ResolveOrder(input); err == nil {
			t.Fatal("expected non-string order target error")
		}
	})

	t.Run("struct without exported sort", func(t *testing.T) {
		type embedded struct{ sort string }
		input := &struct {
			embedded `order:"name"`
		}{embedded: embedded{sort: "name"}}
		if _, err := ResolveOrder(input); err == nil {
			t.Fatal("expected missing Sort field error")
		}
	})

	t.Run("empty field in tag", func(t *testing.T) {
		input := &struct {
			Sort string `order:"name| "`
		}{Sort: "name"}
		if _, err := ResolveOrder(input); err == nil {
			t.Fatal("expected invalid order tag error")
		}
	})
}

func TestApplySortAndApplyOrderExecuteBranches(t *testing.T) {
	t.Parallel()

	query, _ := gormx.NewQuery[struct{}]()
	if err := ApplySort(query, "name,-created_at", NewSortSchema("name", "created_at")); err != nil {
		t.Fatalf("ApplySort: %v", err)
	}
	if err := ApplyOrder(query, &taggedSortInput{Sort: "name,-created"}); err != nil {
		t.Fatalf("ApplyOrder: %v", err)
	}

	var input *taggedSortInput
	if fields, err := ResolveOrder(input); err != nil || len(fields) != 0 {
		t.Fatalf("ResolveOrder(nil pointer) = (%+v, %v)", fields, err)
	}
}

func TestApplyDBNilQueryReturnsNil(t *testing.T) {
	t.Parallel()

	db, err := ApplyDB(nil, &taggedSortInput{Sort: "name"})
	if err != nil {
		t.Fatalf("ApplyDB(nil) error = %v", err)
	}
	if db != nil {
		t.Fatalf("expected nil db, got %#v", db)
	}
}
