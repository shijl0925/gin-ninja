package ninja_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestAPI() *ninja.NinjaAPI {
	return ninja.New(ninja.Config{Title: "Test", Version: "0.0.1"})
}

func doRequest(api *ninja.NinjaAPI, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// NinjaAPI construction
// ---------------------------------------------------------------------------

func TestNew_DefaultConfig(t *testing.T) {
	api := ninja.New(ninja.Config{})
	if api == nil {
		t.Fatal("expected non-nil NinjaAPI")
	}
}

func TestNew_DocsRouteExists(t *testing.T) {
	api := newTestAPI()
	w := doRequest(api, http.MethodGet, "/docs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML content-type, got %s", ct)
	}
}

func TestNew_OpenAPIRouteExists(t *testing.T) {
	api := newTestAPI()
	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var spec map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("failed to parse openapi JSON: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3 got %v", spec["openapi"])
	}
}

// ---------------------------------------------------------------------------
// GET with query parameters
// ---------------------------------------------------------------------------

type listInput struct {
	Name string `form:"name"`
	Page int    `form:"page"`
}

type listOutput struct {
	Items []string `json:"items"`
	Page  int      `json:"page"`
}

func TestGet_QueryParams(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/items", ninja.WithTags("items"))

	ninja.Get(r, "/", func(ctx *ninja.Context, in *listInput) (*listOutput, error) {
		return &listOutput{Items: []string{in.Name}, Page: in.Page}, nil
	})

	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/items/?name=hello&page=3", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out listOutput
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0] != "hello" {
		t.Errorf("unexpected items: %v", out.Items)
	}
	if out.Page != 3 {
		t.Errorf("expected page=3, got %d", out.Page)
	}
}

// ---------------------------------------------------------------------------
// GET with path parameter
// ---------------------------------------------------------------------------

type getInput struct {
	ID int `path:"id"`
}

type getOutput struct {
	ID int `json:"id"`
}

func TestGet_PathParam(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/things")

	ninja.Get(r, "/:id", func(ctx *ninja.Context, in *getInput) (*getOutput, error) {
		return &getOutput{ID: in.ID}, nil
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/things/42", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out getOutput
	json.Unmarshal(w.Body.Bytes(), &out) //nolint:errcheck
	if out.ID != 42 {
		t.Errorf("expected ID=42, got %d", out.ID)
	}
}

// ---------------------------------------------------------------------------
// POST with JSON body
// ---------------------------------------------------------------------------

type createInput struct {
	Name  string `json:"name"  binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type createOutput struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestPost_JSONBody(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/users")

	ninja.Post(r, "/", func(ctx *ninja.Context, in *createInput) (*createOutput, error) {
		return &createOutput{ID: 1, Name: in.Name, Email: in.Email}, nil
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodPost, "/users/", map[string]string{
		"name": "Alice", "email": "alice@example.com",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var out createOutput
	json.Unmarshal(w.Body.Bytes(), &out) //nolint:errcheck
	if out.Name != "Alice" {
		t.Errorf("expected Name=Alice, got %s", out.Name)
	}
}

func TestPost_ValidationError(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/users")

	ninja.Post(r, "/", func(ctx *ninja.Context, in *createInput) (*createOutput, error) {
		return &createOutput{ID: 1, Name: in.Name, Email: in.Email}, nil
	})
	api.AddRouter(r)

	// Missing required fields.
	w := doRequest(api, http.MethodPost, "/users/", map[string]string{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DELETE (void handler)
// ---------------------------------------------------------------------------

type deleteInput struct {
	ID int `path:"id"`
}

func TestDelete_NoContent(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/items")

	ninja.Delete(r, "/:id", func(ctx *ninja.Context, in *deleteInput) error {
		return nil
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodDelete, "/items/5", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

func TestErrorResponse_NotFound(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/items")

	ninja.Get(r, "/:id", func(ctx *ninja.Context, in *getInput) (*getOutput, error) {
		return nil, ninja.ErrNotFound
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/items/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Nested routers
// ---------------------------------------------------------------------------

func TestNestedRouters(t *testing.T) {
	api := ninja.New(ninja.Config{Prefix: "/api/v1"})
	parent := ninja.NewRouter("/users")
	child := ninja.NewRouter("/:userID/posts")

	type childIn struct {
		UserID int `path:"userID"`
	}
	type childOut struct {
		UserID int `json:"userId"`
	}

	ninja.Get(child, "/", func(ctx *ninja.Context, in *childIn) (*childOut, error) {
		return &childOut{UserID: in.UserID}, nil
	})

	parent.AddRouter(child)
	api.AddRouter(parent)

	w := doRequest(api, http.MethodGet, "/api/v1/users/7/posts/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out childOut
	json.Unmarshal(w.Body.Bytes(), &out) //nolint:errcheck
	if out.UserID != 7 {
		t.Errorf("expected UserID=7, got %d", out.UserID)
	}
}

// ---------------------------------------------------------------------------
// OpenAPI spec content
// ---------------------------------------------------------------------------

func TestOpenAPISpec_ContainsPaths(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/users", ninja.WithTags("Users"))

	ninja.Get(r, "/", func(ctx *ninja.Context, in *listInput) (*listOutput, error) {
		return nil, nil
	}, ninja.Summary("List users"))

	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("expected paths object in spec")
	}
	if _, ok := paths["/users/"]; !ok {
		t.Errorf("expected /users/ in paths, got: %v", paths)
	}
}

func TestOpenAPISpec_PrefixAppliedOnce(t *testing.T) {
	api := ninja.New(ninja.Config{Prefix: "/api/v1"})
	r := ninja.NewRouter("/users", ninja.WithTags("Users"))

	ninja.Get(r, "/", func(ctx *ninja.Context, in *listInput) (*listOutput, error) {
		return nil, nil
	}, ninja.Summary("List users"))

	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("expected paths object in spec")
	}
	if _, ok := paths["/api/v1/users/"]; !ok {
		t.Fatalf("expected /api/v1/users/ in paths, got: %v", paths)
	}
	if _, ok := paths["/api/v1/api/v1/users/"]; ok {
		t.Fatalf("expected duplicated prefix path to be absent, got: %v", paths)
	}
}

// ---------------------------------------------------------------------------
// UseGin middleware
// ---------------------------------------------------------------------------

func TestUseGin_MiddlewareRuns(t *testing.T) {
	api := ninja.New(ninja.Config{DisableGinDefault: true})
	called := false
	api.UseGin(func(c *gin.Context) {
		called = true
		c.Next()
	})

	r := ninja.NewRouter("/test")
	ninja.Get(r, "/", func(ctx *ninja.Context, _ *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})
	api.AddRouter(r)

	doRequest(api, http.MethodGet, "/test/", nil)
	if !called {
		t.Error("expected UseGin middleware to be called")
	}
}

func TestRouter_UseGin_MiddlewareRuns(t *testing.T) {
	api := ninja.New(ninja.Config{DisableGinDefault: true})
	called := false

	r := ninja.NewRouter("/test")
	r.UseGin(func(c *gin.Context) {
		called = true
		c.Next()
	})
	ninja.Get(r, "/", func(ctx *ninja.Context, _ *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})
	api.AddRouter(r)

	doRequest(api, http.MethodGet, "/test/", nil)
	if !called {
		t.Error("expected router UseGin middleware to be called")
	}
}

func TestDisableGinDefault(t *testing.T) {
	// Just verify that DisableGinDefault: true doesn't panic and the API works.
	api := ninja.New(ninja.Config{
		Title:             "No Default",
		DisableGinDefault: true,
	})
	w := doRequest(api, http.MethodGet, "/docs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
