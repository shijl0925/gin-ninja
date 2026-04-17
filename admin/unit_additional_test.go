package admin

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminHelperCoverage(t *testing.T) {
	t.Run("time parsing and tag splitting", func(t *testing.T) {
		for _, raw := range []string{
			"2026-04-17T04:57:29Z",
			"2026-04-17 04:57:29",
			"2026-04-17",
		} {
			if parsed, err := parseFlexibleTime(raw); err != nil || parsed.IsZero() {
				t.Fatalf("parseFlexibleTime(%q) = (%v, %v)", raw, parsed, err)
			}
		}
		if _, err := parseFlexibleTime("bad-time"); err == nil {
			t.Fatal("expected invalid time error")
		}
		if got := splitTagList(" name , , email "); len(got) != 2 || got[0] != "name" || got[1] != "email" {
			t.Fatalf("unexpected splitTagList result: %v", got)
		}
		if got := splitTagList("   "); got != nil {
			t.Fatalf("expected nil splitTagList result, got %v", got)
		}
	})

	t.Run("field parsing and assignment", func(t *testing.T) {
		type nested struct {
			Value *int
		}
		type sample struct {
			Count  int
			Score  float64
			Nested *nested
		}

		intField := &fieldMeta{Meta: FieldMeta{Name: "count", Column: "count"}, index: []int{0}, fieldType: reflect.TypeOf(int(0))}
		floatField := &fieldMeta{Meta: FieldMeta{Name: "score", Column: "score"}, index: []int{1}, fieldType: reflect.TypeOf(float64(0))}
		nestedField := &fieldMeta{Meta: FieldMeta{Name: "value", Column: "value"}, index: []int{2, 0}, fieldType: reflect.TypeOf(int(0))}
		timeField := &fieldMeta{Meta: FieldMeta{Name: "created_at", Column: "created_at"}, fieldType: reflect.TypeOf(time.Time{}), timeField: true}
		boolField := &fieldMeta{Meta: FieldMeta{Name: "active", Column: "active"}, fieldType: reflect.TypeOf(true)}
		badField := &fieldMeta{Meta: FieldMeta{Name: "unsupported"}, fieldType: reflect.TypeOf(struct{}{})}

		if value, err := intField.parseString("12"); err != nil || value.(int) != 12 {
			t.Fatalf("parseString int = (%v, %v)", value, err)
		}
		if value, err := floatField.parseString("2.5"); err != nil || value.(float64) != 2.5 {
			t.Fatalf("parseString float = (%v, %v)", value, err)
		}
		if value, err := boolField.parseString("true"); err != nil || value.(bool) != true {
			t.Fatalf("parseString bool = (%v, %v)", value, err)
		}
		if value, err := timeField.parseString("2026-04-17"); err != nil || value.(time.Time).IsZero() {
			t.Fatalf("parseString time = (%v, %v)", value, err)
		}
		if _, err := badField.parseString("anything"); err == nil {
			t.Fatal("expected unsupported filter type error")
		}

		if value, err := intField.decodeJSON(json.RawMessage("12")); err != nil || value.(int) != 12 {
			t.Fatalf("decodeJSON int = (%v, %v)", value, err)
		}
		if value, err := intField.decodeJSON(json.RawMessage("null")); err != nil || value != nil {
			t.Fatalf("decodeJSON null = (%v, %v)", value, err)
		}

		var target sample
		target.Score = 1.5
		if err := intField.setValue(reflect.ValueOf(&target).Elem(), int64(7)); err != nil {
			t.Fatalf("setValue convertible: %v", err)
		}
		if err := nestedField.setValue(reflect.ValueOf(&target).Elem(), 9); err != nil {
			t.Fatalf("setValue nested: %v", err)
		}
		if err := floatField.setValue(reflect.ValueOf(&target).Elem(), nil); err != nil {
			t.Fatalf("setValue nil: %v", err)
		}
		if err := intField.setValue(reflect.ValueOf(&target).Elem(), "bad"); err == nil {
			t.Fatal("expected incompatible assignment error")
		}
		if got := intField.value(reflect.ValueOf(target)); got != 7 {
			t.Fatalf("value() = %v, want 7", got)
		}
		if target.Nested == nil || target.Nested.Value == nil || *target.Nested.Value != 9 {
			t.Fatalf("expected nested pointer assignment, got %+v", target.Nested)
		}
	})
}

func TestAdminRuntimeHelpers(t *testing.T) {
	t.Run("persisted columns and non persisted values", func(t *testing.T) {
		view := &resolvedResource{
			fieldByName: map[string]*fieldMeta{},
			fields: []*fieldMeta{
				{Meta: FieldMeta{Name: "id", Column: "id"}, primaryKey: true, persisted: true},
				{Meta: FieldMeta{Name: "name", Column: "name"}, persisted: true},
				{Meta: FieldMeta{Name: "alias", Column: "name"}, persisted: true},
				{Meta: FieldMeta{Name: "role_ids"}, persisted: false},
				nil,
			},
		}
		view.fieldByName["name"] = view.fields[1]
		view.fieldByName["role_ids"] = view.fields[3]

		resource := &Resource{}
		columns := resource.persistedColumnsFor(view)
		if len(columns) != 1 || columns[0] != "name" {
			t.Fatalf("persistedColumnsFor() = %v, want [name]", columns)
		}
		if !resource.hasNonPersistedValues(view, map[string]any{"role_ids": []uint{1}}) {
			t.Fatal("expected non-persisted values to be detected")
		}
		if resource.hasNonPersistedValues(view, map[string]any{"name": "alice"}) {
			t.Fatal("expected persisted values to be ignored")
		}
	})

	t.Run("compose query scope", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open("file:admin-compose?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}

		if scope := composeQueryScope(nil, nil); scope != nil {
			t.Fatalf("expected nil query scope, got %v", scope)
		}

		scope := composeQueryScope([]string{"Owner", " "}, func(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
			return db.Where("name = ?", "alice")
		})
		scoped := scope(nil, db.Session(&gorm.Session{}))
		if len(scoped.Statement.Preloads) != 1 {
			t.Fatalf("expected preloads to be applied, got %v", scoped.Statement.Preloads)
		}
		if _, ok := scoped.Statement.Preloads["Owner"]; !ok {
			t.Fatalf("expected Owner preload, got %v", scoped.Statement.Preloads)
		}
		if _, ok := scoped.Statement.Clauses["WHERE"]; !ok {
			t.Fatalf("expected WHERE clause, got %v", scoped.Statement.Clauses)
		}

		scope = composeQueryScope([]string{"Owner"}, func(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
			return nil
		})
		scoped = scope(nil, db.Session(&gorm.Session{}))
		if _, ok := scoped.Statement.Preloads["Owner"]; !ok {
			t.Fatalf("expected preloads to remain when query scope returns nil, got %v", scoped.Statement.Preloads)
		}
	})
}

func TestAdminSiteRegistrationCoverage(t *testing.T) {
	site := NewSite()

	if err := site.Register(nil); err == nil {
		t.Fatal("expected Register(nil) to fail")
	}
	if err := site.RegisterModel(nil); err == nil {
		t.Fatal("expected RegisterModel(nil) to fail")
	}

	resource := &Resource{Name: "users", Model: autoResourceUser{}}
	if err := site.Register(resource); err != nil {
		t.Fatalf("Register(): %v", err)
	}
	if err := site.Register(&Resource{Name: "users", Model: autoResourceUser{}}); err == nil {
		t.Fatal("expected duplicate Register() to fail")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected MustRegister to panic")
			}
		}()
		NewSite().MustRegister(nil)
	}()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected MustRegisterModel to panic")
			}
		}()
		NewSite().MustRegisterModel(nil)
	}()
}

func TestAdminApplyFilterCoverage(t *testing.T) {
	type filterModel struct {
		ID        uint `gorm:"primaryKey"`
		Name      string
		Count     int
		CreatedAt time.Time
	}

	db, err := gorm.Open(sqlite.Open("file:admin-filters?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&filterModel{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	seed := []filterModel{
		{Name: "alpha", Count: 1, CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "beta", Count: 2, CreatedAt: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		{Name: "gamma", Count: 3, CreatedAt: time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	countField := &fieldMeta{Meta: FieldMeta{Name: "count", Column: "count"}, fieldType: reflect.TypeOf(int(0))}
	nameField := &fieldMeta{Meta: FieldMeta{Name: "name", Column: "name"}, fieldType: reflect.TypeOf("")}
	createdAtField := &fieldMeta{Meta: FieldMeta{Name: "created_at", Column: "created_at"}, fieldType: reflect.TypeOf(time.Time{}), timeField: true}

	cases := []struct {
		name   string
		field  *fieldMeta
		query  url.Values
		want   int64
		hasErr bool
	}{
		{name: "eq", field: countField, query: url.Values{"count": {"2"}}, want: 1},
		{name: "ne", field: countField, query: url.Values{"count__ne": {"2"}}, want: 2},
		{name: "gt", field: countField, query: url.Values{"count__gt": {"1"}}, want: 2},
		{name: "gte", field: countField, query: url.Values{"count__gte": {"2"}}, want: 2},
		{name: "lt", field: countField, query: url.Values{"count__lt": {"3"}}, want: 2},
		{name: "lte", field: countField, query: url.Values{"count__lte": {"2"}}, want: 2},
		{name: "like", field: nameField, query: url.Values{"name__like": {"a"}}, want: 3},
		{name: "in", field: countField, query: url.Values{"count__in": {"1,3"}}, want: 2},
		{name: "from_to", field: createdAtField, query: url.Values{"created_at__from": {"2026-04-02"}, "created_at__to": {"2026-04-03"}}, want: 2},
		{name: "bad filter", field: countField, query: url.Values{"count__gt": {"bad"}}, hasErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := applyFilter(db.Model(&filterModel{}), tc.query, tc.field)
			if tc.hasErr {
				if err == nil || !strings.Contains(err.Error(), "BAD_FILTER") {
					t.Fatalf("applyFilter() error = %v, want BAD_FILTER", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("applyFilter(): %v", err)
			}
			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Count(): %v", err)
			}
			if count != tc.want {
				t.Fatalf("Count() = %d, want %d", count, tc.want)
			}
		})
	}
}
