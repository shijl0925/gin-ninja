package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testClaims struct{ id uint }

func (c testClaims) GetUserID() uint { return c.id }

type adminUser struct {
	ID        uint           `gorm:"primaryKey"`
	Name      string         `gorm:"not null" json:"name"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Password  string         `gorm:"not null" json:"-"`
	Age       int            `json:"age"`
	IsAdmin   bool           `json:"is_admin"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type adminProject struct {
	ID        uint           `gorm:"primaryKey"`
	Title     string         `gorm:"not null" json:"title"`
	OwnerID   uint           `gorm:"column:owner_id;not null;index" json:"owner_id"`
	Owner     adminUser      `gorm:"foreignKey:OwnerID" json:"-"`
	Secret    string         `json:"secret"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type autoResourceUser struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `json:"name"`
}

type adminOwnerByID struct {
	OwnerID uint      `json:"owner_id"`
	Owner   adminUser `json:"-"`
}

type adminOwnerById struct {
	OwnerId uint      `json:"owner_id"`
	Owner   adminUser `json:"-"`
}

type adminOwnerWithoutRelation struct {
	OwnerID uint `json:"owner_id"`
}

type adminOwnerWithScalarField struct {
	OwnerID   uint   `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

type adminMetrics struct {
	ID    uint `gorm:"primaryKey"`
	Count int  `json:"count"`
}

type adminTaggedRole struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type adminTaggedUser struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `json:"name"`
	Password string `gorm:"not null" json:"-" admin:"component:password;create;update;readonly:false"`
	RoleIDs  []uint `gorm:"-" json:"role_ids" admin:"label:Roles;relation:roles"`
}

type adminExplicitAccessUser struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `json:"name"`
	Password   string `gorm:"not null" json:"password" ninja:"writeOnly"`
	InviteCode string `json:"invite_code" ninja:"createOnly"`
	StatusNote string `json:"status_note" crud:"updateOnly"`
}

func newAdminAPI(t *testing.T, site *Site, seed ...adminUser) *ninja.NinjaAPI {
	api, _ := newAdminAPIWithDB(t, site, seed...)
	return api
}

func newAdminAPIWithDB(t *testing.T, site *Site, seed ...adminUser) (*ninja.NinjaAPI, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&adminUser{}, &adminProject{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, item := range seed {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{Title: "admin test", DisableGinDefault: true})
	api.UseGin(orm.Middleware(db))
	router := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
	router.UseGin(func(c *gin.Context) {
		if c.GetHeader("X-User-ID") != "" {
			c.Set(contextkeys.JWTClaims, testClaims{id: 1})
		}
		c.Next()
	})
	site.Mount(router)
	api.AddRouter(router)
	return api, db
}

func performJSON(t *testing.T, api *ninja.NinjaAPI, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestAdminSiteMetadataAndCRUD(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Path:         "/users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email", "is_admin", "created_at"},
		DetailFields: []string{"id", "name", "email", "age", "is_admin", "created_at", "updated_at"},
		CreateFields: []string{"name", "email", "password", "age", "is_admin"},
		UpdateFields: []string{"name", "email", "password", "age", "is_admin"},
		FilterFields: []string{"id", "is_admin", "created_at"},
		SortFields:   []string{"id", "name", "email", "created_at"},
		SearchFields: []string{"name", "email"},
		FieldOptions: map[string]FieldOptions{
			"password": {Component: "password", Create: boolPtr(true), Update: boolPtr(true)},
		},
		BeforeCreate: func(ctx *ninja.Context, values map[string]any) error {
			if password, ok := values["password"].(string); ok {
				values["password"] = "hashed:" + password
			}
			return nil
		},
		BeforeUpdate: func(ctx *ninja.Context, current any, values map[string]any) error {
			if password, ok := values["password"].(string); ok {
				values["password"] = "hashed:" + password
			}
			return nil
		},
		Permissions: func(ctx *ninja.Context, action Action, resource *Resource) error {
			if action == ActionDelete && ctx.GetHeader("X-Admin") != "true" {
				return ninja.ForbiddenError()
			}
			return nil
		},
	})

	api := newAdminAPI(t, site, adminUser{
		Name:      "Alice",
		Email:     "alice@example.com",
		Password:  "hashed:password123",
		Age:       20,
		IsAdmin:   true,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	})

	metaResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/meta", nil, map[string]string{"X-User-ID": "1"})
	if metaResp.Code != http.StatusOK {
		t.Fatalf("meta status = %d body=%s", metaResp.Code, metaResp.Body.String())
	}
	var meta ResourceMetadata
	if err := json.NewDecoder(metaResp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if !containsName(meta.CreateFields, "password") {
		t.Fatalf("expected password create field, got %+v", meta.CreateFields)
	}
	var passwordField *FieldMeta
	for i := range meta.Fields {
		if meta.Fields[i].Name == "password" {
			passwordField = &meta.Fields[i]
			break
		}
	}
	if passwordField == nil || passwordField.Component != "password" || passwordField.List || passwordField.Detail {
		t.Fatalf("unexpected password metadata: %+v", passwordField)
	}

	createResp := performJSON(t, api, http.MethodPost, "/admin/resources/users", map[string]any{
		"name":      "Bob",
		"email":     "bob@example.com",
		"password":  "secret123",
		"age":       25,
		"is_admin":  false,
		"createdAt": "ignored",
	}, map[string]string{"X-User-ID": "1"})
	if createResp.Code != http.StatusBadRequest {
		t.Fatalf("expected create to reject unknown field, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	createResp = performJSON(t, api, http.MethodPost, "/admin/resources/users", map[string]any{
		"name":     "Bob",
		"email":    "bob@example.com",
		"password": "secret123",
		"age":      25,
		"is_admin": false,
	}, map[string]string{"X-User-ID": "1"})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createResp.Code, createResp.Body.String())
	}
	var created ResourceRecordOutput
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Item["email"] != "bob@example.com" {
		t.Fatalf("unexpected create payload: %+v", created.Item)
	}
	if _, ok := created.Item["password"]; ok {
		t.Fatalf("password must not be returned: %+v", created.Item)
	}

	listResp := performJSON(t, api, http.MethodGet, "/admin/resources/users?search=bob&sort=-name&is_admin=false", nil, map[string]string{"X-User-ID": "1"})
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var page ResourceListOutput
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0]["name"] != "Bob" {
		t.Fatalf("unexpected list payload: %+v", page)
	}

	sortedByIDResp := performJSON(t, api, http.MethodGet, "/admin/resources/users?sort=-id", nil, map[string]string{"X-User-ID": "1"})
	if sortedByIDResp.Code != http.StatusOK {
		t.Fatalf("sort by id status = %d body=%s", sortedByIDResp.Code, sortedByIDResp.Body.String())
	}
	var sortedByID ResourceListOutput
	if err := json.NewDecoder(sortedByIDResp.Body).Decode(&sortedByID); err != nil {
		t.Fatalf("decode sorted by id list: %v", err)
	}
	if len(sortedByID.Items) < 2 || fmt.Sprint(sortedByID.Items[0]["id"]) != "2" || fmt.Sprint(sortedByID.Items[1]["id"]) != "1" {
		t.Fatalf("unexpected sort by id payload: %+v", sortedByID)
	}

	filteredByIDResp := performJSON(t, api, http.MethodGet, "/admin/resources/users?id=2", nil, map[string]string{"X-User-ID": "1"})
	if filteredByIDResp.Code != http.StatusOK {
		t.Fatalf("filter by id status = %d body=%s", filteredByIDResp.Code, filteredByIDResp.Body.String())
	}
	var filteredByID ResourceListOutput
	if err := json.NewDecoder(filteredByIDResp.Body).Decode(&filteredByID); err != nil {
		t.Fatalf("decode filtered by id list: %v", err)
	}
	if filteredByID.Total != 1 || len(filteredByID.Items) != 1 || fmt.Sprint(filteredByID.Items[0]["id"]) != "2" {
		t.Fatalf("unexpected filter by id payload: %+v", filteredByID)
	}

	updateResp := performJSON(t, api, http.MethodPut, "/admin/resources/users/2", map[string]any{
		"name":     "Bobby",
		"password": "updated123",
		"is_admin": true,
	}, map[string]string{"X-User-ID": "1"})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	var updated ResourceRecordOutput
	if err := json.NewDecoder(updateResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Item["name"] != "Bobby" || updated.Item["is_admin"] != true {
		t.Fatalf("unexpected update payload: %+v", updated.Item)
	}

	deleteResp := performJSON(t, api, http.MethodDelete, "/admin/resources/users/2", nil, map[string]string{"X-User-ID": "1"})
	if deleteResp.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden delete, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	deleteResp = performJSON(t, api, http.MethodDelete, "/admin/resources/users/2", nil, map[string]string{"X-User-ID": "1", "X-Admin": "true"})
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestAdminSiteInfersResourceIdentityFromModel(t *testing.T) {
	site := NewSite()
	site.MustRegister(&Resource{
		Model:        autoResourceUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name"},
	})

	resource := site.byName["auto-resource-users"]
	if resource == nil {
		t.Fatalf("expected inferred resource to be registered")
	}
	if resource.Label != "Auto Resource Users" {
		t.Fatalf("expected inferred label, got %q", resource.Label)
	}
	if resource.Path != "/auto-resource-users" {
		t.Fatalf("expected inferred path, got %q", resource.Path)
	}
}

func TestAdminCreateReportsSoftDeletedDuplicateConflict(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
		CreateFields: []string{"name", "email", "password"},
		UpdateFields: []string{"name", "email", "password"},
		FieldOptions: map[string]FieldOptions{
			"password": {Create: boolPtr(true), Update: boolPtr(true), Component: "password"},
		},
	})

	api, db := newAdminAPIWithDB(t, site, adminUser{
		Name:     "Deleted Bob",
		Email:    "bob@example.com",
		Password: "secret123",
	})

	var deleted adminUser
	if err := db.Unscoped().Where("email = ?", "bob@example.com").First(&deleted).Error; err != nil {
		t.Fatalf("load seeded user: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("soft delete seeded user: %v", err)
	}

	createResp := performJSON(t, api, http.MethodPost, "/admin/resources/users", map[string]any{
		"name":     "Bob",
		"email":    "bob@example.com",
		"password": "secret123",
	}, map[string]string{"X-User-ID": "1"})
	if createResp.Code != http.StatusConflict {
		t.Fatalf("expected conflict for soft-deleted duplicate, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode conflict payload: %v", err)
	}
	if payload.Error.Code != "SOFT_DELETED_CONFLICT" {
		t.Fatalf("expected SOFT_DELETED_CONFLICT, got %+v", payload.Error)
	}
	if payload.Error.Message != "a soft-deleted record with the same value for field(s): email already exists; restore or permanently remove it before saving" {
		t.Fatalf("expected soft-delete guidance in message, got %+v", payload.Error)
	}
}

func TestAdminSiteBulkDeleteAndAuth(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
		CreateFields: []string{"name", "email", "password"},
		UpdateFields: []string{"name"},
		FieldOptions: map[string]FieldOptions{
			"password": {Create: boolPtr(true), Update: boolPtr(true), Component: "password"},
		},
	})

	api := newAdminAPI(t, site,
		adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"},
		adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2"},
		adminUser{Name: "Cara", Email: "cara@example.com", Password: "p3"},
	)

	unauthorized := performJSON(t, api, http.MethodGet, "/admin/resources/users", nil, nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized list, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	bulkDelete := performJSON(t, api, http.MethodPost, "/admin/resources/users/bulk-delete", map[string]any{
		"ids": []uint{1, 2},
	}, map[string]string{"X-User-ID": "1"})
	if bulkDelete.Code != http.StatusCreated {
		t.Fatalf("bulk delete status = %d body=%s", bulkDelete.Code, bulkDelete.Body.String())
	}
	var deleted BulkDeleteOutput
	if err := json.NewDecoder(bulkDelete.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode bulk delete: %v", err)
	}
	if deleted.Deleted != 2 {
		t.Fatalf("expected 2 deleted rows, got %+v", deleted)
	}

	listResp := performJSON(t, api, http.MethodGet, "/admin/resources/users?search=@example.com", nil, map[string]string{"X-User-ID": "1"})
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), "Cara") || strings.Contains(listResp.Body.String(), "Alice") {
		t.Fatalf("unexpected remaining users: %s", listResp.Body.String())
	}
}

func TestAdminSiteQueryScopeAppliesToItemAccessAndBulkDelete(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email", "is_admin"},
		DetailFields: []string{"id", "name", "email", "is_admin"},
		UpdateFields: []string{"name"},
		QueryScope: func(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
			return db.Where("is_admin = ?", false)
		},
	})

	api := newAdminAPI(t, site,
		adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1", IsAdmin: true},
		adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2", IsAdmin: false},
		adminUser{Name: "Cara", Email: "cara@example.com", Password: "p3", IsAdmin: false},
	)

	headers := map[string]string{"X-User-ID": "1"}

	listResp := performJSON(t, api, http.MethodGet, "/admin/resources/users", nil, headers)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	if strings.Contains(listResp.Body.String(), "Alice") || !strings.Contains(listResp.Body.String(), "Bob") || !strings.Contains(listResp.Body.String(), "Cara") {
		t.Fatalf("unexpected scoped list: %s", listResp.Body.String())
	}

	detailResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/1", nil, headers)
	if detailResp.Code != http.StatusNotFound {
		t.Fatalf("expected scoped detail to hide Alice, got %d body=%s", detailResp.Code, detailResp.Body.String())
	}

	updateResp := performJSON(t, api, http.MethodPut, "/admin/resources/users/1", map[string]any{"name": "Blocked"}, headers)
	if updateResp.Code != http.StatusNotFound {
		t.Fatalf("expected scoped update to hide Alice, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	updateResp = performJSON(t, api, http.MethodPut, "/admin/resources/users/2", map[string]any{"name": "Bobby"}, headers)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("scoped update status = %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	if !strings.Contains(updateResp.Body.String(), "Bobby") {
		t.Fatalf("expected updated name in response, got %s", updateResp.Body.String())
	}

	deleteResp := performJSON(t, api, http.MethodDelete, "/admin/resources/users/1", nil, headers)
	if deleteResp.Code != http.StatusNotFound {
		t.Fatalf("expected scoped delete to hide Alice, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	bulkDeleteResp := performJSON(t, api, http.MethodPost, "/admin/resources/users/bulk-delete", map[string]any{
		"ids": []uint{1, 3},
	}, headers)
	if bulkDeleteResp.Code != http.StatusCreated {
		t.Fatalf("bulk delete status = %d body=%s", bulkDeleteResp.Code, bulkDeleteResp.Body.String())
	}
	var deleted BulkDeleteOutput
	if err := json.NewDecoder(bulkDeleteResp.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode bulk delete: %v", err)
	}
	if deleted.Deleted != 1 {
		t.Fatalf("expected scoped bulk delete to remove only one record, got %+v", deleted)
	}

	listAfterDelete := performJSON(t, api, http.MethodGet, "/admin/resources/users", nil, headers)
	if listAfterDelete.Code != http.StatusOK {
		t.Fatalf("post-delete list status = %d body=%s", listAfterDelete.Code, listAfterDelete.Body.String())
	}
	if strings.Contains(listAfterDelete.Body.String(), "Cara") || strings.Contains(listAfterDelete.Body.String(), "Alice") || !strings.Contains(listAfterDelete.Body.String(), "Bobby") {
		t.Fatalf("unexpected scoped list after delete: %s", listAfterDelete.Body.String())
	}
}

func TestAdminSiteFiltersResourcesAndMetadataActionsByPermission(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
		CreateFields: []string{"name", "email", "password"},
		UpdateFields: []string{"name"},
		FieldOptions: map[string]FieldOptions{
			"password": {Create: boolPtr(true), Update: boolPtr(true), Component: "password"},
		},
		Permissions: func(ctx *ninja.Context, action Action, resource *Resource) error {
			switch action {
			case ActionCreate, ActionDelete, ActionBulkDelete:
				if ctx.GetHeader("X-Admin") != "true" {
					return ninja.ForbiddenError()
				}
			}
			return nil
		},
	})
	site.MustRegister(&Resource{
		Name:         "audits",
		Model:        adminUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name"},
		Permissions: func(ctx *ninja.Context, action Action, resource *Resource) error {
			if action == ActionList || action == ActionDetail {
				return ninja.ForbiddenError()
			}
			return nil
		},
	})

	api := newAdminAPI(t, site, adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"})

	unauthorizedIndex := performJSON(t, api, http.MethodGet, "/admin/resources", nil, nil)
	if unauthorizedIndex.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized resources index, got %d body=%s", unauthorizedIndex.Code, unauthorizedIndex.Body.String())
	}

	headers := map[string]string{"X-User-ID": "1"}
	indexResp := performJSON(t, api, http.MethodGet, "/admin/resources", nil, headers)
	if indexResp.Code != http.StatusOK {
		t.Fatalf("resources index status = %d body=%s", indexResp.Code, indexResp.Body.String())
	}
	var index ResourceIndex
	if err := json.NewDecoder(indexResp.Body).Decode(&index); err != nil {
		t.Fatalf("decode index: %v", err)
	}
	if len(index.Resources) != 1 || index.Resources[0].Name != "users" {
		t.Fatalf("unexpected visible resources: %+v", index.Resources)
	}

	hiddenMeta := performJSON(t, api, http.MethodGet, "/admin/resources/audits/meta", nil, headers)
	if hiddenMeta.Code != http.StatusForbidden {
		t.Fatalf("expected hidden resource metadata 403, got %d body=%s", hiddenMeta.Code, hiddenMeta.Body.String())
	}

	metaResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/meta", nil, headers)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("metadata status = %d body=%s", metaResp.Code, metaResp.Body.String())
	}
	var meta ResourceMetadata
	if err := json.NewDecoder(metaResp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	for _, action := range []Action{ActionList, ActionDetail, ActionUpdate} {
		if !containsAction(meta.Actions, action) {
			t.Fatalf("expected %q in metadata actions, got %+v", action, meta.Actions)
		}
	}
	for _, action := range []Action{ActionCreate, ActionDelete, ActionBulkDelete} {
		if containsAction(meta.Actions, action) {
			t.Fatalf("expected %q to be hidden from metadata actions, got %+v", action, meta.Actions)
		}
	}

	adminMetaResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/meta", nil, map[string]string{"X-User-ID": "1", "X-Admin": "true"})
	if adminMetaResp.Code != http.StatusOK {
		t.Fatalf("admin metadata status = %d body=%s", adminMetaResp.Code, adminMetaResp.Body.String())
	}
	var adminMeta ResourceMetadata
	if err := json.NewDecoder(adminMetaResp.Body).Decode(&adminMeta); err != nil {
		t.Fatalf("decode admin metadata: %v", err)
	}
	for _, action := range []Action{ActionCreate, ActionDelete, ActionBulkDelete} {
		if !containsAction(adminMeta.Actions, action) {
			t.Fatalf("expected %q in admin metadata actions, got %+v", action, adminMeta.Actions)
		}
	}
}

func TestAdminSiteFieldLevelPermissionsAffectMetadataSerializationAndWrites(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegister(&Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email", "is_admin"},
		DetailFields: []string{"id", "name", "email", "is_admin"},
		CreateFields: []string{"name", "email", "password", "is_admin"},
		UpdateFields: []string{"name", "email", "is_admin"},
		FieldOptions: map[string]FieldOptions{
			"password": {Create: boolPtr(true), Update: boolPtr(true), Component: "password"},
		},
		FieldPermissions: func(ctx *ninja.Context, resource *Resource, meta *FieldMeta) {
			if ctx.GetHeader("X-Admin") == "true" {
				return
			}
			switch meta.Name {
			case "email":
				meta.List = false
				meta.Detail = false
				meta.Filterable = false
				meta.Sortable = false
				meta.Searchable = false
			case "is_admin":
				meta.List = false
				meta.Detail = false
				meta.Create = false
				meta.Update = false
				meta.Filterable = false
				meta.Sortable = false
				meta.Searchable = false
			}
		},
	})

	api := newAdminAPI(t, site, adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1", IsAdmin: true})
	headers := map[string]string{"X-User-ID": "1"}

	metaResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/meta", nil, headers)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("metadata status = %d body=%s", metaResp.Code, metaResp.Body.String())
	}
	var meta ResourceMetadata
	if err := json.NewDecoder(metaResp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if containsName(meta.ListFields, "email") || containsName(meta.ListFields, "is_admin") {
		t.Fatalf("expected restricted fields to be hidden from list metadata, got %+v", meta.ListFields)
	}
	if containsName(meta.CreateFields, "is_admin") || containsName(meta.UpdateFields, "is_admin") {
		t.Fatalf("expected is_admin to be hidden from write metadata, got create=%+v update=%+v", meta.CreateFields, meta.UpdateFields)
	}

	listResp := performJSON(t, api, http.MethodGet, "/admin/resources/users", nil, headers)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var page ResourceListOutput
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("unexpected list payload: %+v", page)
	}
	if _, ok := page.Items[0]["email"]; ok {
		t.Fatalf("expected email to be hidden from list payload: %+v", page.Items[0])
	}
	if _, ok := page.Items[0]["is_admin"]; ok {
		t.Fatalf("expected is_admin to be hidden from list payload: %+v", page.Items[0])
	}

	createResp := performJSON(t, api, http.MethodPost, "/admin/resources/users", map[string]any{
		"name":     "Bob",
		"email":    "bob@example.com",
		"password": "secret123",
		"is_admin": true,
	}, headers)
	if createResp.Code != http.StatusBadRequest {
		t.Fatalf("expected restricted write field to be rejected, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	adminMetaResp := performJSON(t, api, http.MethodGet, "/admin/resources/users/meta", nil, map[string]string{"X-User-ID": "1", "X-Admin": "true"})
	if adminMetaResp.Code != http.StatusOK {
		t.Fatalf("admin metadata status = %d body=%s", adminMetaResp.Code, adminMetaResp.Body.String())
	}
	var adminMeta ResourceMetadata
	if err := json.NewDecoder(adminMetaResp.Body).Decode(&adminMeta); err != nil {
		t.Fatalf("decode admin metadata: %v", err)
	}
	if !containsName(adminMeta.ListFields, "email") || !containsName(adminMeta.CreateFields, "is_admin") || !containsName(adminMeta.UpdateFields, "is_admin") {
		t.Fatalf("expected admin metadata to include restricted fields, got %+v", adminMeta)
	}
}

func TestAdminSiteRowPermissionsAndRelationSelectors(t *testing.T) {
	site := NewSite(WithPermissionChecker(func(ctx *ninja.Context, action Action, resource *Resource) error {
		if ctx.GetUserID() == 0 {
			return ninja.UnauthorizedError()
		}
		return nil
	}))
	site.MustRegisterModel(&ModelResource{
		Name:         "projects",
		Model:        adminProject{},
		ListFields:   []string{"id", "title", "owner_id"},
		DetailFields: []string{"id", "title", "owner_id", "secret"},
		CreateFields: []string{"title", "owner_id", "secret"},
		UpdateFields: []string{"title", "owner_id", "secret"},
		RowPermissions: RowPermissionFunc(func(ctx *ninja.Context, action Action, resource *Resource, db *gorm.DB) *gorm.DB {
			return db.Where("owner_id = ?", ctx.GetUserID())
		}),
	})
	site.MustRegisterModel(&ModelResource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
		SearchFields: []string{"name", "email"},
	})

	api, db := newAdminAPIWithDB(t, site,
		adminUser{Name: "Alice", Email: "alice@example.com", Password: "p1"},
		adminUser{Name: "Bob", Email: "bob@example.com", Password: "p2"},
	)
	if err := db.Create(&adminProject{Title: "Alice Project", OwnerID: 1, Secret: "alpha"}).Error; err != nil {
		t.Fatalf("seed project 1: %v", err)
	}
	if err := db.Create(&adminProject{Title: "Bob Project", OwnerID: 2, Secret: "beta"}).Error; err != nil {
		t.Fatalf("seed project 2: %v", err)
	}

	headers := map[string]string{"X-User-ID": "1"}

	metaResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects/meta", nil, headers)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("metadata status = %d body=%s", metaResp.Code, metaResp.Body.String())
	}
	var meta ResourceMetadata
	if err := json.NewDecoder(metaResp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	var ownerField *FieldMeta
	for i := range meta.Fields {
		if meta.Fields[i].Name == "owner_id" {
			ownerField = &meta.Fields[i]
			break
		}
	}
	if ownerField == nil || ownerField.Component != "select" || ownerField.Relation == nil || ownerField.Relation.Resource != "users" {
		t.Fatalf("expected relation-backed owner field metadata, got %+v", ownerField)
	}
	if ownerField.Relation.LabelField != "name" || !containsName(ownerField.Relation.SearchFields, "email") {
		t.Fatalf("expected inferred relation label/search fields, got %+v", ownerField.Relation)
	}

	listResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects", nil, headers)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var page ResourceListOutput
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0]["title"] != "Alice Project" {
		t.Fatalf("unexpected row-scoped list payload: %+v", page)
	}

	detailResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects/2", nil, headers)
	if detailResp.Code != http.StatusNotFound {
		t.Fatalf("expected row-scoped detail to hide Bob's project, got %d body=%s", detailResp.Code, detailResp.Body.String())
	}

	optionsResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects/fields/owner_id/options?search=ali", nil, headers)
	if optionsResp.Code != http.StatusOK {
		t.Fatalf("options status = %d body=%s", optionsResp.Code, optionsResp.Body.String())
	}
	var options RelationOptionsOutput
	if err := json.NewDecoder(optionsResp.Body).Decode(&options); err != nil {
		t.Fatalf("decode relation options: %v", err)
	}
	if options.Total != 1 || len(options.Items) != 1 || options.Items[0].Label != "Alice" || fmt.Sprint(options.Items[0].Value) != "1" {
		t.Fatalf("unexpected relation options payload: %+v", options)
	}

	idOptionsResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects/fields/owner_id/options?search=1", nil, headers)
	if idOptionsResp.Code != http.StatusOK {
		t.Fatalf("id options status = %d body=%s", idOptionsResp.Code, idOptionsResp.Body.String())
	}
	var idOptions RelationOptionsOutput
	if err := json.NewDecoder(idOptionsResp.Body).Decode(&idOptions); err != nil {
		t.Fatalf("decode id relation options: %v", err)
	}
	if idOptions.Total != 1 || len(idOptions.Items) != 1 || idOptions.Items[0].Label != "Alice" || fmt.Sprint(idOptions.Items[0].Value) != "1" {
		t.Fatalf("unexpected id relation options payload: %+v", idOptions)
	}

	missingIDOptionsResp := performJSON(t, api, http.MethodGet, "/admin/resources/projects/fields/owner_id/options?search=999", nil, headers)
	if missingIDOptionsResp.Code != http.StatusOK {
		t.Fatalf("missing id options status = %d body=%s", missingIDOptionsResp.Code, missingIDOptionsResp.Body.String())
	}
	var missingIDOptions RelationOptionsOutput
	if err := json.NewDecoder(missingIDOptionsResp.Body).Decode(&missingIDOptions); err != nil {
		t.Fatalf("decode missing id relation options: %v", err)
	}
	if missingIDOptions.Total != 0 || len(missingIDOptions.Items) != 0 {
		t.Fatalf("expected empty missing-id relation options payload: %+v", missingIDOptions)
	}
}

func TestInferResourceName(t *testing.T) {
	if got := inferResourceName(reflect.TypeOf(adminUser{})); got != "admin-users" {
		t.Fatalf("inferResourceName(adminUser) = %q", got)
	}
	if got := inferResourceName(reflect.TypeOf(autoResourceUser{})); got != "auto-resource-users" {
		t.Fatalf("inferResourceName(autoResourceUser) = %q", got)
	}
	if got := inferResourceName(reflect.TypeOf(APIKey{})); got != "api-keys" {
		t.Fatalf("inferResourceName(APIKey) = %q", got)
	}
	if got := inferResourceName(reflect.TypeOf(Person{})); got != "people" {
		t.Fatalf("inferResourceName(Person) = %q", got)
	}
}

type APIKey struct {
	ID uint `gorm:"primaryKey"`
}

type Person struct {
	ID uint `gorm:"primaryKey"`
}

type adminAcronymColumns struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `json:"user_id"`
	APIKey string `json:"api_key"`
}

func TestCollectFieldsInferAutoRelation(t *testing.T) {
	tests := []struct {
		name      string
		model     any
		fieldName string
		want      bool
	}{
		{name: "owner id", model: adminOwnerByID{}, fieldName: "owner_id", want: true},
		{name: "owner Id", model: adminOwnerById{}, fieldName: "owner_id", want: true},
		{name: "missing relation field", model: adminOwnerWithoutRelation{}, fieldName: "owner_id", want: false},
		{name: "scalar relation field", model: adminOwnerWithScalarField{}, fieldName: "owner_id", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := collectFields(reflect.TypeOf(tt.model), nil, nil)
			var match *fieldMeta
			for _, field := range fields {
				if field.Meta.Name == tt.fieldName {
					match = field
					break
				}
			}
			if match == nil {
				t.Fatalf("field %q not found", tt.fieldName)
			}
			if got := match.autoRelation != nil; got != tt.want {
				t.Fatalf("autoRelation = %v", got)
			}
		})
	}
}

func TestCollectFieldsUsesStableSnakeCaseColumnsForAcronyms(t *testing.T) {
	fields := collectFields(reflect.TypeOf(adminAcronymColumns{}), nil, nil)
	columns := map[string]string{}
	for _, field := range fields {
		columns[field.Meta.Name] = field.Meta.Column
	}
	if got := columns["user_id"]; got != "user_id" {
		t.Fatalf("expected user_id column, got %q", got)
	}
	if got := columns["api_key"]; got != "api_key" {
		t.Fatalf("expected api_key column, got %q", got)
	}
}

func TestInferRelationLabelAndSearchFields(t *testing.T) {
	userResource := &Resource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name", "email"},
		DetailFields: []string{"id", "name", "email"},
	}
	if err := userResource.prepare(); err != nil {
		t.Fatalf("prepare user resource: %v", err)
	}
	if label := inferRelationLabelField(userResource); label != "name" {
		t.Fatalf("label field = %q", label)
	}
	searchFields := inferRelationSearchFields(userResource, "name")
	if len(searchFields) != 2 || searchFields[0] != "name" || searchFields[1] != "email" {
		t.Fatalf("search fields = %+v", searchFields)
	}

	metricsResource := &Resource{
		Name:         "metrics",
		Model:        adminMetrics{},
		ListFields:   []string{"id", "count"},
		DetailFields: []string{"id", "count"},
	}
	if err := metricsResource.prepare(); err != nil {
		t.Fatalf("prepare metrics resource: %v", err)
	}
	if label := inferRelationLabelField(metricsResource); label != "id" {
		t.Fatalf("metrics label field = %q", label)
	}
	if searchFields := inferRelationSearchFields(metricsResource, "id"); len(searchFields) != 0 {
		t.Fatalf("metrics search fields = %+v", searchFields)
	}
}

func TestAdminTagRelationAndPasswordInference(t *testing.T) {
	site := NewSite()
	site.MustRegisterModel(&ModelResource{
		Name:         "roles",
		Model:        adminTaggedRole{},
		ListFields:   []string{"id", "name", "code"},
		DetailFields: []string{"id", "name", "code"},
		SearchFields: []string{"name", "code"},
	})
	site.MustRegisterModel(&ModelResource{
		Name:         "users",
		Model:        adminTaggedUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name", "role_ids"},
		CreateFields: []string{"name", "password", "role_ids"},
		UpdateFields: []string{"name", "password", "role_ids"},
	})

	user := site.byName["users"]
	if user == nil {
		t.Fatalf("expected tagged user resource")
	}
	passwordField := user.fieldByName["password"]
	if passwordField == nil || passwordField.Meta.Component != "password" || !passwordField.Meta.Create || !passwordField.Meta.Update || passwordField.Meta.Detail {
		t.Fatalf("unexpected password field metadata: %+v", passwordField)
	}
	roleField := user.fieldByName["role_ids"]
	if roleField == nil || roleField.Meta.Relation == nil {
		t.Fatalf("expected role_ids relation metadata")
	}
	if roleField.Meta.Relation.Resource != "roles" || roleField.Meta.Relation.ValueField != "id" || roleField.Meta.Relation.LabelField != "name" {
		t.Fatalf("unexpected role_ids relation: %+v", roleField.Meta.Relation)
	}
	if !containsName(roleField.Meta.Relation.SearchFields, "code") {
		t.Fatalf("expected inferred relation search fields, got %+v", roleField.Meta.Relation.SearchFields)
	}
}

func TestAdminExplicitFieldAccessTagsAffectMetadata(t *testing.T) {
	site := NewSite()
	site.MustRegisterModel(&ModelResource{
		Name:         "users",
		Model:        adminExplicitAccessUser{},
		ListFields:   []string{"id", "name", "invite_code", "status_note"},
		DetailFields: []string{"id", "name", "invite_code", "status_note"},
		CreateFields: []string{"name", "password", "invite_code"},
		UpdateFields: []string{"name", "password", "status_note"},
	})

	user := site.byName["users"]
	if user == nil {
		t.Fatalf("expected explicit access user resource")
	}
	passwordField := user.fieldByName["password"]
	if passwordField == nil {
		t.Fatalf("expected password field metadata")
	}
	if passwordField.Meta.ReadOnly {
		t.Fatalf("expected writeOnly field to stay writable, got %+v", passwordField.Meta)
	}
	if passwordField.Meta.List || passwordField.Meta.Detail {
		t.Fatalf("expected writeOnly field to be hidden from read metadata, got %+v", passwordField.Meta)
	}
	if !passwordField.Meta.Create || !passwordField.Meta.Update {
		t.Fatalf("expected writeOnly field to stay writable in both create and update, got %+v", passwordField.Meta)
	}
	if containsName(user.metadata.ListFields, "password") || containsName(user.metadata.DetailFields, "password") {
		t.Fatalf("expected writeOnly field to be hidden from read metadata, got %+v", user.metadata)
	}

	inviteField := user.fieldByName["invite_code"]
	if inviteField == nil {
		t.Fatalf("expected invite_code field metadata")
	}
	if !inviteField.Meta.Create {
		t.Fatalf("expected createOnly field to be writable on create, got %+v", inviteField.Meta)
	}
	if inviteField.Meta.Update {
		t.Fatalf("expected createOnly field to be hidden from update metadata, got %+v", inviteField.Meta)
	}
	if !containsName(user.metadata.CreateFields, "invite_code") || containsName(user.metadata.UpdateFields, "invite_code") {
		t.Fatalf("expected createOnly field to only appear in create metadata, got %+v", user.metadata)
	}

	statusField := user.fieldByName["status_note"]
	if statusField == nil {
		t.Fatalf("expected status_note field metadata")
	}
	if statusField.Meta.Create {
		t.Fatalf("expected updateOnly field to be hidden from create metadata, got %+v", statusField.Meta)
	}
	if !statusField.Meta.Update {
		t.Fatalf("expected updateOnly field to be writable on update, got %+v", statusField.Meta)
	}
	if containsName(user.metadata.CreateFields, "status_note") || !containsName(user.metadata.UpdateFields, "status_note") {
		t.Fatalf("expected updateOnly field to only appear in update metadata, got %+v", user.metadata)
	}
	if !containsName(user.metadata.ListFields, "invite_code") || !containsName(user.metadata.DetailFields, "status_note") {
		t.Fatalf("expected non-writeOnly access-tagged fields to remain readable, got %+v", user.metadata)
	}
}

func TestAdminSiteAmbiguousModelDisablesAutoRelationResolution(t *testing.T) {
	site := NewSite()
	site.MustRegisterModel(&ModelResource{
		Name:         "users",
		Model:        adminUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name"},
	})
	site.MustRegisterModel(&ModelResource{
		Name:         "projects",
		Model:        adminProject{},
		ListFields:   []string{"id", "title", "owner_id"},
		DetailFields: []string{"id", "title", "owner_id"},
	})
	site.MustRegisterModel(&ModelResource{
		Name:         "staff",
		Model:        adminUser{},
		ListFields:   []string{"id", "name"},
		DetailFields: []string{"id", "name"},
	})

	project := site.byName["projects"]
	if project == nil {
		t.Fatalf("expected projects resource")
	}
	ownerField := project.fieldByName["owner_id"]
	if ownerField == nil || ownerField.Meta.Relation == nil {
		t.Fatalf("expected owner relation metadata")
	}
	if ownerField.Meta.Relation.Resource != "" {
		t.Fatalf("expected ambiguous model to skip relation resolution, got %+v", ownerField.Meta.Relation)
	}
}

func TestHumanizeHandlesEmptyAndUnicodeParts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "separators only", input: "__--__", want: ""},
		{name: "unicode", input: "éclair_name", want: "Éclair Name"},
		{name: "camel case", input: "userID", want: "User Id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanize(tt.input); got != tt.want {
				t.Fatalf("humanize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func containsAction(set []Action, action Action) bool {
	for _, current := range set {
		if current == action {
			return true
		}
	}
	return false
}
