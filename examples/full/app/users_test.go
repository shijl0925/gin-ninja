package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRegisterTestAPI(t *testing.T) *ninja.NinjaAPI {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &Permission{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{Title: "Test", Version: "0.0.1"})
	authRouter := ninja.NewRouter("/auth", ninja.WithTags("Auth"))
	ninja.Post(authRouter, "/register", Register)
	api.AddRouter(authRouter)
	return api
}

func setupAppTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &Permission{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	orm.Init(db)
	settings.Global.JWT.Secret = "test-secret"
	settings.Global.JWT.ExpireHours = 24
	settings.Global.JWT.Issuer = "gin-ninja"
	return db
}

func newRBACExampleAPI(t *testing.T) *ninja.NinjaAPI {
	t.Helper()

	db := setupAppTestDB(t)
	if err := SeedRBAC(db); err != nil {
		t.Fatalf("SeedRBAC: %v", err)
	}

	api := ninja.New(ninja.Config{
		Title:   "Test",
		Version: "0.0.1",
		SecuritySchemes: map[string]ninja.SecurityScheme{
			"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
		},
	})
	api.UseGin(orm.Middleware(db))

	authRouter := ninja.NewRouter("/auth", ninja.WithTags("Auth"))
	ninja.Post(authRouter, "/login", Login)
	api.AddRouter(authRouter)

	usersRouter := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithBearerAuth())
	usersRouter.UseGin(middleware.JWTAuth())
	ninja.Get(usersRouter, "/", ListUsers,
		ninja.WithMiddleware(middleware.RequirePermissions(ResolvePermissions, PermissionUserList)))
	ninja.Post(usersRouter, "/", CreateUser,
		ninja.WithMiddleware(middleware.RequirePermissions(ResolvePermissions, PermissionUserCreate)))
	api.AddRouter(usersRouter)

	rbacRouter := ninja.NewRouter("/rbac", ninja.WithTags("RBAC"), ninja.WithBearerAuth())
	rbacRouter.UseGin(middleware.JWTAuth())
	ninja.Get(rbacRouter, "/me", GetCurrentSubject)
	api.AddRouter(rbacRouter)

	return api
}

func registerRequest(t *testing.T, api *ninja.NinjaAPI, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestRegister_SucceedsWithoutAuth(t *testing.T) {
	api := newRegisterTestAPI(t)

	w := registerRequest(t, api, RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Age:      18,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var out UserOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %q", out.Email)
	}
	if out.Name != "Alice" {
		t.Fatalf("expected name Alice, got %q", out.Name)
	}
}

func TestRegister_DuplicateEmailReturnsConflict(t *testing.T) {
	api := newRegisterTestAPI(t)

	first := RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Age:      18,
	}
	if w := registerRequest(t, api, first); w.Code != http.StatusCreated {
		t.Fatalf("expected first register to succeed, got %d: %s", w.Code, w.Body.String())
	}

	w := registerRequest(t, api, first)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHelpersAndAuthFlow(t *testing.T) {
	setupAppTestDB(t)

	if NewUserRepo() == nil {
		t.Fatal("expected repo instance")
	}

	hash := hashPassword("password123")
	if hash == "password123" || !checkPassword(hash, "password123") || checkPassword(hash, "wrong") {
		t.Fatal("expected bcrypt helper functions to work")
	}

	registered, err := Register(nil, &RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Age:      18,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if registered.Email != "alice@example.com" {
		t.Fatalf("unexpected register output: %+v", registered)
	}

	loginOut, err := Login(nil, &LoginInput{Email: "alice@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loginOut.Token == "" || loginOut.UserID == 0 || loginOut.Name != "Alice" {
		t.Fatalf("unexpected login output: %+v", loginOut)
	}

	if _, err := Login(nil, &LoginInput{Email: "alice@example.com", Password: "wrong"}); err == nil {
		t.Fatal("expected invalid password error")
	}
	if _, err := Login(nil, &LoginInput{Email: "missing@example.com", Password: "password123"}); err == nil {
		t.Fatal("expected missing user error")
	}

	duplicate, err := Register(nil, &RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Age:      18,
	})
	if err == nil || duplicate != nil {
		t.Fatalf("expected duplicate email error, got result=%+v err=%v", duplicate, err)
	}
}

func TestUserCRUDFunctions(t *testing.T) {
	setupAppTestDB(t)

	created, err := CreateUser(nil, &CreateUserInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Age:      18,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created id, got %+v", created)
	}

	second, err := CreateUser(nil, &CreateUserInput{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
		Age:      20,
	})
	if err != nil {
		t.Fatalf("CreateUser second: %v", err)
	}

	got, err := GetUser(nil, &GetUserInput{UserID: created.ID})
	if err != nil || got.Email != "alice@example.com" {
		t.Fatalf("GetUser: result=%+v err=%v", got, err)
	}

	page, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Search:    "Ali",
	})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Email != "alice@example.com" {
		t.Fatalf("unexpected list result: %+v", page)
	}

	updated, err := UpdateUser(nil, &UpdateUserInput{
		UserID: second.ID,
		Name:   "Bobby",
		Email:  "bobby@example.com",
		Age:    21,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Name != "Bobby" || updated.Email != "bobby@example.com" || updated.Age != 21 {
		t.Fatalf("unexpected updated user: %+v", updated)
	}

	if err := DeleteUser(nil, &DeleteUserInput{UserID: second.ID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	deleted, err := GetUser(nil, &GetUserInput{UserID: second.ID})
	if !errors.Is(err, ninja.ErrNotFound) || deleted != nil {
		t.Fatalf("expected deleted user to be missing, got result=%+v err=%v", deleted, err)
	}

	out := toUserOut(User{Email: "x@example.com", Name: "X", Age: 9, IsAdmin: true})
	if out.Email != "x@example.com" || !out.IsAdmin {
		t.Fatalf("unexpected toUserOut result: %+v", out)
	}
}

func TestRBACExampleFlow(t *testing.T) {
	api := newRBACExampleAPI(t)

	login := func(email string) LoginOutput {
		data, err := json.Marshal(LoginInput{Email: email, Password: "password123"})
		if err != nil {
			t.Fatalf("marshal login: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("login %s failed: %d %s", email, w.Code, w.Body.String())
		}
		var out LoginOutput
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal login: %v", err)
		}
		return out
	}

	manager := login("manager@example.com")
	auditor := login("auditor@example.com")

	req := httptest.NewRequest(http.MethodGet, "/rbac/me", nil)
	req.Header.Set("Authorization", "Bearer "+manager.Token)
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected /rbac/me 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/users/", bytes.NewReader([]byte(`{"name":"New","email":"new@example.com","password":"password123","age":18}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+manager.Token)
	w = httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected manager create user 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/users/", bytes.NewReader([]byte(`{"name":"Denied","email":"denied@example.com","password":"password123","age":18}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auditor.Token)
	w = httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected auditor create user 403, got %d: %s", w.Code, w.Body.String())
	}
}
