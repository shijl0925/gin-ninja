package orm

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestRequestDatabaseNilAndWrongTypeBranches(t *testing.T) {
	if db, ok := RequestBaseDB(nil); ok || db != nil {
		t.Fatalf("RequestBaseDB(nil) = (%v, %v)", db, ok)
	}
	if db, ok := RequestDB(nil); ok || db != nil {
		t.Fatalf("RequestDB(nil) = (%v, %v)", db, ok)
	}
	if db := WithContext(nil); db != nil {
		t.Fatalf("WithContext(nil) = %v", db)
	}
	if db, ok := RequestWithContext(nil); ok || db != nil {
		t.Fatalf("RequestWithContext(nil) = (%v, %v)", db, ok)
	}
	if db := GetBaseDB(nil); db != nil {
		t.Fatalf("GetBaseDB(nil) = %v", db)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(dbContextKey, "wrong")
	c.Set(txContextKey, "wrong")
	if db, ok := RequestBaseDB(c); ok || db != nil {
		t.Fatalf("RequestBaseDB(wrong type) = (%v, %v)", db, ok)
	}
	if db, ok := RequestDB(c); ok || db != nil {
		t.Fatalf("RequestDB(wrong type) = (%v, %v)", db, ok)
	}

	db := testDB(t)
	Init(db)
	Middleware(db)(c)
	c.Request = nil
	if got := WithContext(c); got != db {
		t.Fatalf("WithContext without request = %v, want original db", got)
	}
	if got, ok := RequestWithContext(c); !ok || got != db {
		t.Fatalf("RequestWithContext without request = (%v, %v)", got, ok)
	}
	if got := GetBaseDB(c); got != db {
		t.Fatalf("GetBaseDB = %v, want %v", got, db)
	}

	c.Set(txContextKey, (*gorm.DB)(nil))
	if got, ok := RequestDB(c); !ok || got != db {
		t.Fatalf("RequestDB nil tx should fall back to base db, got (%v, %v)", got, ok)
	}
}
