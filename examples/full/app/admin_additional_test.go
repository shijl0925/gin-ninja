package app

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type exampleAdminClaims struct{ id uint }

func (c exampleAdminClaims) GetUserID() uint { return c.id }

func newExampleAdminAPI(t *testing.T) (*ninja.NinjaAPI, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &Project{}, &userRole{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{Title: "admin example", DisableGinDefault: true})
	api.UseGin(orm.Middleware(db))
	router := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
	router.UseGin(func(c *gin.Context) {
		if raw := strings.TrimSpace(c.GetHeader("X-User-ID")); raw != "" {
			id, _ := strconv.Atoi(raw)
			c.Set(contextkeys.JWTClaims, exampleAdminClaims{id: uint(id)})
		}
		c.Next()
	})
	NewAdminSite().Mount(router)
	api.AddRouter(router)
	return api, db
}

func performExampleAdminRequest(api *ninja.NinjaAPI, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestExampleAdminSiteAuthorizationAndRowFiltering(t *testing.T) {
	api, db := newExampleAdminAPI(t)

	roleAdmin := Role{Name: "Admin", Code: "admin"}
	roleEditor := Role{Name: "Editor", Code: "editor"}
	if err := db.Create(&roleAdmin).Error; err != nil {
		t.Fatalf("create roleAdmin: %v", err)
	}
	if err := db.Create(&roleEditor).Error; err != nil {
		t.Fatalf("create roleEditor: %v", err)
	}
	owner := User{Name: "Owner", Email: "owner@example.com", Password: "password123", RoleIDs: []uint{roleAdmin.ID, roleEditor.ID}}
	other := User{Name: "Other", Email: "other@example.com", Password: "password123"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}
	if err := db.Create(&Project{Title: "Mine", Summary: "owned", OwnerID: owner.ID}).Error; err != nil {
		t.Fatalf("create owner project: %v", err)
	}
	if err := db.Create(&Project{Title: "Theirs", Summary: "other", OwnerID: other.ID}).Error; err != nil {
		t.Fatalf("create other project: %v", err)
	}

	unauthorized := performExampleAdminRequest(api, http.MethodGet, "/admin/resources", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", unauthorized.Code, unauthorized.Body.String())
	}

	resources := performExampleAdminRequest(api, http.MethodGet, "/admin/resources", map[string]string{"X-User-ID": strconv.FormatUint(uint64(owner.ID), 10)})
	if resources.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resources.Code, resources.Body.String())
	}
	body := resources.Body.String()
	for _, name := range []string{"users", "roles", "projects"} {
		if !strings.Contains(body, `"`+name+`"`) {
			t.Fatalf("expected resource %q in body %q", name, body)
		}
	}

	projects := performExampleAdminRequest(api, http.MethodGet, "/admin/resources/projects", map[string]string{"X-User-ID": strconv.FormatUint(uint64(owner.ID), 10)})
	if projects.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", projects.Code, projects.Body.String())
	}
	if !strings.Contains(projects.Body.String(), `"title":"Mine"`) || strings.Contains(projects.Body.String(), `"title":"Theirs"`) {
		t.Fatalf("expected row permissions to filter projects, got %q", projects.Body.String())
	}

	meta := performExampleAdminRequest(api, http.MethodGet, "/admin/resources/users/meta", map[string]string{"X-User-ID": strconv.FormatUint(uint64(owner.ID), 10)})
	if meta.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", meta.Code, meta.Body.String())
	}
	if !strings.Contains(meta.Body.String(), `"name":"role_ids"`) || !strings.Contains(meta.Body.String(), `"resource":"roles"`) {
		t.Fatalf("expected relation metadata for role_ids, got %q", meta.Body.String())
	}
}

func TestExampleAdminPrototypeAndModelHooks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ServeAdminPrototype(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Gin Ninja Admin") {
		t.Fatalf("expected admin prototype HTML, got %q", recorder.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = request
	ninjaCtx := &ninja.Context{Context: ginCtx}
	if err := requireAuthenticatedAdmin(ninjaCtx, "", nil); err == nil {
		t.Fatal("expected unauthenticated admin check to fail")
	}
	ginCtx.Set(contextkeys.JWTClaims, exampleAdminClaims{id: 1})
	if err := requireAuthenticatedAdmin(ninjaCtx, "", nil); err != nil {
		t.Fatalf("requireAuthenticatedAdmin(): %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"-hooks?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &userRole{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	roleOne := Role{Name: "Admin", Code: "admin"}
	roleTwo := Role{Name: "Editor", Code: "editor"}
	if err := db.Create(&roleOne).Error; err != nil {
		t.Fatalf("create roleOne: %v", err)
	}
	if err := db.Create(&roleTwo).Error; err != nil {
		t.Fatalf("create roleTwo: %v", err)
	}

	user := User{
		Name:     "  Alice  ",
		Email:    "  ALICE@EXAMPLE.COM  ",
		Password: "password123",
		RoleIDs:  []uint{roleTwo.ID, roleTwo.ID, roleOne.ID},
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Fatalf("expected normalized user fields, got %+v", user)
	}
	if !isHashedPassword(user.Password) {
		t.Fatalf("expected password to be hashed, got %q", user.Password)
	}
	if !reflect.DeepEqual(user.RoleIDs, []uint{roleTwo.ID, roleOne.ID}) {
		t.Fatalf("expected RoleIDs to be normalized, got %v", user.RoleIDs)
	}

	var links []userRole
	if err := db.Order("role_id").Find(&links).Error; err != nil {
		t.Fatalf("find userRole links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 userRole links, got %d", len(links))
	}

	var loaded User
	if err := db.Preload("Roles").First(&loaded, user.ID).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}
	if len(loaded.RoleIDs) != 2 {
		t.Fatalf("expected AfterFind to sync two RoleIDs, got %v", loaded.RoleIDs)
	}
	gotIDs := map[uint]struct{}{}
	for _, id := range loaded.RoleIDs {
		gotIDs[id] = struct{}{}
	}
	if _, ok := gotIDs[roleOne.ID]; !ok {
		t.Fatalf("expected roleOne ID in RoleIDs, got %v", loaded.RoleIDs)
	}
	if _, ok := gotIDs[roleTwo.ID]; !ok {
		t.Fatalf("expected roleTwo ID in RoleIDs, got %v", loaded.RoleIDs)
	}

	if err := syncUserRoles(db, &loaded, []uint{}); err != nil {
		t.Fatalf("syncUserRoles empty: %v", err)
	}
	if len(loaded.RoleIDs) != 0 || loaded.Roles != nil {
		t.Fatalf("expected empty role sync result, got %+v", loaded)
	}
	if err := syncUserRoles(db, &loaded, []uint{0}); err == nil {
		t.Fatal("expected zero role id error")
	}
	if err := syncUserRoles(db, &loaded, []uint{999999}); err == nil {
		t.Fatal("expected missing role error")
	}
	if err := syncUserRoles(db, nil, []uint{roleOne.ID}); err != nil {
		t.Fatalf("syncUserRoles nil user: %v", err)
	}

	shortPassword := User{Name: "Bob", Email: "bob@example.com", Password: "short"}
	if err := shortPassword.BeforeSave(db); err == nil {
		t.Fatal("expected short password validation error")
	}
	emptyPassword := User{Name: "Bob", Email: "bob@example.com", Password: "   "}
	if err := emptyPassword.BeforeSave(db); err != nil {
		t.Fatalf("BeforeSave empty password: %v", err)
	}
	if emptyPassword.Password != "" {
		t.Fatalf("expected blank password to stay blank, got %q", emptyPassword.Password)
	}
	if err := (&User{}).AfterFind(db); err != nil {
		t.Fatalf("AfterFind(): %v", err)
	}
}
