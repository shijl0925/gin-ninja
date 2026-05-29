package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func TestControllerExample_CRUD(t *testing.T) {
	api := buildAPI()

	// List — empty initially.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/", nil)
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("list: parse response: %v", err)
	}

	// Create.
	createBody := `{"title":"Go Programming","author":"Donovan"}`
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/books/", strings.NewReader(createBody))
	req2.Header.Set("Content-Type", "application/json")
	api.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	var created BookOut
	if err := json.Unmarshal(w2.Body.Bytes(), &created); err != nil {
		t.Fatalf("create: parse response: %v", err)
	}
	if created.Title != "Go Programming" {
		t.Errorf("create: unexpected title: %q", created.Title)
	}
	if created.ID == 0 {
		t.Error("create: expected non-zero ID")
	}

	// Get.
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/books/1", nil)
	api.Handler().ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	// Get non-existent.
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/books/999", nil)
	api.Handler().ServeHTTP(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Fatalf("get missing: expected 404, got %d", w4.Code)
	}

	// Delete.
	w5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodDelete, "/api/v1/books/1", nil)
	api.Handler().ServeHTTP(w5, req5)
	if w5.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", w5.Code, w5.Body.String())
	}
}

func TestControllerExample_UpdateListPaginationAndMain(t *testing.T) {
	api := buildAPI()

	for _, body := range []string{
		`{"title":"First","author":"Ada"}`,
		`{"title":"Second","author":"Grace"}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/books/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		api.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create status = %d body=%s", w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/books/1", strings.NewReader(`{"title":"First Revised"}`))
	req.Header.Set("Content-Type", "application/json")
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", w.Code, w.Body.String())
	}
	var updated BookOut
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("parse update: %v", err)
	}
	if updated.Title != "First Revised" || updated.Author != "Ada" {
		t.Fatalf("unexpected updated book: %+v", updated)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/books/999", strings.NewReader(`{"author":"Nobody"}`))
	req.Header.Set("Content-Type", "application/json")
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("update missing status = %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/books/?page=3&size=1", nil)
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list page past end status = %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	api.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d body=%s", w.Code, w.Body.String())
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
