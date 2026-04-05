package orm

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/gorm"
)

type txUser struct {
	ID   uint
	Name string
}

func newTxContext(t *testing.T) (*gin.Context, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db := testDB(t)
	if err := db.AutoMigrate(&txUser{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Set(dbContextKey, db)
	return c, db
}

func TestWithTransactionCommits(t *testing.T) {
	c, db := newTxContext(t)
	name := "commit-user"

	if err := WithTransaction(c, func() error {
		if !InTransaction(c) {
			t.Fatal("expected active transaction")
		}
		return WithContext(c).Create(&txUser{Name: name}).Error
	}); err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	var count int64
	if err := db.Model(&txUser{}).Where("name = ?", name).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after commit, got %d", count)
	}
}

func TestWithTransactionRollsBackOnError(t *testing.T) {
	c, db := newTxContext(t)
	name := "rollback-user"

	err := WithTransaction(c, func() error {
		if err := WithContext(c).Create(&txUser{Name: name}).Error; err != nil {
			return err
		}
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var count int64
	if err := db.Model(&txUser{}).Where("name = ?", name).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to leave 0 rows, got %d", count)
	}
}

func TestNinjaContextTransactionWrappers(t *testing.T) {
	c, db := newTxContext(t)
	ctx := &ninja.Context{Context: c}

	if err := BeginTx(ctx); err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if !InTransaction(c) {
		t.Fatal("expected active transaction")
	}
	if err := WithContext(c).Create(&txUser{Name: "wrapper-user"}).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := CommitTx(ctx); err != nil {
		t.Fatalf("CommitTx: %v", err)
	}

	var count int64
	if err := db.Model(&txUser{}).Where("name = ?", "wrapper-user").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected committed wrapper transaction, got %d", count)
	}
}
