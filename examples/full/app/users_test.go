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
