package order

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSortSchemaAndDeclarativeOrderErrorBranches(t *testing.T) {
	var nilSchema *SortSchema
	if nilSchema.Allow("name") != nil {
		t.Fatal("expected nil Allow receiver to stay nil")
	}
	if nilSchema.AllowExpression("rank", "count(*)") != nil {
		t.Fatal("expected nil AllowExpression receiver to stay nil")
	}

	schema := NewSortSchema()
	schema.Allow(" ", "name")
	if _, err := ResolveSort("name", schema); err == nil || !strings.Contains(err.Error(), "empty field") {
		t.Fatalf("expected schema empty field error, got %v", err)
	}

	schema = NewSortSchema()
	schema.Allow("name", "name desc")
	if _, err := ResolveSort("name", schema); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected unsafe schema error, got %v", err)
	}

	if fields := ParseSort(" , +name, -created_at, - , "); !reflect.DeepEqual(fields, []SortField{{Name: "name"}, {Name: "created_at", Desc: true}}) {
		t.Fatalf("ParseSort = %+v", fields)
	}
	if _, err := ResolveOrder(12); err == nil || !strings.Contains(err.Error(), "struct") {
		t.Fatalf("expected non-struct order input error, got %v", err)
	}
	if _, err := ResolveSort("name", NewSortSchema()); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected empty schema error, got %v", err)
	}
	if _, err := ResolveSort("missing", NewSortSchema("name")); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported sort field error, got %v", err)
	}
	if _, err := parseOrderTagSchema("name|", reflect.StructField{Name: "Sort"}); err == nil || !strings.Contains(err.Error(), "empty field") {
		t.Fatalf("expected empty order tag error, got %v", err)
	}
	if _, err := parseOrderTagSchema("alias:", reflect.StructField{Name: "Sort"}); err == nil || !strings.Contains(err.Error(), "empty field") {
		t.Fatalf("expected empty alias target error, got %v", err)
	}

	type pointerInput struct {
		Sort *string `order:"name"`
	}
	if fields, err := ResolveOrder(pointerInput{}); err != nil || len(fields) != 0 {
		t.Fatalf("nil pointer order field = %+v, %v", fields, err)
	}
	type embeddedBad struct {
		Embedded struct {
			Sort int `order:"name"`
		}
	}
	typ := reflect.TypeOf(embeddedBad{})
	field := typ.Field(0)
	if _, _, err := resolveTaggedOrder(field, reflect.ValueOf(embeddedBad{Embedded: struct {
		Sort int `order:"name"`
	}{Sort: 1}}).Field(0), "name"); err == nil {
		t.Fatal("expected direct embedded bad field error")
	}
	type inputWithBad struct {
		Sort string `order:"name"`
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if _, err := ApplyDB(db, inputWithBad{Sort: "missing"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected ApplyDB unsupported sort error, got %v", err)
	}
}
