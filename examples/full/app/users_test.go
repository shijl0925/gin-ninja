package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
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
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	orm.Init(db)

	api := ninja.New(ninja.Config{Title: "Test", Version: "0.0.1"})
	authRouter := ninja.NewRouter("/auth", ninja.WithTags("Auth"))
	ninja.Post(authRouter, "/register", Register)
	api.AddRouter(authRouter)
	return api
}

func setupAppTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	orm.Init(db)
	settings.Global.JWT.Secret = "test-secret"
	settings.Global.JWT.ExpireHours = 24
	settings.Global.JWT.Issuer = "gin-ninja"
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

	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out["email"] != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %v", out["email"])
	}
	if out["name"] != "Alice" {
		t.Fatalf("expected name Alice, got %v", out["name"])
	}
	if _, ok := out["id"]; !ok {
		t.Fatalf("expected id to be present, got %+v", out)
	}
	if _, ok := out["password"]; ok {
		t.Fatalf("expected password to be filtered, got %+v", out)
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
	if registered.Model.Email != "alice@example.com" {
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
	if created.Model.ID == 0 {
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
	repo := NewUserRepo()
	if err := repo.UpdateById(int(second.Model.ID), map[string]interface{}{"is_admin": true}); err != nil {
		t.Fatalf("set second user admin: %v", err)
	}

	got, err := GetUser(nil, &GetUserInput{UserID: created.Model.ID})
	if err != nil || got.Model.Email != "alice@example.com" {
		t.Fatalf("GetUser: result=%+v err=%v", got, err)
	}

	page, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Search:    "Ali",
	})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Model.Email != "alice@example.com" {
		t.Fatalf("unexpected list result: %+v", page)
	}

	emailPage, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Search:    "bob@example.com",
	})
	if err != nil {
		t.Fatalf("ListUsers email search: %v", err)
	}
	if emailPage.Total != 1 || len(emailPage.Items) != 1 || emailPage.Items[0].Model.Email != "bob@example.com" {
		t.Fatalf("unexpected email search result: %+v", emailPage)
	}

	adminOnly := true
	adminPage, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		IsAdmin:   &adminOnly,
	})
	if err != nil {
		t.Fatalf("ListUsers admin filter: %v", err)
	}
	if adminPage.Total != 1 || len(adminPage.Items) != 1 || adminPage.Items[0].Model.Email != "bob@example.com" || !adminPage.Items[0].Model.IsAdmin {
		t.Fatalf("unexpected admin list result: %+v", adminPage)
	}

	adminSearchPage, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Search:    "example.com",
		IsAdmin:   &adminOnly,
	})
	if err != nil {
		t.Fatalf("ListUsers admin search filter: %v", err)
	}
	if adminSearchPage.Total != 1 || len(adminSearchPage.Items) != 1 || adminSearchPage.Items[0].Model.Email != "bob@example.com" {
		t.Fatalf("unexpected admin search result: %+v", adminSearchPage)
	}

	sortedPage, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Sort:      "-age",
	})
	if err != nil {
		t.Fatalf("ListUsers sort: %v", err)
	}
	if len(sortedPage.Items) != 2 || sortedPage.Items[0].Model.Age < sortedPage.Items[1].Model.Age {
		t.Fatalf("unexpected sorted list result: %+v", sortedPage)
	}

	if _, err := ListUsers(nil, &ListUsersInput{
		PageInput: pagination.PageInput{Page: 1, Size: 10},
		Sort:      "-password",
	}); err == nil {
		t.Fatal("expected invalid sort to fail")
	}

	updated, err := UpdateUser(nil, &UpdateUserInput{
		UserID: second.Model.ID,
		Name:   "Bobby",
		Email:  "bobby@example.com",
		Age:    21,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Model.Name != "Bobby" || updated.Model.Email != "bobby@example.com" || updated.Model.Age != 21 {
		t.Fatalf("unexpected updated user: %+v", updated)
	}

	if err := DeleteUser(nil, &DeleteUserInput{UserID: second.Model.ID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	deleted, err := GetUser(nil, &GetUserInput{UserID: second.Model.ID})
	if !ninja.IsNotFound(err) || deleted != nil {
		t.Fatalf("expected deleted user to be missing, got result=%+v err=%v", deleted, err)
	}

	out, err := toUserOut(User{Model: gorm.Model{ID: 7}, Email: "x@example.com", Name: "X", Age: 9, IsAdmin: true, Password: "secret"})
	if err != nil {
		t.Fatalf("toUserOut: %v", err)
	}
	if out.Model.Email != "x@example.com" || !out.Model.IsAdmin {
		t.Fatalf("unexpected toUserOut result: %+v", out)
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal UserOut: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal UserOut json: %v", err)
	}
	if _, ok := payload["password"]; ok {
		t.Fatalf("expected password to be filtered, got %+v", payload)
	}
	if payload["id"] != float64(7) {
		t.Fatalf("expected id to remain, got %+v", payload)
	}
}
