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

type ormContextKey string

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
	if got, ok := RequestDB(c); !ok || got != db {
		t.Fatalf("expected request db, got %v ok=%v", got, ok)
	}
	if got, ok := RequestWithContext(c); !ok || got == nil {
		t.Fatalf("expected request db with context, got %v ok=%v", got, ok)
	}
}

func TestRequestDBDoesNotFallBackToGlobal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	if got, ok := RequestDB(c); ok || got != nil {
		t.Fatalf("expected no request db, got %v ok=%v", got, ok)
	}
	if got := GetDB(c); got != nil {
		t.Fatalf("expected no db, got %v", got)
	}
}

func TestGetDBUsesRequestDBAndWithContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ormContextKey("trace_id"), "trace-1"))
	c.Request = req
	Middleware(db)(c)

	if got := GetDB(c); got != db {
		t.Fatalf("expected request db, got %v", got)
	}

	withCtx := WithContext(c)
	if withCtx == nil || withCtx.Statement.Context.Value(ormContextKey("trace_id")) != "trace-1" {
		t.Fatalf("expected request context propagation, got %#v", withCtx)
	}

	if _, ok := RequestWithContext(c); !ok {
		t.Fatal("expected RequestWithContext to use request db")
	}
}

func TestBaseDBHelpersAndInit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseDB := testDB(t)
	txDB := testDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	Middleware(baseDB)(c)
	c.Set(txContextKey, txDB)

	if got, ok := RequestBaseDB(c); !ok || got != baseDB {
		t.Fatalf("expected base db, got %v ok=%v", got, ok)
	}
	if got := GetBaseDB(c); got != baseDB {
		t.Fatalf("expected GetBaseDB to return base db, got %v", got)
	}
	if got, ok := RequestDB(c); !ok || got != txDB {
		t.Fatalf("expected request db to prefer tx db, got %v ok=%v", got, ok)
	}

	Init(baseDB)
}

func TestDBHelpersHandleNilAndWrongTypes(t *testing.T) {
	if got, ok := RequestBaseDB(nil); ok || got != nil {
		t.Fatalf("expected nil base db, got %v ok=%v", got, ok)
	}
	if got := GetBaseDB(nil); got != nil {
		t.Fatalf("expected nil base db from GetBaseDB, got %v", got)
	}
	if got := WithContext(nil); got != nil {
		t.Fatalf("expected nil db from WithContext(nil), got %v", got)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(dbContextKey, "wrong")
	c.Set(txContextKey, "wrong")

	if got, ok := RequestBaseDB(c); ok || got != nil {
		t.Fatalf("expected wrong-typed base db to be ignored, got %v ok=%v", got, ok)
	}
	if got, ok := RequestDB(c); ok || got != nil {
		t.Fatalf("expected wrong-typed request db to be ignored, got %v ok=%v", got, ok)
	}
	if got := GetBaseDB(c); got != nil {
		t.Fatalf("expected nil base db for wrong type, got %v", got)
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
