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
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type csvStringer string

func (s csvStringer) String() string { return "stringer:" + string(s) }

type noPrimaryFields struct {
	Name string
}

type unexportedOnly struct {
}

type metadataShape struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `binding:"required,email"`
	Password    string
	Description string         `gorm:"type:text"`
	Tags        []string       `gorm:"-"`
	Blob        map[string]any `gorm:"-"`
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt
	Hidden      string `json:"-"`
	RelationID  uint
	Relation    adminUser
}

type ownerRelationShape struct {
	ID      uint `gorm:"primaryKey"`
	OwnerID uint
	Owner   adminUser
}

func adminUnitContextWithDB(t *testing.T, db *gorm.DB, method, target string, body io.Reader) *ninja.Context {
	t.Helper()

	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(method, target, body)
	if db != nil {
		orm.Middleware(db)(ginCtx)
	}
	return &ninja.Context{Context: ginCtx}
}

func openAdminMemoryDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
	}
	return db
}

func TestAdminBranchCoverageForRegistrationAndRelations(t *testing.T) {
	if err := NewSite().Register(&Resource{Name: "bad", Model: 12}); err == nil {
		t.Fatal("expected Register to surface prepare errors")
	}
	if (&ModelResource{}).Resource() == nil {
		t.Fatal("expected empty ModelResource to still produce a Resource")
	}
	var nilModelResource *ModelResource
	if nilModelResource.Resource() != nil {
		t.Fatal("expected nil ModelResource to return nil")
	}

	hidden := false
	modelResource := &ModelResource{
		Name:         "projects",
		Model:        adminProject{},
		Preloads:     []string{"Owner"},
		ListFields:   []string{"id", "title"},
		FieldOptions: map[string]FieldOptions{"title": {Hidden: &hidden, Enum: []any{"a"}, Relation: &RelationOptions{Resource: "users", SearchFields: []string{"name"}}}},
	}
	resource := modelResource.Resource()
	modelResource.ListFields[0] = "mutated"
	modelResource.FieldOptions["title"].Enum[0] = "mutated"
	modelResource.FieldOptions["title"].Relation.SearchFields[0] = "mutated"
	if resource.ListFields[0] != "id" || resource.FieldOptions["title"].Enum[0] != "a" || resource.FieldOptions["title"].Relation.SearchFields[0] != "name" {
		t.Fatalf("expected ModelResource.Resource to deep clone mutable fields: %+v", resource)
	}

	site := NewSite()
	first := &Resource{Name: "users", Model: autoResourceUser{}}
	second := &Resource{Name: "members", Model: autoResourceUser{}}
	third := &Resource{Name: "people", Model: autoResourceUser{}}
	for _, current := range []*Resource{first, second, third} {
		if err := current.prepare(); err != nil {
			t.Fatalf("prepare %q: %v", current.Name, err)
		}
		site.registerModel(current)
	}
	if _, ok := site.byModel[first.modelType]; ok {
		t.Fatalf("expected duplicate model registration to be marked ambiguous, got %+v", site.byModel)
	}
	if _, ok := site.ambiguousModels[first.modelType]; !ok {
		t.Fatalf("expected ambiguous model marker, got %+v", site.ambiguousModels)
	}
	site.registerModel(nil)
	(*Site)(nil).registerModel(first)

	orphan := &Resource{Name: "projects", Model: ownerRelationShape{}}
	if err := orphan.prepare(); err != nil {
		t.Fatalf("prepare orphan: %v", err)
	}
	NewSite().resolveAutoRelations()
	(*Site)(nil).resolveAutoRelations()
	(&Site{resources: []*Resource{orphan}, byModel: map[reflect.Type]*Resource{}, byName: map[string]*Resource{}}).resolveAutoRelations()
	if got := orphan.fieldByName["ownerID"].Meta.Relation; got == nil || got.Resource != "" || got.ValueField != "id" {
		t.Fatalf("expected orphan auto relation to be reset without a resource name, got %+v", got)
	}
	explicit := &Resource{Name: "explicit", Model: ownerRelationShape{}}
	if err := explicit.prepare(); err != nil {
		t.Fatalf("prepare explicit: %v", err)
	}
	explicit.fieldByName["ownerID"].autoRelation = nil
	explicit.fieldByName["ownerID"].Meta.Relation = &RelationMeta{Resource: "missing"}
	(&Site{resources: []*Resource{nil, explicit}, byModel: map[reflect.Type]*Resource{}, byName: map[string]*Resource{}}).resolveAutoRelations()
	if got := explicit.fieldByName["ownerID"].Meta.Relation; got == nil || got.Resource != "missing" || got.ValueField != "id" {
		t.Fatalf("expected explicit missing relation to keep resource and default value field, got %+v", got)
	}
	clone := cloneRelationMeta(nil)
	if clone != nil {
		t.Fatalf("expected nil relation clone, got %+v", clone)
	}
	resetAutoRelation(nil)
	resetAutoRelation(&fieldMeta{})

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected Mount(nil) to panic")
			}
		}()
		site.Mount(nil)
	}()
}

func TestAdminBranchCoverageForMetadataHelpers(t *testing.T) {
	if err := (&Resource{Model: nil}).prepare(); err == nil {
		t.Fatal("expected nil model to fail prepare")
	}
	if err := (&Resource{Name: "bad", Model: 12}).prepare(); err == nil {
		t.Fatal("expected non-struct model to fail prepare")
	}
	if err := (&Resource{Name: "hidden", Model: unexportedOnly{}}).prepare(); err == nil {
		t.Fatal("expected model without exported fields to fail prepare")
	}
	if err := (&Resource{Name: "nopk", Model: noPrimaryFields{}}).prepare(); err == nil {
		t.Fatal("expected model without primary key to fail prepare")
	}
	blank := reflect.StructOf([]reflect.StructField{{
		Name: "ID",
		Type: reflect.TypeOf(uint(0)),
		Tag:  `gorm:"primaryKey"`,
	}})
	if got := inferResourceName(blank); got != "" {
		t.Fatalf("anonymous reflected struct inferred name = %q, want empty", got)
	}

	resource := &Resource{
		Name:  "shapes",
		Model: metadataShape{},
		FieldOptions: map[string]FieldOptions{
			"relationID": {
				Relation: &RelationOptions{Resource: "users"},
			},
		},
	}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare metadataShape: %v", err)
	}
	if got := resource.fieldByName["email"].Meta.Component; got != "email" {
		t.Fatalf("email component = %q", got)
	}
	if got := resource.fieldByName["password"].Meta.Component; got != "password" {
		t.Fatalf("password component = %q", got)
	}
	if got := resource.fieldByName["description"].Meta.Component; got != "textarea" {
		t.Fatalf("description component = %q", got)
	}
	if got := resource.fieldByName["tags"].Meta.Type; got != "array" {
		t.Fatalf("tags type = %q", got)
	}
	if got := resource.fieldByName["blob"].Meta.Type; got != "object" {
		t.Fatalf("blob type = %q", got)
	}
	if !resource.fieldByName["hidden"].Meta.ReadOnly || resource.fieldByName["hidden"].Meta.List {
		t.Fatalf("expected json-hidden field to be read-only and hidden from list: %+v", resource.fieldByName["hidden"].Meta)
	}
	if got := normalizePath(" /admin/ "); got != "/admin" {
		t.Fatalf("normalizePath() = %q", got)
	}
	if got := normalizePath(" "); got != "/" {
		t.Fatalf("normalizePath(blank) = %q", got)
	}
	if got := toSnake("HTTPRequestID"); got != "http_request_id" {
		t.Fatalf("toSnake acronym = %q", got)
	}
	if got := defaultJSONFieldName("URLValue"); got != "urlValue" {
		t.Fatalf("defaultJSONFieldName acronym = %q", got)
	}
	if got := humanize("api_userID"); got != "Api User Id" {
		t.Fatalf("humanize = %q", got)
	}

	tagged := &fieldMeta{Meta: FieldMeta{Name: "owner_id", Column: "owner_id"}}
	applyAdminTag(tagged, parseTagSettings("relation:users;relation_value:id;relation_label:name;relation_search:name,email;readonly:false;list:false;sort:true"))
	if tagged.Meta.Relation == nil || tagged.Meta.Relation.Resource != "users" || !reflect.DeepEqual(tagged.Meta.Relation.SearchFields, []string{"name", "email"}) {
		t.Fatalf("unexpected relation tag metadata: %+v", tagged.Meta.Relation)
	}
	if tagged.Meta.ReadOnly || tagged.Meta.List || !tagged.Meta.Sortable {
		t.Fatalf("unexpected bool tag metadata: %+v", tagged.Meta)
	}
}

func TestAdminBranchCoverageForSearchStatsAndLabels(t *testing.T) {
	db := openAdminMemoryDB(t, &adminUser{})
	if err := db.Create(&adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	resource := &Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email", "age"},
		DetailFields: []string{"id", "name", "email"},
		SearchFields: []string{"name", "email"},
	}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare resource: %v", err)
	}
	site := NewSite()
	site.resources = []*Resource{resource}
	ctx := adminUnitContextWithDB(t, db, http.MethodGet, "/", nil)

	shortSearch, err := site.searchResources(ctx, &searchInput{Q: "a", Size: 99})
	if err != nil {
		t.Fatalf("short search: %v", err)
	}
	if shortSearch.Size != 20 || shortSearch.Total != 0 || len(shortSearch.Results) != 0 {
		t.Fatalf("unexpected short search output: %+v", shortSearch)
	}
	search, err := site.searchResources(ctx, &searchInput{Q: " alice ", Size: 0})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if search.Size != 5 || search.Total != 1 || len(search.Results) != 1 || search.Results[0].Items[0].Label != "Alice" {
		t.Fatalf("unexpected search output: %+v", search)
	}
	stats, err := site.resourceStats(ctx, nil)
	if err != nil {
		t.Fatalf("resourceStats: %v", err)
	}
	if stats.Total != 1 || len(stats.Resources) != 1 || stats.Resources[0].Total != 1 {
		t.Fatalf("unexpected stats output: %+v", stats)
	}

	genericErr := errors.New("permission backend down")
	site.checker = func(ctx *ninja.Context, action Action, resource *Resource) error { return genericErr }
	if _, err := site.listResources(ctx, nil); !errors.Is(err, genericErr) {
		t.Fatalf("expected generic list authorization error, got %v", err)
	}
	if _, err := site.resourceStats(ctx, nil); !errors.Is(err, genericErr) {
		t.Fatalf("expected generic stats authorization error, got %v", err)
	}
	if _, err := site.searchResources(ctx, &searchInput{Q: "alice"}); !errors.Is(err, genericErr) {
		t.Fatalf("expected generic search authorization error, got %v", err)
	}

	if got := (*Resource)(nil).summary(); got != (ResourceSummary{}) {
		t.Fatalf("nil resource summary = %+v", got)
	}
	if got := (*Resource)(nil).searchLabelFor(resource.resolved(nil), reflect.ValueOf(adminUser{})); got != "" {
		t.Fatalf("nil resource searchLabelFor = %q", got)
	}
	noString := &Resource{Name: "metrics", Model: adminMetrics{}, ListFields: []string{"id", "count"}, SearchFields: []string{"count"}}
	if err := noString.prepare(); err != nil {
		t.Fatalf("prepare metrics: %v", err)
	}
	if got := noString.searchLabelFor(noString.resolved(nil), reflect.ValueOf(adminMetrics{ID: 44, Count: 7})); got != "44" {
		t.Fatalf("expected primary-key fallback label, got %q", got)
	}
	noString.primaryKey = nil
	if got := noString.searchLabelFor(noString.resolved(nil), reflect.ValueOf(adminMetrics{ID: 44, Count: 7})); got != "" {
		t.Fatalf("expected empty label without primary key, got %q", got)
	}
}

func TestAdminBranchCoverageForCSVExportAndWriteEdges(t *testing.T) {
	now := time.Date(2026, 4, 17, 4, 57, 29, 0, time.UTC)
	cases := map[string]string{
		"nil":       csvCellValue(nil),
		"zeroTime":  csvCellValue(time.Time{}),
		"time":      csvCellValue(now),
		"stringer":  csvCellValue(csvStringer("value")),
		"bytes":     csvCellValue([]byte("raw")),
		"slice":     csvCellValue([]int{1, 2}),
		"map":       csvCellValue(map[string]int{"a": 1}),
		"struct":    csvCellValue(struct{ Name string }{"Alice"}),
		"badStruct": csvCellValue(struct{ Fn func() }{}),
		"number":    csvCellValue(12),
	}
	if cases["nil"] != "" || cases["zeroTime"] != "" || cases["time"] != now.Format(time.RFC3339) || cases["stringer"] != "stringer:value" || cases["bytes"] != "raw" {
		t.Fatalf("unexpected scalar csv values: %+v", cases)
	}
	if cases["slice"] != `[1,2]` || cases["map"] != `{"a":1}` || cases["struct"] != `{"Name":"Alice"}` || !strings.Contains(cases["badStruct"], "<nil>") || cases["number"] != "12" {
		t.Fatalf("unexpected composite csv values: %+v", cases)
	}
	if got := exportFilename(" Users API!! ", now); got != "users-api-20260417-045729.csv" {
		t.Fatalf("exportFilename = %q", got)
	}
	if got := exportFilename("!!!", now); got != "resource-20260417-045729.csv" {
		t.Fatalf("exportFilename fallback = %q", got)
	}
	if got := exportFilename("user_list", now); got != "user_list-20260417-045729.csv" {
		t.Fatalf("exportFilename underscore = %q", got)
	}

	resource := &Resource{Name: "users", Model: adminUser{}}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare resource: %v", err)
	}
	if _, err := resource.findByID(openAdminMemoryDB(t, &adminUser{}), "not-a-number"); err == nil {
		t.Fatal("expected invalid id error")
	}
	if _, err := resource.parsePrimaryKeyJSON(json.RawMessage(`"bad"`)); err == nil {
		t.Fatal("expected invalid primary key JSON error")
	}
	if err := resource.reloadScopedWrite(nil, &adminUser{}); !ninja.IsInternal(err) {
		t.Fatalf("expected internal error for nil db, got %v", err)
	}
	if err := resource.reloadScopedWrite(openAdminMemoryDB(t, &adminUser{}), adminUser{}); !ninja.IsInternal(err) {
		t.Fatalf("expected internal error for non-pointer model, got %v", err)
	}

	db := openAdminMemoryDB(t, &adminUser{})
	ctx := adminUnitContextWithDB(t, db, http.MethodPost, "/", bytes.NewReader(nil))
	resource.BeforeCreate = func(ctx *ninja.Context, values map[string]any) error { return errors.New("before create failed") }
	if _, err := resource.handleCreate(NewSite())(ctx, nil); err == nil || !strings.Contains(err.Error(), "before create failed") {
		t.Fatalf("expected before create hook error, got %v", err)
	}
	resource.BeforeCreate = nil
	resource.AfterCreate = func(ctx *ninja.Context, model any) error { return errors.New("after create failed") }
	ctx = adminUnitContextWithDB(t, db, http.MethodPost, "/", strings.NewReader(`{"name":"Alice","email":"alice@example.com"}`))
	if _, err := resource.handleCreate(NewSite())(ctx, nil); err == nil || !strings.Contains(err.Error(), "after create failed") {
		t.Fatalf("expected after create hook error, got %v", err)
	}

	resource.AfterCreate = nil
	if err := db.Create(&adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	resource.BeforeUpdate = func(ctx *ninja.Context, model any, values map[string]any) error {
		return errors.New("before update failed")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodPut, "/", strings.NewReader(`{"name":"Bobby"}`))
	if _, err := resource.handleUpdate(NewSite())(ctx, &pathIDInput{ID: "1"}); err == nil || !strings.Contains(err.Error(), "before update failed") {
		t.Fatalf("expected before update hook error, got %v", err)
	}
	resource.BeforeUpdate = nil
	resource.AfterUpdate = func(ctx *ninja.Context, model any) error { return errors.New("after update failed") }
	ctx = adminUnitContextWithDB(t, db, http.MethodPut, "/", strings.NewReader(`{"name":"Bobby"}`))
	if _, err := resource.handleUpdate(NewSite())(ctx, &pathIDInput{ID: "1"}); err == nil || !strings.Contains(err.Error(), "after update failed") {
		t.Fatalf("expected after update hook error, got %v", err)
	}
}

func TestAdminBranchCoverageForHandlerErrorEdges(t *testing.T) {
	db := openAdminMemoryDB(t, &adminUser{})
	user := adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	resource := &Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name"},
		UpdateFields: []string{"name"},
	}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare resource: %v", err)
	}
	site := NewSite()
	ctx := adminUnitContextWithDB(t, db, http.MethodGet, "/?search=alice", nil)
	resource.resolvedView.metadata.SearchFields = nil
	if _, err := resource.handleList(site)(ctx, &listInput{Search: "alice"}); err == nil {
		t.Fatal("expected search-disabled list error")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodGet, "/?id__gt=bad", nil)
	resource.resolvedView.metadata.FilterFields = []string{"id"}
	if _, err := resource.handleList(site)(ctx, &listInput{}); err == nil {
		t.Fatal("expected bad filter value error")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodGet, "/", nil)
	resource.resolvedView.metadata.SortFields = []string{"id"}
	if _, err := resource.handleList(site)(ctx, &listInput{Sort: "name"}); err == nil {
		t.Fatal("expected unsupported sort field error")
	}
	if _, err := resource.handleDetail(site)(ctx, &pathIDInput{ID: "bad"}); err == nil {
		t.Fatal("expected invalid detail id error")
	}
	if _, err := resource.handleDetail(site)(ctx, &pathIDInput{ID: "999"}); !ninja.IsNotFound(err) {
		t.Fatalf("expected detail not found, got %v", err)
	}

	resource.ListFields = nil
	resource.metadata.ListFields = nil
	resource.resolvedView.metadata.ListFields = nil
	if _, err := resource.handleExport(site)(ctx, &listInput{}); err == nil {
		t.Fatal("expected export without fields to fail")
	}

	ctx = adminUnitContextWithDB(t, db, http.MethodPost, "/", strings.NewReader(`{`))
	if _, err := resource.handleBulkDelete(site)(ctx, nil); err == nil {
		t.Fatal("expected invalid bulk delete JSON")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodPost, "/", strings.NewReader(`{"ids":[]}`))
	if _, err := resource.handleBulkDelete(site)(ctx, nil); err == nil {
		t.Fatal("expected empty bulk delete ids error")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodPost, "/", strings.NewReader(`{"ids":["bad"]}`))
	if _, err := resource.handleBulkDelete(site)(ctx, nil); err == nil {
		t.Fatal("expected invalid bulk delete id error")
	}
}

func TestAdminBranchCoverageForMoreHandlerAndDBErrors(t *testing.T) {
	db := openAdminMemoryDB(t, &adminUser{})
	if err := db.Create(&adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2"}).Error; err != nil {
		t.Fatalf("seed user 2: %v", err)
	}
	resource := &Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
		CreateFields: []string{"name", "email"},
		UpdateFields: []string{"name", "email"},
		SearchFields: []string{"name", "email"},
	}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare resource: %v", err)
	}
	ctx := adminUnitContextWithDB(t, db, http.MethodGet, "/", nil)

	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if action == ActionList {
			return ninja.ForbiddenError()
		}
		return nil
	}))
	site.resources = []*Resource{resource}
	if stats, err := site.resourceStats(ctx, nil); err != nil || stats.Total != 0 || len(stats.Resources) != 0 {
		t.Fatalf("expected forbidden resource stats to be skipped, got %+v err=%v", stats, err)
	}
	if search, err := site.searchResources(ctx, &searchInput{Q: "alice"}); err != nil || search.Total != 0 || len(search.Results) != 0 {
		t.Fatalf("expected forbidden resource search to be skipped, got %+v err=%v", search, err)
	}

	site = NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		return ninja.UnauthorizedError()
	}))
	site.resources = []*Resource{resource}
	if _, err := site.resourceStats(ctx, nil); !ninja.IsUnauthorized(err) {
		t.Fatalf("expected unauthorized stats error, got %v", err)
	}
	if _, err := site.searchResources(ctx, &searchInput{Q: "alice"}); !ninja.IsUnauthorized(err) {
		t.Fatalf("expected unauthorized search error, got %v", err)
	}

	metaErr := errors.New("metadata action failed")
	site = NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if action == ActionCreate {
			return metaErr
		}
		return nil
	}))
	if _, err := resource.handleMetadata(site)(ctx, nil); !errors.Is(err, metaErr) {
		t.Fatalf("expected metadata action error, got %v", err)
	}
	authErr := errors.New("auth down")
	site = NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		return authErr
	}))
	if _, err := resource.handleDetail(site)(ctx, &pathIDInput{ID: "1"}); !errors.Is(err, authErr) {
		t.Fatalf("expected detail authorization error, got %v", err)
	}
	if _, err := resource.handleCreate(site)(ctx, nil); !errors.Is(err, authErr) {
		t.Fatalf("expected create authorization error, got %v", err)
	}
	if _, err := resource.handleUpdate(site)(ctx, &pathIDInput{ID: "1"}); !errors.Is(err, authErr) {
		t.Fatalf("expected update authorization error, got %v", err)
	}
	if err := resource.handleDelete(site)(ctx, &pathIDInput{ID: "1"}); !errors.Is(err, authErr) {
		t.Fatalf("expected delete authorization error, got %v", err)
	}
	if _, err := resource.handleBulkDelete(site)(ctx, nil); !errors.Is(err, authErr) {
		t.Fatalf("expected bulk delete authorization error, got %v", err)
	}

	site = NewSite()
	ctx = adminUnitContextWithDB(t, db, http.MethodGet, "/?fields=missing", nil)
	if _, err := resource.handleExport(site)(ctx, &listInput{}); err == nil {
		t.Fatal("expected export field validation error")
	}

	ctx = adminUnitContextWithDB(t, db, http.MethodPut, "/", strings.NewReader(`{"unknown":true}`))
	if _, err := resource.handleUpdate(site)(ctx, &pathIDInput{ID: "1"}); err == nil {
		t.Fatal("expected update decode error")
	}
	ctx = adminUnitContextWithDB(t, db, http.MethodPut, "/", strings.NewReader(`{"email":"bob@example.com"}`))
	if _, err := resource.handleUpdate(site)(ctx, &pathIDInput{ID: "1"}); !ninja.IsConflict(err) {
		t.Fatalf("expected duplicate update conflict, got %v", err)
	}

	resource.BeforeDelete = func(ctx *ninja.Context, model any) error { return errors.New("before delete failed") }
	if deleted, err := resource.deleteModelWithHooks(ctx, db, &adminUser{ID: 1}); err == nil || deleted || !strings.Contains(err.Error(), "before delete failed") {
		t.Fatalf("expected before delete error, got deleted=%v err=%v", deleted, err)
	}
	resource.BeforeDelete = nil

	readErrCtx := adminUnitContextWithDB(t, db, http.MethodPost, "/", nil)
	readErrCtx.Request.Body = errReadCloser{}
	if _, err := resource.handleBulkDelete(site)(readErrCtx, nil); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("expected bulk delete read error, got %v", err)
	}

	closed := openAdminMemoryDB(t, &adminUser{})
	sqlDB, err := closed.DB()
	if err != nil {
		t.Fatalf("closed.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	closedCtx := adminUnitContextWithDB(t, closed, http.MethodGet, "/", nil)
	if _, err := (&Site{resources: []*Resource{resource}}).resourceStats(closedCtx, nil); err == nil {
		t.Fatal("expected stats count error on closed db")
	}
	if _, err := resource.handleList(NewSite())(closedCtx, &listInput{}); err == nil {
		t.Fatal("expected list count error on closed db")
	}
	if _, err := resource.handleExport(NewSite())(closedCtx, &listInput{}); err == nil {
		t.Fatal("expected export query error on closed db")
	}
	if _, err := resource.findByID(closed, "1"); err == nil {
		t.Fatal("expected findByID db error")
	}
	if err := resource.reloadScopedWrite(closed, &adminUser{ID: 1}); err == nil {
		t.Fatal("expected reload db error")
	}
	if deleted, err := resource.deleteModelWithHooks(closedCtx, closed, &adminUser{ID: 1}); err == nil || deleted {
		t.Fatalf("expected delete db error, got deleted=%v err=%v", deleted, err)
	}
	closedCtx = adminUnitContextWithDB(t, closed, http.MethodPost, "/", strings.NewReader(`{"ids":[1]}`))
	if _, err := resource.handleBulkDelete(NewSite())(closedCtx, nil); err == nil {
		t.Fatal("expected bulk delete find error on closed db")
	}
}

func TestAdminBranchCoverageForMoreMetadataAndSoftDeleteEdges(t *testing.T) {
	for name, configure := range map[string]func(*Resource){
		"list":   func(r *Resource) { r.ListFields = []string{"missing"} },
		"detail": func(r *Resource) { r.DetailFields = []string{"missing"} },
		"create": func(r *Resource) { r.CreateFields = []string{"missing"} },
		"update": func(r *Resource) { r.UpdateFields = []string{"missing"} },
		"filter": func(r *Resource) { r.FilterFields = []string{"missing"} },
		"sort":   func(r *Resource) { r.SortFields = []string{"missing"} },
		"search": func(r *Resource) { r.SearchFields = []string{"missing"} },
	} {
		t.Run("prepare_"+name, func(t *testing.T) {
			resource := &Resource{Name: "users", Model: autoResourceUser{}}
			configure(resource)
			if err := resource.prepare(); err == nil || !strings.Contains(err.Error(), "unknown admin field") {
				t.Fatalf("expected unknown field error, got %v", err)
			}
		})
	}
	(*Resource)(nil).syncMetadataFields()
	if err := (&Resource{Name: "nil-field", fields: []*fieldMeta{nil, {Meta: FieldMeta{Name: "id", Column: "id"}}}}).validateFieldColumns(); err != nil {
		t.Fatalf("nil field should be skipped during column validation: %v", err)
	}

	type embeddedFields struct {
		ID   uint
		Name string
	}
	type withEmbedded struct {
		*embeddedFields
		When time.Time
	}
	if fields := collectFields(reflect.TypeOf(withEmbedded{}), nil, nil); len(fields) < 3 {
		t.Fatalf("expected embedded fields to be collected, got %+v", fields)
	}
	if meta := buildFieldMeta(reflect.StructField{Name: "", Type: reflect.TypeOf("")}, nil); meta != nil {
		t.Fatalf("expected blank field name metadata to be nil, got %+v", meta)
	}

	if _, err := (&fieldMeta{fieldType: reflect.TypeOf(int8(0))}).parseString("200"); err == nil || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("expected int overflow, got %v", err)
	}
	if _, err := (&fieldMeta{fieldType: reflect.TypeOf(uint8(0))}).parseString("300"); err == nil || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("expected uint overflow, got %v", err)
	}
	if _, err := (&fieldMeta{fieldType: reflect.TypeOf(float32(0))}).parseString("1e100"); err == nil || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("expected float overflow, got %v", err)
	}
	if _, err := (&fieldMeta{fieldType: reflect.TypeOf(uint(0))}).parseString("-1"); err == nil {
		t.Fatal("expected uint parse error")
	}

	noStrings := &Resource{
		fieldByName: map[string]*fieldMeta{"id": {Meta: FieldMeta{Name: "id"}, fieldType: reflect.TypeOf(uint(0))}},
		fields:      []*fieldMeta{{Meta: FieldMeta{Name: "id"}, fieldType: reflect.TypeOf(uint(0))}},
		primaryKey:  &fieldMeta{Meta: FieldMeta{Name: "id"}},
	}
	if got := inferRelationLabelField(noStrings); got != "id" {
		t.Fatalf("expected primary-key relation label fallback, got %q", got)
	}
	noStrings.primaryKey = nil
	if got := inferRelationLabelField(noStrings); got != "id" {
		t.Fatalf("expected default relation label fallback, got %q", got)
	}
	if got := inferRelationSearchFields(nil, "name"); got != nil {
		t.Fatalf("expected nil relation search fields, got %+v", got)
	}
	if relationMetaFromTag(map[string]string{"relation": "true"}) != nil {
		t.Fatal("expected relation:true to not produce relation metadata")
	}
	applyAdminTag(nil, map[string]string{"hidden": "true"})
	applyAdminTag(&fieldMeta{}, nil)

	resource := &Resource{Name: "users", Model: adminUser{}}
	if err := resource.prepare(); err != nil {
		t.Fatalf("prepare user resource: %v", err)
	}
	if got := (&Resource{fields: []*fieldMeta{nil}}).softDeleteField(); got != nil {
		t.Fatalf("expected nil soft delete field, got %+v", got)
	}
	if fields := resource.softDeletedConflictFields(nil, ActionCreate, reflect.ValueOf(adminUser{}), nil); fields != nil {
		t.Fatalf("expected nil conflict fields without context, got %+v", fields)
	}
	emptyCtx := &ninja.Context{}
	if fields := resource.softDeletedConflictFields(emptyCtx, ActionCreate, reflect.ValueOf(adminUser{}), nil); fields != nil {
		t.Fatalf("expected nil conflict fields without gin context, got %+v", fields)
	}
	db := openAdminMemoryDB(t, &adminUser{})
	ctx := adminUnitContextWithDB(t, db, http.MethodPost, "/", nil)
	if fields := resource.softDeletedConflictFields(ctx, ActionCreate, reflect.Value{}, nil); fields != nil {
		t.Fatalf("expected nil conflict fields with invalid desired value, got %+v", fields)
	}
	noDBCtx := adminUnitContextWithDB(t, nil, http.MethodPost, "/", nil)
	if fields := resource.softDeletedConflictFields(noDBCtx, ActionCreate, reflect.ValueOf(adminUser{}), nil); fields != nil {
		t.Fatalf("expected nil conflict fields without request db, got %+v", fields)
	}
	if err := db.Create(&adminUser{Name: "Active", Email: "active@example.com", Password: "p1"}).Error; err != nil {
		t.Fatalf("seed active user: %v", err)
	}
	if fields := resource.softDeletedConflictFields(ctx, ActionCreate, reflect.ValueOf(adminUser{Email: "active@example.com"}), nil); fields != nil {
		t.Fatalf("expected active duplicate to suppress soft-delete guidance, got %+v", fields)
	}
	if got := (&Resource{}).primaryKeyValue(reflect.ValueOf(adminUser{})); got != nil {
		t.Fatalf("expected nil primary key value without primary field, got %v", got)
	}
}

func TestAdminBranchCoverageForRuntimeHelpersAndRelationOptions(t *testing.T) {
	field := &fieldMeta{Meta: FieldMeta{Name: "name", Column: "name", List: true}, fieldType: reflect.TypeOf(""), index: []int{0}, persisted: true}
	view := &resolvedResource{fields: []*fieldMeta{field}, fieldByName: map[string]*fieldMeta{"name": field}}
	if got := view.meta(nil); !reflect.DeepEqual(got, FieldMeta{}) {
		t.Fatalf("meta(nil) = %+v", got)
	}
	if view.allowed(field, fieldMode("unknown")) {
		t.Fatal("expected unknown field mode to be disallowed")
	}
	normalizeResolvedField(nil)
	meta := FieldMeta{Relation: &RelationMeta{}}
	normalizeResolvedField(&meta)
	if meta.Component != "select" {
		t.Fatalf("expected relation metadata to default component, got %+v", meta)
	}
	if includeFieldInMetadata(nil) {
		t.Fatal("expected nil field to be omitted from metadata")
	}

	type target struct {
		Name string
		Age  int
	}
	badSetter := &fieldMeta{Meta: FieldMeta{Name: "name", Column: "name"}, fieldType: reflect.TypeOf(""), index: []int{1}}
	if err := (&Resource{}).applyValuesFor(&resolvedResource{fieldByName: map[string]*fieldMeta{"missing": nil, "name": badSetter}}, reflect.ValueOf(&target{}).Elem(), map[string]any{"missing": "ignored", "name": "bad"}); err == nil {
		t.Fatal("expected applyValuesFor to surface assignment error")
	}
	before := reflect.ValueOf(target{Name: "same", Age: 1})
	after := reflect.ValueOf(target{Name: "same", Age: 2})
	columns, err := (&Resource{}).updateColumnsFor(&resolvedResource{
		fields: []*fieldMeta{
			nil,
			{Meta: FieldMeta{Name: "id", Column: "id"}, primaryKey: true, persisted: true, index: []int{1}},
			{Meta: FieldMeta{Name: "blank", Column: " "}, persisted: true, index: []int{1}},
			{Meta: FieldMeta{Name: "name", Column: "name"}, persisted: true, index: []int{0}},
			{Meta: FieldMeta{Name: "age", Column: "age"}, persisted: true, index: []int{1}},
		},
	}, before, after)
	if err != nil || !reflect.DeepEqual(columns, []string{"age"}) {
		t.Fatalf("updateColumnsFor helpers = %v, %v", columns, err)
	}
	if out := (&Resource{}).serializeFor(view, reflect.ValueOf(&target{Name: "Alice"}), fieldModeList); out["name"] != "Alice" {
		t.Fatalf("serializeFor pointer = %+v", out)
	}
	if queryColumnFor(nil, FieldMeta{Name: "name", Column: "name"}) != "" {
		t.Fatal("expected nil queryColumnFor to be empty")
	}

	db := openAdminMemoryDB(t, &adminUser{}, &adminProject{})
	if err := db.Create(&adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userResource := &Resource{Name: "users", Model: adminUser{}, ListFields: []string{"id", "name"}, SearchFields: []string{"name"}}
	projectResource := &Resource{Name: "projects", Model: adminProject{}, ListFields: []string{"id", "owner_id"}}
	if err := userResource.prepare(); err != nil {
		t.Fatalf("prepare users: %v", err)
	}
	if err := projectResource.prepare(); err != nil {
		t.Fatalf("prepare projects: %v", err)
	}
	projectResource.fieldByName["owner_id"].Meta.Relation = &RelationMeta{Resource: "users", ValueField: "id", LabelField: "name", SearchFields: []string{"missing", "name"}}
	site := &Site{byName: map[string]*Resource{"users": userResource}}
	if _, err := projectResource.handleRelationOptions(site)(nil, &relationOptionsInput{Field: "owner_id"}); !ninja.IsInternal(err) {
		t.Fatalf("expected nil ctx relation options error, got %v", err)
	}
	noDBCtx := adminUnitContextWithDB(t, nil, http.MethodGet, "/", nil)
	if _, err := projectResource.handleRelationOptions(site)(noDBCtx, &relationOptionsInput{Field: "owner_id"}); !ninja.IsInternal(err) {
		t.Fatalf("expected missing db relation options error, got %v", err)
	}
	ctx := adminUnitContextWithDB(t, db, http.MethodGet, "/", nil)
	out, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id", Search: "1"})
	if err != nil {
		t.Fatalf("relation options numeric search: %v", err)
	}
	if out.Total != 1 || len(out.Items) != 1 || out.Items[0].Label != "Alice" {
		t.Fatalf("unexpected relation options output: %+v", out)
	}
	userResource.FieldPermissions = func(ctx *ninja.Context, resource *Resource, meta *FieldMeta) {
		if meta.Name == "name" {
			meta.Column = "name desc"
		}
	}
	if _, err := projectResource.handleRelationOptions(site)(ctx, &relationOptionsInput{Field: "owner_id", Search: "Alice"}); err == nil || !strings.Contains(err.Error(), "unsafe column") {
		t.Fatalf("expected unsafe relation search column error, got %v", err)
	}
}

func TestAdminBranchCoverageForUIHelpers(t *testing.T) {
	cfg := normalizeUIConfig(UIConfig{Title: " T ", BrandName: " B ", LogoText: " XY ", Locale: " zh ", DefaultTheme: "dark", TokenStorage: "session", APIBasePath: "/api", AuthLoginPath: "/login", AuthMePath: "/me", AdminPath: "/admin", LoginPath: "/admin/login", TokenExtractExpr: " token ", UserNameExtractExpr: " name ", UserIDExtractExpr: " id "})
	if cfg.DefaultTheme != "dark" || cfg.TokenStorage != "session" || cfg.Title != " T " {
		t.Fatalf("unexpected normalized explicit UI config: %+v", cfg)
	}
	defaultedPaths := normalizeUIConfig(UIConfig{Title: "x", BrandName: "x", LogoText: "x", Locale: "x", DefaultTheme: "dark", TokenStorage: "session"})
	if defaultedPaths.AdminPath != "/admin" || defaultedPaths.LoginPath != "/admin/login" {
		t.Fatalf("expected UI paths to default, got %+v", defaultedPaths)
	}
	if got := adminDashboardPath("/admin?x=1"); got != "/admin?x=1&view=dashboard" {
		t.Fatalf("adminDashboardPath query = %q", got)
	}
	if got := shortLogoText(""); got != "G" {
		t.Fatalf("shortLogoText empty = %q", got)
	}
	if got := shortLogoText("Go"); got != "Go" {
		t.Fatalf("shortLogoText short = %q", got)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected missing embedded asset read to panic")
			}
		}()
		_ = mustReadAdminAsset("missing.asset")
	}()
}

func TestAdminBranchCoverageForSmallHelperEdges(t *testing.T) {
	name := "Alice"
	type pointerSample struct {
		Name *string
	}
	resource := &Resource{}
	nameField := &fieldMeta{Meta: FieldMeta{Name: "name", Column: "name"}, fieldType: reflect.TypeOf(""), index: []int{0}}
	if value, ok := resource.fieldValue(reflect.ValueOf(pointerSample{Name: &name}), nameField); !ok || value != "Alice" {
		t.Fatalf("expected pointer field value, got value=%v ok=%v", value, ok)
	}
	if value, ok := resource.fieldValue(reflect.ValueOf(pointerSample{}), nameField); !ok || value != nil {
		t.Fatalf("expected nil pointer field value, got value=%v ok=%v", value, ok)
	}

	actions := appendAction([]Action{ActionList}, ActionCreate, false)
	if !reflect.DeepEqual(actions, []Action{ActionList}) {
		t.Fatalf("expected disabled action append to leave list unchanged, got %+v", actions)
	}
	if anyWritable([]*fieldMeta{{Meta: FieldMeta{Name: "name", Create: false}}}, fieldModeCreate) {
		t.Fatal("expected no writable fields")
	}
	if isDuplicateKeyError(nil) {
		t.Fatal("expected nil error not to be a duplicate key error")
	}
	if !isDuplicateKeyError(errors.New("violates unique constraint users_email_key")) {
		t.Fatal("expected postgres unique violation text to be detected")
	}

	unsafeField := &fieldMeta{Meta: FieldMeta{Name: "name", Column: "name desc"}, fieldType: reflect.TypeOf("")}
	if _, err := applyFilter(nil, url.Values{"name": {"Alice"}}, unsafeField, unsafeField.Meta); err == nil || !strings.Contains(err.Error(), "unsafe column") {
		t.Fatalf("expected unsafe filter column error, got %v", err)
	}
}
