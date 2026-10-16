package filter

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type customExpressionInput struct {
	embeddedFilter
	Name  string
	Email string
}

func (in customExpressionInput) FilterExpression() clause.Expression {
	exprs := make([]clause.Expression, 0, 2)
	if in.Name != "" {
		exprs = append(exprs, clause.Like{Column: clause.Column{Name: "name"}, Value: "%" + in.Name + "%"})
	}
	if in.Email != "" {
		exprs = append(exprs, clause.Like{Column: clause.Column{Name: "email"}, Value: "%" + in.Email + "%"})
	}
	if len(exprs) == 0 {
		return nil
	}
	return clause.Or(exprs...)
}

type nilExpressionInput struct {
	embeddedFilter
}

func (nilExpressionInput) FilterExpression() clause.Expression {
	return nil
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

func TestParseRejectsNonStructInput(t *testing.T) {
	if _, err := Parse("alice"); err == nil || !strings.Contains(err.Error(), "struct or pointer to struct") {
		t.Fatalf("expected invalid input error, got %v", err)
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
		{name: "table column", tag: "users.email,eq", wantFields: []string{"users.email"}, wantOp: OpEq},
		{name: "empty field", tag: " |email,like", wantErr: "empty field name"},
		{name: "extra comma", tag: "name,email,like", wantErr: "must be in the form"},
		{name: "sql fragment", tag: "name;drop,eq", wantErr: "unsafe field name"},
		{name: "quoted field", tag: "`name`,eq", wantErr: "unsafe field name"},
		{name: "multiple dots", tag: "users.profile.email,eq", wantErr: "unsafe field name"},
		{name: "leading digit", tag: "1name,eq", wantErr: "unsafe field name"},
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

func TestToColumnSupportsSafeTableColumn(t *testing.T) {
	column := toColumn("users.email")
	if column.Table != "users" || column.Name != "email" {
		t.Fatalf("unexpected table column: %+v", column)
	}
}

func TestApplyDBMultiFieldLikeUsesORSemantics(t *testing.T) {
	db := setupFilterTestDB(t)

	if err := db.Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	query, err := ApplyDB(db.Model(&userRecord{}), &listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Search:         "example.com",
	})
	if err != nil {
		t.Fatalf("ApplyDB: %v", err)
	}

	var got []userRecord
	if err := query.Find(&got).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 1 || got[0].Email != "bob@example.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestApplyDBAppliesCustomFilterExpressionFromInput(t *testing.T) {
	db := setupFilterTestDB(t)

	if err := db.Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	query, err := ApplyDB(db.Model(&userRecord{}), customExpressionInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Name:           "Alice",
		Email:          "sample.com",
	})
	if err != nil {
		t.Fatalf("ApplyDB: %v", err)
	}

	var got []userRecord
	if err := query.Find(&got).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 1 || got[0].Email != "carol@sample.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestApplyDB(t *testing.T) {
	db := setupFilterTestDB(t)

	if err := db.Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	query, err := ApplyDB(db.Model(&userRecord{}), &listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Search:         "example.com",
	})
	if err != nil {
		t.Fatalf("ApplyDB: %v", err)
	}

	var got []userRecord
	if err := query.Find(&got).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 1 || got[0].Email != "bob@example.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestApplyDBAppliesCustomFilterExpression(t *testing.T) {
	db := setupFilterTestDB(t)

	if err := db.Create([]userRecord{
		{Name: "Alice", Email: "alice@example.com", Age: 20, IsAdmin: false},
		{Name: "Bob", Email: "bob@example.com", Age: 21, IsAdmin: true},
		{Name: "Carol", Email: "carol@sample.com", Age: 22, IsAdmin: true},
	}).Error; err != nil {
		t.Fatalf("seed db: %v", err)
	}

	admin := true
	query, err := ApplyDB(db.Model(&userRecord{}), customExpressionInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Name:           "Alice",
		Email:          "sample.com",
	})
	if err != nil {
		t.Fatalf("ApplyDB: %v", err)
	}

	var got []userRecord
	if err := query.Find(&got).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(got) != 1 || got[0].Email != "carol@sample.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestApplyDBNilQueryReturnsNil(t *testing.T) {
	db, err := ApplyDB(nil, &embeddedFilter{})
	if err != nil {
		t.Fatalf("ApplyDB(nil) error = %v", err)
	}
	if db != nil {
		t.Fatalf("expected nil db, got %#v", db)
	}
}

func TestBuildOptionAndHelperEdges(t *testing.T) {
	t.Run("unsupported combiner", func(t *testing.T) {
		_, err := applyDBClause(nil, Clause{
			Fields:   []string{"name", "email"},
			Op:       OpLike,
			Value:    "alice",
			Combiner: "and",
		})
		if err == nil || !strings.Contains(err.Error(), "unsupported filter combiner") {
			t.Fatalf("expected unsupported combiner error, got %v", err)
		}
	})

	t.Run("like requires string", func(t *testing.T) {
		_, err := buildExpression("name", OpLike, 123)
		if err == nil || !strings.Contains(err.Error(), "requires a string value") {
			t.Fatalf("expected like string error, got %v", err)
		}
	})

	t.Run("column and values helpers", func(t *testing.T) {
		column := toColumn("users.email")
		if column.Table != "users" || column.Name != "email" {
			t.Fatalf("unexpected column %+v", column)
		}

		values := toValues([]int{1, 2, 3})
		if len(values) != 3 || values[0] != 1 || values[2] != 3 {
			t.Fatalf("unexpected slice values %+v", values)
		}

		scalar := toValues("alice")
		if len(scalar) != 1 || scalar[0] != "alice" {
			t.Fatalf("unexpected scalar values %+v", scalar)
		}
	})

	t.Run("empty value helpers", func(t *testing.T) {
		if !isEmptyValue(reflect.ValueOf("")) {
			t.Fatal("expected empty string to be empty")
		}
		if isEmptyValue(reflect.ValueOf(false)) {
			t.Fatal("expected false bool to remain non-empty")
		}
	})

	t.Run("nil custom expression is ignored", func(t *testing.T) {
		admin := true
		base := setupFilterTestDB(t)
		db, err := ApplyDB(base.Model(&userRecord{}), nilExpressionInput{embeddedFilter: embeddedFilter{IsAdmin: &admin}})
		if err != nil {
			t.Fatalf("ApplyDB: %v", err)
		}
		if db == nil {
			t.Fatal("expected db to be returned")
		}
	})
}

func setupFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&userRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
