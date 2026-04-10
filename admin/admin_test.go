package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newAdminAPI(t *testing.T, site *Site, seed ...adminUser) *ninja.NinjaAPI {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&adminUser{}); err != nil {
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
	return api
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
		FilterFields: []string{"is_admin", "created_at"},
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

func TestHumanizeHandlesEmptyAndUnicodeParts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "separators only", input: "__--__", want: ""},
		{name: "unicode", input: "éXample_name", want: "Éxample Name"},
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
