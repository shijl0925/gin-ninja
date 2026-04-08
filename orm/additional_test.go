package orm

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/gorm"
)

func TestNinjaTransactionHelpersValidateNilContext(t *testing.T) {
	t.Parallel()

	if err := BeginTx(nil); err == nil {
		t.Fatal("expected BeginTx(nil) error")
	}
	if err := CommitTx(nil); err == nil {
		t.Fatal("expected CommitTx(nil) error")
	}
	if err := RollbackTx(nil); err == nil {
		t.Fatal("expected RollbackTx(nil) error")
	}
}

func TestRegisterDefaultErrorMappersAdditional(t *testing.T) {
	t.Run("maps gorm errors through API responses", func(t *testing.T) {
		api := ninja.New(ninja.Config{Title: "test", Version: "1"})
		RegisterDefaultErrorMappers(api)

		router := ninja.NewRouter("/orm")
		ninja.Get(router, "/missing", func(ctx *ninja.Context, in *struct{}) (*struct{}, error) {
			return nil, gorm.ErrRecordNotFound
		})
		ninja.Get(router, "/duplicate", func(ctx *ninja.Context, in *struct{}) (*struct{}, error) {
			return nil, gorm.ErrDuplicatedKey
		})
		api.AddRouter(router)

		for path, want := range map[string]int{
			"/orm/missing":   http.StatusNotFound,
			"/orm/duplicate": http.StatusConflict,
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			api.Handler().ServeHTTP(w, req)
			if w.Code != want {
				t.Fatalf("%s: expected %d, got %d", path, want, w.Code)
			}
		}
	})

	t.Run("nil api panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected nil api panic")
			}
		}()
		RegisterDefaultErrorMappers(nil)
	})
}

func TestTransactionEdgeCases(t *testing.T) {
	c, db := newTxContext(t)

	if err := Commit(c); err != nil {
		t.Fatalf("Commit without tx: %v", err)
	}
	if err := Rollback(c); err != nil {
		t.Fatalf("Rollback without tx: %v", err)
	}

	tx, err := Begin(c)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	txAgain, err := Begin(c)
	if err != nil {
		t.Fatalf("Begin existing: %v", err)
	}
	if txAgain == nil || !InTransaction(c) {
		t.Fatal("expected existing transaction to stay active")
	}
	c.Set(txContextKey, "wrong")
	if _, ok := currentTx(c); ok {
		t.Fatal("expected wrong tx type to be ignored")
	}
	c.Set(txContextKey, tx)

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from WithTransaction callback")
		}
		var count int64
		if err := db.Model(&txUser{}).Where("name = ?", "panic-user").Count(&count).Error; err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected rollback after panic, got %d rows", count)
		}
	}()

	if err := Rollback(c); err != nil {
		t.Fatalf("Rollback existing tx: %v", err)
	}
	if err := WithTransaction(c, func() error { return nil }); err != nil {
		t.Fatalf("WithTransaction(nil error): %v", err)
	}
	if err := WithTransaction(c, func() error {
		if err := WithContext(c).Create(&txUser{Name: "panic-user"}).Error; err != nil {
			return err
		}
		panic(errors.New("boom"))
	}); err != nil {
		t.Fatalf("expected panic, got error %v", err)
	}
}
