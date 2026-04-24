package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
