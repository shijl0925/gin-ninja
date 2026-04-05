package orm

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func TestMiddlewareAndGetDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	Middleware(db)(c)
	if got := GetDB(c); got != db {
		t.Fatalf("expected middleware db, got %v", got)
	}
}

func TestGetDBFallsBackToGlobalAndWithContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	Init(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), "trace_id", "trace-1"))
	c.Request = req

	if got := GetDB(c); got != db {
		t.Fatalf("expected global db fallback, got %v", got)
	}

	withCtx := WithContext(c)
	if withCtx == nil || withCtx.Statement.Context.Value("trace_id") != "trace-1" {
		t.Fatalf("expected request context propagation, got %#v", withCtx)
	}
}

func TestRegisterDefaultErrorMappers(t *testing.T) {
	api := ninja.New(ninja.Config{})
	RegisterDefaultErrorMappers(api)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("gin_ninja_api", api)

	ninja.WriteError(c, gorm.ErrRecordNotFound)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	err := errors.New("plain")
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("gin_ninja_api", api)
	ninja.WriteError(c, err)
	if w.Code != 500 {
		t.Fatalf("expected 500 fallback, got %d: %s", w.Code, w.Body.String())
	}
}
