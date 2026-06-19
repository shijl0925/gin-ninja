package main

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	ninjatest "github.com/shijl0925/gin-ninja/testing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func TestControllerExample_CRUD(t *testing.T) {
	client := ninjatest.NewWithT(t, buildAPI())

	// List — empty initially.
	w := client.Get("/api/v1/books/")
	if w.StatusCode != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.StatusCode, w.String())
	}
	var listResp map[string]interface{}
	if err := w.DecodeJSON(&listResp); err != nil {
		t.Fatalf("list: parse response: %v", err)
	}

	// Create.
	w2 := client.Post("/api/v1/books/", CreateBookInput{Title: "Go Programming", Author: "Donovan"})
	if w2.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w2.StatusCode, w2.String())
	}
	var created BookOut
	if err := w2.DecodeJSON(&created); err != nil {
		t.Fatalf("create: parse response: %v", err)
	}
	if created.Title != "Go Programming" {
		t.Errorf("create: unexpected title: %q", created.Title)
	}
	if created.ID == 0 {
		t.Error("create: expected non-zero ID")
	}

	// Get.
	w3 := client.Get("/api/v1/books/1")
	if w3.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w3.StatusCode, w3.String())
	}

	// Get non-existent.
	w4 := client.Get("/api/v1/books/999")
	if w4.StatusCode != http.StatusNotFound {
		t.Fatalf("get missing: expected 404, got %d", w4.StatusCode)
	}

	// Delete.
	w5 := client.Delete("/api/v1/books/1")
	if w5.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", w5.StatusCode, w5.String())
	}
}

func TestControllerExample_UpdateListPaginationAndMain(t *testing.T) {
	client := ninjatest.NewWithT(t, buildAPI())

	for _, body := range []CreateBookInput{
		{Title: "First", Author: "Ada"},
		{Title: "Second", Author: "Grace"},
	} {
		w := client.Post("/api/v1/books/", body)
		if w.StatusCode != http.StatusCreated {
			t.Fatalf("create status = %d body=%s", w.StatusCode, w.String())
		}
	}

	w := client.Put("/api/v1/books/1", map[string]string{"title": "First Revised"})
	if w.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d body=%s", w.StatusCode, w.String())
	}
	var updated BookOut
	if err := w.DecodeJSON(&updated); err != nil {
		t.Fatalf("parse update: %v", err)
	}
	if updated.Title != "First Revised" || updated.Author != "Ada" {
		t.Fatalf("unexpected updated book: %+v", updated)
	}

	w = client.Put("/api/v1/books/999", map[string]string{"author": "Nobody"})
	if w.StatusCode != http.StatusNotFound {
		t.Fatalf("update missing status = %d body=%s", w.StatusCode, w.String())
	}

	w = client.Get("/api/v1/books/?page=3&size=1")
	if w.StatusCode != http.StatusOK {
		t.Fatalf("list page past end status = %d body=%s", w.StatusCode, w.String())
	}

	w = client.Get("/health")
	if w.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d body=%s", w.StatusCode, w.String())
	}

	origRun := runControllerMain
	origFatal := fatalController
	defer func() {
		runControllerMain = origRun
		fatalController = origFatal
	}()
	called := false
	runControllerMain = func(addr string) error {
		if addr != ":8080" {
			t.Fatalf("run addr = %q", addr)
		}
		called = true
		return http.ErrServerClosed
	}
	fatalController = func(v ...any) { t.Fatalf("fatal should not be called: %v", v) }
	main()
	if !called {
		t.Fatal("expected main to call runControllerMain")
	}
	runControllerMain = func(addr string) error { return errors.New("boom") }
	fatalCalled := false
	fatalController = func(v ...any) { fatalCalled = true }
	main()
	if !fatalCalled {
		t.Fatal("expected main to call fatal on non-shutdown error")
	}
}

func TestBookControllerDirectDatabaseErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	c := &BookController{db: db}
	if _, err := c.List(nil, &ListBooksInput{}); err == nil {
		t.Fatal("expected list error without migrated table")
	}
	if _, err := c.Create(nil, &CreateBookInput{Title: "T", Author: "A"}); err == nil {
		t.Fatal("expected create error without migrated table")
	}
	if _, err := c.Get(nil, &GetBookInput{BookID: 1}); err == nil {
		t.Fatal("expected get error without migrated table")
	}
	if _, err := c.Update(nil, &UpdateBookInput{BookID: 1, Title: "T"}); err == nil {
		t.Fatal("expected update error without migrated table")
	}
	if err := c.Delete(nil, &DeleteBookInput{BookID: 1}); err == nil {
		t.Fatal("expected delete error without migrated table")
	}
}
