package filter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shijl0925/go-toolkits/gormx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type embeddedFilter struct {
	IsAdmin *bool `filter:"is_admin,eq"`
}

type boolValueFilter struct {
	IsAdmin bool `filter:"is_admin,eq"`
}

type listInput struct {
	embeddedFilter
	Search string `filter:"name|email,like"`
	AgeMin int    `filter:"age,ge"`
}

type invalidMultiFieldInput struct {
	Search string `filter:"name|,like"`
}

type invalidOperatorInput struct {
	Search string `filter:"name,contains"`
}

type userRecord struct {
	ID      uint
	Name    string
	Email   string
	Age     int
	IsAdmin bool
}

func TestParse(t *testing.T) {
	admin := true
	clauses, err := Parse(&listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Search:         "alice",
		AgeMin:         18,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %d", len(clauses))
	}
	if clauses[0].Field != "is_admin" || clauses[0].Op != OpEq || clauses[0].Value != true {
		t.Fatalf("unexpected clause[0]: %+v", clauses[0])
	}
	if clauses[1].Field != "name|email" || clauses[1].Op != OpLike || clauses[1].Value != "alice" || clauses[1].Combiner != CombinerOr {
		t.Fatalf("unexpected clause[1]: %+v", clauses[1])
	}
	if len(clauses[1].Fields) != 2 || clauses[1].Fields[0] != "name" || clauses[1].Fields[1] != "email" {
		t.Fatalf("unexpected clause[1] fields: %+v", clauses[1])
	}
	if clauses[2].Field != "age" || clauses[2].Op != OpGe || clauses[2].Value != 18 {
		t.Fatalf("unexpected clause[2]: %+v", clauses[2])
	}
}

func TestParseSkipsZeroValues(t *testing.T) {
	clauses, err := Parse(&listInput{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 0 {
		t.Fatalf("expected no clauses, got %+v", clauses)
	}
}

func TestParseKeepsFalseBoolPointers(t *testing.T) {
	admin := false
	clauses, err := Parse(&listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 1 || clauses[0].Value != false {
		t.Fatalf("expected false bool clause, got %+v", clauses)
	}
}

func TestParseKeepsFalseBoolValues(t *testing.T) {
	clauses, err := Parse(&boolValueFilter{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 1 || clauses[0].Value != false {
		t.Fatalf("expected false bool value clause, got %+v", clauses)
	}
}

func TestParseRejectsInvalidMultiFieldTag(t *testing.T) {
	_, err := Parse(&invalidMultiFieldInput{Search: "alice"})
	if err == nil || !strings.Contains(err.Error(), "empty field name") {
		t.Fatalf("expected invalid multi-field tag error, got %v", err)
	}
}

func TestParseRejectsInvalidOperator(t *testing.T) {
	_, err := Parse(&invalidOperatorInput{Search: "alice"})
	if err == nil || !strings.Contains(err.Error(), "unsupported operator") {
		t.Fatalf("expected invalid operator error, got %v", err)
	}
}

func TestParseTagBoundaryCases(t *testing.T) {
	field := reflect.TypeOf(struct {
		Search string `filter:"name|email,like"`
	}{}).Field(0)

	cases := []struct {
		name       string
		tag        string
		wantFields []string
		wantOp     Operator
		wantErr    string
	}{
		{name: "doc example", tag: "name|email,like", wantFields: []string{"name", "email"}, wantOp: OpLike},
		{name: "trim whitespace", tag: " name | email , like ", wantFields: []string{"name", "email"}, wantOp: OpLike},
		{name: "default operator", tag: "status", wantFields: []string{"status"}, wantOp: OpEq},
		{name: "empty field", tag: " |email,like", wantErr: "empty field name"},
		{name: "extra comma", tag: "name,email,like", wantErr: "must be in the form"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields, op, _, err := parseTag(tc.tag, field)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTag: %v", err)
			}
			if op != tc.wantOp {
				t.Fatalf("expected op %q, got %q", tc.wantOp, op)
			}
			if len(fields) != len(tc.wantFields) {
				t.Fatalf("expected fields %+v, got %+v", tc.wantFields, fields)
			}
			for i := range fields {
				if fields[i] != tc.wantFields[i] {
					t.Fatalf("expected fields %+v, got %+v", tc.wantFields, fields)
				}
			}
		})
	}
}

func TestBuildOptionsMultiFieldLikeUsesORSemantics(t *testing.T) {
	setupFilterTestDB(t)

	if err := gormx.GetDb().Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	opts, err := BuildOptions(&listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Search:         "example.com",
	})
	if err != nil {
		t.Fatalf("BuildOptions: %v", err)
	}

	var got []userRecord
	if err := gormx.GetDb(opts...).Find(&got).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 1 || got[0].Email != "bob@example.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestApplyRejectsMultiFieldFilters(t *testing.T) {
	query, _ := gormx.NewQuery[userRecord]()
	err := Apply(query, &listInput{Search: "alice"})
	if err == nil || !strings.Contains(err.Error(), "BuildOptions") {
		t.Fatalf("expected multi-field apply error, got %v", err)
	}
}

func TestApplySingleFieldFilters(t *testing.T) {
	setupFilterTestDB(t)

	if err := gormx.GetDb().Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	query, _ := gormx.NewQuery[userRecord]()
	if err := Apply(query, &embeddedFilter{IsAdmin: &admin}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var got []userRecord
	if err := query.Find(&got); err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func setupFilterTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	gormx.Init(db)

	if err := db.AutoMigrate(&userRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
