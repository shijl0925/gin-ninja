package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

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

func TestAdminRuntimeEdgeCoverage(t *testing.T) {
	t.Run("decode payload and restore request body", func(t *testing.T) {
		makeCtx := func(body io.ReadCloser) *ninja.Context {
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Body = body
			ginCtx.Request = req
			return &ninja.Context{Context: ginCtx}
		}

		nameField := &fieldMeta{
			Meta:      FieldMeta{Name: "name", Column: "name", Create: true, Update: true, Required: true},
			fieldType: reflect.TypeOf(""),
			persisted: true,
			index:     []int{0},
		}
		ageField := &fieldMeta{
			Meta:      FieldMeta{Name: "age", Column: "age", Create: false, Update: false},
			fieldType: reflect.TypeOf(int(0)),
			persisted: true,
			index:     []int{1},
		}
		view := &resolvedResource{
			fields:      []*fieldMeta{nameField, ageField},
			fieldByName: map[string]*fieldMeta{"name": nameField, "age": ageField},
		}
		resource := &Resource{}

		ctx := makeCtx(io.NopCloser(strings.NewReader(`{"name":"Alice"}`)))
		values, err := resource.decodeWritePayloadFor(view, ctx, fieldModeCreate)
		if err != nil {
			t.Fatalf("decodeWritePayloadFor(): %v", err)
		}
		if values["name"] != "Alice" {
			t.Fatalf("unexpected decoded values: %+v", values)
		}
		restored, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			t.Fatalf("ReadAll(restored body): %v", err)
		}
		if string(restored) != `{"name":"Alice"}` {
			t.Fatalf("expected request body to be restored, got %q", restored)
		}

		ctx = makeCtx(io.NopCloser(strings.NewReader("   ")))
		values, err = resource.decodeWritePayloadFor(view, ctx, fieldModeCreate)
		if err != nil || len(values) != 0 {
			t.Fatalf("expected empty payload to decode to empty map, got values=%v err=%v", values, err)
		}

		ctx = makeCtx(io.NopCloser(strings.NewReader("{")))
		if _, err := resource.decodeWritePayloadFor(view, ctx, fieldModeCreate); err == nil {
			t.Fatal("expected invalid JSON error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "INVALID_JSON" {
			t.Fatalf("expected INVALID_JSON error, got %T %v", err, err)
		}

		ctx = makeCtx(io.NopCloser(strings.NewReader(`{"unknown":"value"}`)))
		if _, err := resource.decodeWritePayloadFor(view, ctx, fieldModeCreate); err == nil {
			t.Fatal("expected unknown field error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `unknown field "unknown"`) {
			t.Fatalf("expected BAD_REQUEST unknown field, got %T %v", err, err)
		}

		ctx = makeCtx(io.NopCloser(strings.NewReader(`{"age":1}`)))
		if _, err := resource.decodeWritePayloadFor(view, ctx, fieldModeUpdate); err == nil {
			t.Fatal("expected non-writable field error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `field "age" is not writable`) {
			t.Fatalf("expected BAD_REQUEST not writable, got %T %v", err, err)
		}

		ctx = makeCtx(io.NopCloser(strings.NewReader(`{"name":1}`)))
		if _, err := resource.decodeWritePayloadFor(view, ctx, fieldModeCreate); err == nil {
			t.Fatal("expected field decode error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `field "name"`) {
			t.Fatalf("expected BAD_REQUEST field decode message, got %T %v", err, err)
		}

		ctx = makeCtx(errReadCloser{})
		if _, err := readAndRestoreRequestBody(ctx); err == nil || !strings.Contains(err.Error(), "read failed") {
			t.Fatalf("expected read failure, got %v", err)
		}
	})

	t.Run("required fields update columns and conflict normalization", func(t *testing.T) {
		type sample struct {
			Name  string
			Alias string
			Age   int
			Meta  *struct {
				Code string
			}
		}

		requiredField := &fieldMeta{
			Meta:      FieldMeta{Name: "name", Column: "name", Create: true, Required: true},
			fieldType: reflect.TypeOf(""),
			persisted: true,
			index:     []int{0},
		}
		aliasField := &fieldMeta{
			Meta:      FieldMeta{Name: "alias", Column: "name", Update: true},
			fieldType: reflect.TypeOf(""),
			persisted: true,
			index:     []int{1},
		}
		ageField := &fieldMeta{
			Meta:      FieldMeta{Name: "age", Column: "age", Update: true},
			fieldType: reflect.TypeOf(int(0)),
			persisted: true,
			index:     []int{2},
		}
		nestedField := &fieldMeta{
			Meta:      FieldMeta{Name: "code", Column: "code"},
			fieldType: reflect.TypeOf(""),
			index:     []int{3, 0},
		}
		view := &resolvedResource{
			fields:      []*fieldMeta{requiredField, aliasField, ageField},
			fieldByName: map[string]*fieldMeta{"name": requiredField, "alias": aliasField, "age": ageField},
		}
		resource := &Resource{}

		if err := resource.validateRequiredFor(view, map[string]any{}, fieldModeCreate); err == nil {
			t.Fatal("expected missing required field error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `field "name" is required`) {
			t.Fatalf("expected BAD_REQUEST required error, got %T %v", err, err)
		}
		if err := resource.validateRequiredFor(view, map[string]any{}, fieldModeUpdate); err != nil {
			t.Fatalf("expected update validation to skip required check, got %v", err)
		}
		if err := resource.validateRequiredFor(view, map[string]any{"name": "Alice"}, fieldModeCreate); err != nil {
			t.Fatalf("expected provided required field to pass, got %v", err)
		}

		before := reflect.ValueOf(sample{Name: "Alice", Alias: "old", Age: 18})
		after := reflect.ValueOf(sample{Name: "Alice", Alias: "new", Age: 18})
		columns, err := resource.updateColumnsFor(view, before, after)
		if err != nil {
			t.Fatalf("updateColumnsFor(): %v", err)
		}
		if !reflect.DeepEqual(columns, []string{"name"}) {
			t.Fatalf("expected duplicate column changes to collapse to one entry, got %v", columns)
		}

		if value, ok := resource.fieldValue(reflect.ValueOf(sample{}), nestedField); !ok || value != nil {
			t.Fatalf("expected nil nested pointer value, got value=%v ok=%v", value, ok)
		}
		if value, ok := resource.fieldValue(reflect.Value{}, nestedField); ok || value != nil {
			t.Fatalf("expected invalid value to report ok=false, got value=%v ok=%v", value, ok)
		}
		if value, ok := resource.fieldValue(reflect.ValueOf(sample{}), nil); ok || value != nil {
			t.Fatalf("expected nil field to report ok=false, got value=%v ok=%v", value, ok)
		}
		if queryColumn(nil) != "" {
			t.Fatal("expected nil queryColumn to be empty")
		}

		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}
		if err := db.AutoMigrate(&adminUser{}); err != nil {
			t.Fatalf("AutoMigrate: %v", err)
		}
		if err := db.Create(&adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}).Error; err != nil {
			t.Fatalf("Create(user): %v", err)
		}
		orm.Init(db)

		adminResource := &Resource{Name: "users", Model: adminUser{}}
		if err := adminResource.prepare(); err != nil {
			t.Fatalf("prepare(): %v", err)
		}
		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginCtx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
		ctx := &ninja.Context{Context: ginCtx}

		conflictErr := adminResource.normalizeWriteError(ctx, ActionCreate, reflect.ValueOf(adminUser{
			Name:     "Alice Again",
			Email:    "alice@example.com",
			Password: "p2",
		}), nil, errors.New("duplicate key"))
		if !ninja.IsConflict(conflictErr) {
			t.Fatalf("expected generic conflict error, got %T %v", conflictErr, conflictErr)
		}
	})

	t.Run("relation option error branches", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}
		if err := db.AutoMigrate(&adminUser{}, &adminProject{}); err != nil {
			t.Fatalf("AutoMigrate: %v", err)
		}
		if err := db.Create(&adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}).Error; err != nil {
			t.Fatalf("Create(user): %v", err)
		}
		orm.Init(db)

		makeCtx := func() *ninja.Context {
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			return &ninja.Context{Context: ginCtx}
		}

		projectResource := &Resource{
			Name:         "projects",
			Model:        adminProject{},
			ListFields:   []string{"id", "owner_id"},
			DetailFields: []string{"id", "owner_id"},
		}
		if err := projectResource.prepare(); err != nil {
			t.Fatalf("prepare(projects): %v", err)
		}
		projectResource.fieldByName["owner_id"].Meta.Relation = &RelationMeta{
			Resource:   "users",
			ValueField: "id",
			LabelField: "name",
		}

		site := NewSite()
		ctx := makeCtx()

		if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "title"}); !ninja.IsNotFound(err) {
			t.Fatalf("expected not found for non-relation field, got %v", err)
		}

		if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id"}); err == nil {
			t.Fatal("expected missing relation resource error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `relation resource "users" is not registered`) {
			t.Fatalf("expected missing relation resource BAD_REQUEST, got %T %v", err, err)
		}

		userResource := &Resource{
			Name:         "users",
			Model:        adminUser{},
			ListFields:   []string{"id", "name"},
			DetailFields: []string{"id", "name"},
		}
		if err := userResource.prepare(); err != nil {
			t.Fatalf("prepare(users): %v", err)
		}
		site.byName["users"] = userResource
		if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id"}); err != nil {
			t.Fatalf("expected relation options success, got %v", err)
		}

		projectResource.fieldByName["owner_id"].Meta.Relation.LabelField = "missing"
		if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id"}); err == nil {
			t.Fatal("expected missing relation field error")
		} else if apiErr, ok := err.(*ninja.Error); !ok || apiErr.Code != "BAD_REQUEST" || !strings.Contains(apiErr.Message, `relation fields "id"/"missing" are not available`) {
			t.Fatalf("expected missing relation field BAD_REQUEST, got %T %v", err, err)
		}

		projectResource.fieldByName["owner_id"].Meta.Relation.LabelField = "name"
		userResource.Permissions = func(ctx *ninja.Context, action Action, resource *Resource) error {
			if action == ActionList {
				return ninja.ForbiddenError()
			}
			return nil
		}
		if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id"}); !ninja.IsForbidden(err) {
			t.Fatalf("expected forbidden relation options error, got %v", err)
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

func TestAdminMetadataEdgeHelpers(t *testing.T) {
	t.Run("field options cloning and metadata helpers", func(t *testing.T) {
		hidden := true
		create := true
		meta := &fieldMeta{
			Meta: FieldMeta{
				Name:       "role_ids",
				Column:     "role_ids",
				List:       true,
				Detail:     true,
				Create:     false,
				Update:     true,
				Filterable: true,
				Sortable:   true,
				Searchable: true,
			},
		}
		applyFieldOptions(meta, FieldOptions{
			Label:  "Roles",
			Enum:   []any{"admin", "editor"},
			Hidden: &hidden,
			Create: &create,
			Relation: &RelationOptions{
				Resource:     "roles",
				SearchFields: []string{"name"},
			},
		})
		if meta.Meta.Label != "Roles" || meta.Meta.Component != "select" {
			t.Fatalf("unexpected field options result: %+v", meta.Meta)
		}
		if meta.Meta.List || meta.Meta.Detail || meta.Meta.Update || meta.Meta.Filterable || meta.Meta.Sortable || meta.Meta.Searchable {
			t.Fatalf("expected hidden option to disable metadata flags: %+v", meta.Meta)
		}
		if meta.Meta.Create {
			t.Fatalf("expected hidden option to override create visibility: %+v", meta.Meta)
		}
		normalizeResolvedField(&meta.Meta)
		if includeFieldInMetadata(meta) {
			t.Fatal("expected fully hidden field to be omitted from metadata")
		}
		if isPrimaryKeyField(nil) {
			t.Fatal("expected nil field to not be primary key")
		}
		if !isPrimaryKeyField(&fieldMeta{Meta: FieldMeta{Name: "id", Column: "id", ReadOnly: true}}) {
			t.Fatal("expected readonly id field to count as primary key")
		}

		cloned := cloneFieldOptionsMap(map[string]FieldOptions{
			"role_ids": {
				Enum: []any{"admin"},
				Relation: &RelationOptions{
					Resource:     "roles",
					SearchFields: []string{"name"},
				},
			},
		})
		cloned["role_ids"].Enum[0] = "mutated"
		clonedRelation := cloned["role_ids"].Relation
		clonedRelation.SearchFields[0] = "email"
		original := cloneFieldOptionsMap(map[string]FieldOptions{
			"role_ids": {
				Enum: []any{"admin"},
				Relation: &RelationOptions{
					Resource:     "roles",
					SearchFields: []string{"name"},
				},
			},
		})
		if original["role_ids"].Enum[0] != "admin" || original["role_ids"].Relation.SearchFields[0] != "name" {
			t.Fatalf("expected cloned options to deep copy nested slices: %+v", original["role_ids"])
		}

		resource := &Resource{
			fieldByName: map[string]*fieldMeta{
				"name":  {Meta: FieldMeta{Name: "name"}, fieldType: reflect.TypeOf("")},
				"email": {Meta: FieldMeta{Name: "email"}, fieldType: reflect.TypeOf("")},
			},
			fields: []*fieldMeta{
				{Meta: FieldMeta{Name: "name"}, fieldType: reflect.TypeOf("")},
				{Meta: FieldMeta{Name: "email"}, fieldType: reflect.TypeOf("")},
			},
			primaryKey: &fieldMeta{Meta: FieldMeta{Name: "uuid"}},
		}
		if got := inferRelationLabelField(nil); got != "id" {
			t.Fatalf("inferRelationLabelField(nil) = %q", got)
		}
		if got := inferRelationLabelField(resource); got != "name" {
			t.Fatalf("inferRelationLabelField(resource) = %q", got)
		}
		searchFields := inferRelationSearchFields(resource, "email")
		if !reflect.DeepEqual(searchFields, []string{"email", "name"}) {
			t.Fatalf("unexpected inferred relation search fields: %v", searchFields)
		}
		if !isWritableField(reflect.TypeOf("")) {
			t.Fatal("expected string field to be writable")
		}
		for _, typ := range []reflect.Type{
			reflect.TypeOf(time.Time{}),
			reflect.TypeOf(struct{}{}),
			reflect.TypeOf(map[string]string{}),
			reflect.TypeOf((*interface{})(nil)).Elem(),
			reflect.TypeOf(func() {}),
		} {
			if isWritableField(typ) {
				t.Fatalf("expected %v to be non-writable", typ)
			}
		}
	})

	t.Run("delete helpers and duplicate detection", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}
		if err := db.AutoMigrate(&adminUser{}); err != nil {
			t.Fatalf("AutoMigrate: %v", err)
		}
		orm.Init(db)

		resource := &Resource{Name: "users", Model: adminUser{}}
		if err := resource.prepare(); err != nil {
			t.Fatalf("prepare(): %v", err)
		}

		ctxRecorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(ctxRecorder)
		ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
		ctx := &ninja.Context{Context: ginCtx}

		if deleted, err := resource.deleteModelWithHooks(ctx, db, nil); err != nil || deleted {
			t.Fatalf("deleteModelWithHooks(nil) = (%v, %v)", deleted, err)
		}

		user := adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("Create(user): %v", err)
		}
		model, err := resource.findByID(db, strconv.Itoa(int(user.ID)))
		if err != nil {
			t.Fatalf("findByID(): %v", err)
		}
		if deleted, err := resource.deleteModelWithHooks(ctx, db, model); err != nil || !deleted {
			t.Fatalf("deleteModelWithHooks() = (%v, %v)", deleted, err)
		}
		if deleted, err := resource.deleteModelWithHooks(ctx, db, model); err != nil || deleted {
			t.Fatalf("expected repeated delete to report no-op, got (%v, %v)", deleted, err)
		}

		another := adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2"}
		if err := db.Create(&another).Error; err != nil {
			t.Fatalf("Create(another): %v", err)
		}
		resource.AfterDelete = func(ctx *ninja.Context, model any) error { return errors.New("after delete failed") }
		model, err = resource.findByID(db, strconv.Itoa(int(another.ID)))
		if err != nil {
			t.Fatalf("findByID(another): %v", err)
		}
		if deleted, err := resource.deleteModelWithHooks(ctx, db, model); err == nil || deleted || !strings.Contains(err.Error(), "after delete failed") {
			t.Fatalf("expected after delete error, got deleted=%v err=%v", deleted, err)
		}

		if !isDuplicateKeyError(gorm.ErrDuplicatedKey) {
			t.Fatal("expected gorm duplicated key to be detected")
		}
		if !isDuplicateKeyError(errors.New("UNIQUE constraint failed: users.email")) {
			t.Fatal("expected sqlite unique violation to be detected")
		}
		if isDuplicateKeyError(errors.New("validation failed")) {
			t.Fatal("expected non-duplicate error to be ignored")
		}
	})
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
