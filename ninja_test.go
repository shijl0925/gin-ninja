package ninja_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
	"golang.org/x/net/websocket"
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
	return doRequestWithHeaders(api, method, path, body, nil)
}

func doRequestWithHeaders(api *ninja.NinjaAPI, method, path string, body interface{}, configure func(*http.Request)) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if configure != nil {
		configure(req)
	}
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

func TestLifecycleHooksAndShutdown(t *testing.T) {
	api := newTestAPI()
	var startupCount int32
	var shutdownCount int32
	started := make(chan struct{}, 1)

	api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
		atomic.AddInt32(&startupCount, 1)
		select {
		case started <- struct{}{}:
		default:
		}
		return nil
	})
	api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
		atomic.AddInt32(&shutdownCount, 1)
		return nil
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- api.Serve(listener)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for startup hook")
	}
	if atomic.LoadInt32(&startupCount) != 1 {
		t.Fatalf("expected startup hook to run once, got %d", startupCount)
	}

	if err := api.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	if atomic.LoadInt32(&shutdownCount) != 1 {
		t.Fatalf("expected shutdown hook to run once, got %d", shutdownCount)
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

type cookieInput struct {
	Session string `cookie:"session" binding:"required"`
}

type cookieOutput struct {
	Session string `json:"session"`
}

type defaultsInput struct {
	Name    string `form:"name" default:"guest" description:"effective user name"`
	Trace   string `header:"X-Trace" default:"trace-default" description:"trace identifier"`
	Session string `cookie:"session" default:"anon" description:"session key"`
}

type defaultsOutput struct {
	Name    string `json:"name"`
	Trace   string `json:"trace"`
	Session string `json:"session"`
}

func TestGet_CookieParam(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/session")

	ninja.Get(r, "/", func(ctx *ninja.Context, in *cookieInput) (*cookieOutput, error) {
		return &cookieOutput{Session: in.Session}, nil
	})
	api.AddRouter(r)

	w := doRequestWithHeaders(api, http.MethodGet, "/session/", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "session", Value: "sess-123"})
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out cookieOutput
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if out.Session != "sess-123" {
		t.Fatalf("expected cookie session to round-trip, got %+v", out)
	}
}

func TestGet_DefaultQueryHeaderCookieValues(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/defaults")

	ninja.Get(r, "/", func(ctx *ninja.Context, in *defaultsInput) (*defaultsOutput, error) {
		return &defaultsOutput{Name: in.Name, Trace: in.Trace, Session: in.Session}, nil
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/defaults/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out defaultsOutput
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if out.Name != "guest" || out.Trace != "trace-default" || out.Session != "anon" {
		t.Fatalf("expected defaults to apply, got %+v", out)
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

func TestNestedRouters_InheritParentMiddleware(t *testing.T) {
	api := ninja.New(ninja.Config{DisableGinDefault: true})
	parent := ninja.NewRouter("/users")
	child := ninja.NewRouter("/:userID/posts")

	parent.UseGin(func(c *gin.Context) {
		c.Set("raw-parent", true)
		c.Next()
	})
	parent.Use(func(ctx *ninja.Context) error {
		ctx.Set("typed-parent", true)
		return nil
	})

	ninja.Get(child, "/", func(ctx *ninja.Context, _ *struct{}) (*map[string]bool, error) {
		rawParent, _ := ctx.Get("raw-parent")
		typedParent, _ := ctx.Get("typed-parent")
		return &map[string]bool{
			"raw":   rawParent == true,
			"typed": typedParent == true,
		}, nil
	})

	parent.AddRouter(child)
	api.AddRouter(parent)

	w := doRequest(api, http.MethodGet, "/users/7/posts/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if !out["raw"] || !out["typed"] {
		t.Fatalf("expected parent middleware to run for child route, got %+v", out)
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

func TestOpenAPISpec_GenericSchemaRefExists(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/users", ninja.WithTags("Users"))

	type userOut struct {
		ID uint `json:"id"`
	}

	ninja.Get(r, "/", func(ctx *ninja.Context, in *struct{}) (*pagination.Page[userOut], error) {
		return pagination.NewPage([]userOut{{ID: 1}}, 1, pagination.PageInput{}), nil
	})

	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	paths := spec["paths"].(map[string]interface{})
	get := paths["/users/"].(map[string]interface{})["get"].(map[string]interface{})
	responses := get["responses"].(map[string]interface{})
	okResp := responses["200"].(map[string]interface{})
	content := okResp["content"].(map[string]interface{})
	appJSON := content["application/json"].(map[string]interface{})
	schema := appJSON["schema"].(map[string]interface{})
	ref := schema["$ref"].(string)

	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		t.Fatalf("expected ref to start with %q, got %q", prefix, ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	if strings.Contains(name, "/") {
		t.Fatalf("expected sanitized schema name without '/', got %q", name)
	}

	components := spec["components"].(map[string]interface{})
	schemas := components["schemas"].(map[string]interface{})
	if _, ok := schemas[name]; !ok {
		t.Fatalf("expected referenced schema %q to exist in components, got %v", name, schemas)
	}
}

func TestOpenAPISpec_BearerSecurity(t *testing.T) {
	api := ninja.New(ninja.Config{
		Title:   "Test",
		Version: "0.0.1",
		SecuritySchemes: map[string]ninja.SecurityScheme{
			"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
		},
	})
	r := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithBearerAuth())

	ninja.Get(r, "/", func(ctx *ninja.Context, in *struct{}) (*listOutput, error) {
		return &listOutput{}, nil
	})
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	components := spec["components"].(map[string]interface{})
	securitySchemes := components["securitySchemes"].(map[string]interface{})
	bearer := securitySchemes["bearerAuth"].(map[string]interface{})
	if bearer["type"] != "http" {
		t.Fatalf("expected bearerAuth type=http, got %v", bearer["type"])
	}
	if bearer["scheme"] != "bearer" {
		t.Fatalf("expected bearerAuth scheme=bearer, got %v", bearer["scheme"])
	}
	if bearer["bearerFormat"] != "JWT" {
		t.Fatalf("expected bearerAuth bearerFormat=JWT, got %v", bearer["bearerFormat"])
	}

	paths := spec["paths"].(map[string]interface{})
	get := paths["/users/"].(map[string]interface{})["get"].(map[string]interface{})
	security := get["security"].([]interface{})
	if len(security) != 1 {
		t.Fatalf("expected one security requirement, got %v", security)
	}
	req := security[0].(map[string]interface{})
	scopes, ok := req["bearerAuth"]
	if !ok {
		t.Fatalf("expected bearerAuth requirement, got %v", req)
	}
	scopeList, ok := scopes.([]interface{})
	if !ok {
		t.Fatalf("expected bearerAuth scopes to serialize as array, got %T (%v)", scopes, scopes)
	}
	if len(scopeList) != 0 {
		t.Fatalf("expected bearerAuth scopes to be empty, got %v", scopeList)
	}
}

func TestOpenAPISpec_CookieParamAndExtraResponses(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/session", ninja.WithTags("Session"))

	ninja.Get(r, "/", func(ctx *ninja.Context, in *cookieInput) (*cookieOutput, error) {
		return &cookieOutput{Session: in.Session}, nil
	},
		ninja.Response(http.StatusUnauthorized, "Unauthorized", nil),
		ninja.Response(http.StatusNotFound, "Missing session", &cookieOutput{}),
	)
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	paths := spec["paths"].(map[string]interface{})
	get := paths["/session/"].(map[string]interface{})["get"].(map[string]interface{})

	parameters := get["parameters"].([]interface{})
	if len(parameters) != 1 {
		t.Fatalf("expected one parameter, got %v", parameters)
	}
	parameter := parameters[0].(map[string]interface{})
	if parameter["in"] != "cookie" || parameter["name"] != "session" {
		t.Fatalf("expected cookie parameter, got %v", parameter)
	}

	responses := get["responses"].(map[string]interface{})
	if _, ok := responses["401"]; !ok {
		t.Fatalf("expected documented 401 response, got %v", responses)
	}
	notFound := responses["404"].(map[string]interface{})
	content := notFound["content"].(map[string]interface{})
	appJSON := content["application/json"].(map[string]interface{})
	schema := appJSON["schema"].(map[string]interface{})
	if schema["$ref"] == "" {
		t.Fatalf("expected schema ref for documented response, got %v", schema)
	}
}

func TestOpenAPISpec_DefaultsTagDescriptionsAndPaginatedResponse(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter(
		"/defaults",
		ninja.WithTags("Users"),
		ninja.WithTagDescription("Users", "User operations"),
	)

	type itemOut struct {
		ID int `json:"id"`
	}

	ninja.Get(r, "/", func(ctx *ninja.Context, in *defaultsInput) (*pagination.Page[itemOut], error) {
		return pagination.NewPage([]itemOut{{ID: 1}}, 1, pagination.PageInput{}), nil
	}, ninja.Paginated[itemOut]())
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	tags := spec["tags"].([]interface{})
	tag := tags[0].(map[string]interface{})
	if tag["name"] != "Users" || tag["description"] != "User operations" {
		t.Fatalf("unexpected tag metadata: %v", tag)
	}

	get := spec["paths"].(map[string]interface{})["/defaults/"].(map[string]interface{})["get"].(map[string]interface{})
	params := get["parameters"].([]interface{})
	paramByName := map[string]map[string]interface{}{}
	for _, raw := range params {
		param := raw.(map[string]interface{})
		paramByName[param["name"].(string)] = param
	}

	nameSchema := paramByName["name"]["schema"].(map[string]interface{})
	if nameSchema["default"] != "guest" {
		t.Fatalf("expected query default in schema, got %v", nameSchema)
	}
	if paramByName["name"]["description"] != "effective user name" {
		t.Fatalf("expected query description, got %v", paramByName["name"])
	}

	respSchema := get["responses"].(map[string]interface{})["200"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})["schema"].(map[string]interface{})
	if respSchema["type"] != "object" {
		t.Fatalf("expected standardized paginated object schema, got %v", respSchema)
	}
	items := respSchema["properties"].(map[string]interface{})["items"].(map[string]interface{})
	if items["type"] != "array" {
		t.Fatalf("expected paginated items array, got %v", items)
	}
}

func TestOperationTimeoutAndRateLimit(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/ops")

	ninja.Get(r, "/timeout", func(ctx *ninja.Context, _ *struct{}) (*struct{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}, ninja.Timeout(10*time.Millisecond))

	ninja.Get(r, "/limited", func(ctx *ninja.Context, _ *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	}, ninja.RateLimit(1, 1))

	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/ops/timeout", nil)
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d: %s", w.Code, w.Body.String())
	}

	first := doRequest(api, http.MethodGet, "/ops/limited", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first limited request to pass, got %d: %s", first.Code, first.Body.String())
	}
	second := doRequest(api, http.MethodGet, "/ops/limited", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second limited request to be rejected, got %d: %s", second.Code, second.Body.String())
	}

	openapi := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(openapi.Body.Bytes(), &spec) //nolint:errcheck
	paths := spec["paths"].(map[string]interface{})
	timeoutResponses := paths["/ops/timeout"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})
	if _, ok := timeoutResponses["408"]; !ok {
		t.Fatalf("expected 408 response to be documented, got %v", timeoutResponses)
	}
	limitResponses := paths["/ops/limited"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})
	if _, ok := limitResponses["429"]; !ok {
		t.Fatalf("expected 429 response to be documented, got %v", limitResponses)
	}
}

func TestOpenAPISpec_ExcludeFromDocs(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/internal")

	ninja.Get(r, "/health", func(ctx *ninja.Context, in *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	}, ninja.ExcludeFromDocs())
	api.AddRouter(r)

	w := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &spec) //nolint:errcheck

	paths := spec["paths"].(map[string]interface{})
	if _, ok := paths["/internal/health"]; ok {
		t.Fatalf("expected excluded path to be omitted, got %v", paths)
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

type cacheOutput struct {
	Count int `json:"count"`
}

type externalCacheStore struct {
	items map[string]*ninja.CachedResponse
}

func (s *externalCacheStore) Get(key string) (*ninja.CachedResponse, bool) {
	if s.items == nil {
		return nil, false
	}
	value, ok := s.items[key]
	if !ok {
		return nil, false
	}
	cloned := *value
	cloned.Header = value.Header.Clone()
	cloned.Body = append([]byte(nil), value.Body...)
	return &cloned, true
}

func (s *externalCacheStore) Set(key string, value *ninja.CachedResponse) {
	if value == nil {
		return
	}
	if s.items == nil {
		s.items = map[string]*ninja.CachedResponse{}
	}
	cloned := *value
	cloned.Header = value.Header.Clone()
	cloned.Body = append([]byte(nil), value.Body...)
	s.items[key] = &cloned
}

func TestGet_CacheETagAndCacheControl(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/cache")
	calls := 0

	ninja.Get(r, "/", func(ctx *ninja.Context, _ *struct{}) (*cacheOutput, error) {
		calls++
		return &cacheOutput{Count: calls}, nil
	}, ninja.Cache(time.Minute))
	api.AddRouter(r)

	first := doRequest(api, http.MethodGet, "/cache/", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", first.Code, first.Body.String())
	}
	if got := first.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("expected cache-control header, got %q", got)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	second := doRequest(api, http.MethodGet, "/cache/", nil)
	if second.Code != http.StatusOK {
		t.Fatalf("expected cached 200, got %d: %s", second.Code, second.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected cached handler result, calls=%d", calls)
	}
	if second.Body.String() != first.Body.String() {
		t.Fatalf("expected cached body to match, got %q vs %q", second.Body.String(), first.Body.String())
	}

	notModified := doRequestWithHeaders(api, http.MethodGet, "/cache/", nil, func(req *http.Request) {
		req.Header.Set("If-None-Match", etag)
	})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d: %s", notModified.Code, notModified.Body.String())
	}

	openapi := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(openapi.Body.Bytes(), &spec) //nolint:errcheck
	headers := spec["paths"].(map[string]interface{})["/cache/"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})["200"].(map[string]interface{})["headers"].(map[string]interface{})
	if _, ok := headers["ETag"]; !ok {
		t.Fatalf("expected ETag header to be documented, got %v", headers)
	}
	if _, ok := headers["Cache-Control"]; !ok {
		t.Fatalf("expected Cache-Control header to be documented, got %v", headers)
	}
}

func TestGet_CacheWithExternalStore(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/cache-store")
	calls := 0
	store := &externalCacheStore{}

	ninja.Get(r, "/", func(ctx *ninja.Context, _ *struct{}) (*cacheOutput, error) {
		calls++
		return &cacheOutput{Count: calls}, nil
	}, ninja.Cache(time.Minute, ninja.CacheWithStore(store)))
	api.AddRouter(r)

	first := doRequest(api, http.MethodGet, "/cache-store/", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", first.Code, first.Body.String())
	}
	second := doRequest(api, http.MethodGet, "/cache-store/", nil)
	if second.Code != http.StatusOK {
		t.Fatalf("expected cached 200, got %d: %s", second.Code, second.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected external cache store to serve second response, calls=%d", calls)
	}
	if len(store.items) != 1 {
		t.Fatalf("expected one cached item in external store, got %d", len(store.items))
	}
}

type versionOutput struct {
	Version string `json:"version"`
}

func TestVersionedRoutersAndDocs(t *testing.T) {
	api := ninja.New(ninja.Config{
		Title:   "Versioned",
		Version: "main",
		Prefix:  "/api",
		Versions: map[string]ninja.VersionConfig{
			"v1": {
				Prefix:       "/v1",
				Description:  "Legacy API",
				Deprecated:   true,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/migrate",
			},
			"v2": {Prefix: "/v2"},
		},
	})

	v1 := ninja.NewRouter("/users", ninja.WithVersion("v1"))
	ninja.Get(v1, "/", func(ctx *ninja.Context, _ *struct{}) (*versionOutput, error) {
		return &versionOutput{Version: "v1"}, nil
	})
	api.AddRouter(v1)

	v2 := ninja.NewRouter("/users", ninja.WithVersion("v2"))
	ninja.Get(v2, "/", func(ctx *ninja.Context, _ *struct{}) (*versionOutput, error) {
		return &versionOutput{Version: "v2"}, nil
	})
	api.AddRouter(v2)

	v1Resp := doRequest(api, http.MethodGet, "/api/v1/users/", nil)
	if v1Resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v1Resp.Code, v1Resp.Body.String())
	}
	if v1Resp.Header().Get("Deprecation") != "true" {
		t.Fatalf("expected deprecation header, got %v", v1Resp.Header())
	}
	if v1Resp.Header().Get("Sunset") == "" || v1Resp.Header().Get("Link") == "" {
		t.Fatalf("expected sunset and link headers, got %v", v1Resp.Header())
	}

	v2Resp := doRequest(api, http.MethodGet, "/api/v2/users/", nil)
	if v2Resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v2Resp.Code, v2Resp.Body.String())
	}
	if v2Resp.Header().Get("Deprecation") != "" {
		t.Fatalf("did not expect deprecation header on v2, got %v", v2Resp.Header())
	}

	v1Docs := doRequest(api, http.MethodGet, "/openapi/v1.json", nil)
	if v1Docs.Code != http.StatusOK {
		t.Fatalf("expected versioned docs, got %d: %s", v1Docs.Code, v1Docs.Body.String())
	}
	var v1Spec map[string]interface{}
	json.Unmarshal(v1Docs.Body.Bytes(), &v1Spec) //nolint:errcheck
	v1Paths := v1Spec["paths"].(map[string]interface{})
	if _, ok := v1Paths["/api/v1/users/"]; !ok {
		t.Fatalf("expected v1 path in v1 docs, got %v", v1Paths)
	}
	if _, ok := v1Paths["/api/v2/users/"]; ok {
		t.Fatalf("expected v2 path to be isolated from v1 docs, got %v", v1Paths)
	}
	get := v1Paths["/api/v1/users/"].(map[string]interface{})["get"].(map[string]interface{})
	if deprecated, ok := get["deprecated"].(bool); !ok || !deprecated {
		t.Fatalf("expected deprecated version operations to be marked deprecated, got %v", get)
	}

	docsUI := doRequest(api, http.MethodGet, "/docs/v1", nil)
	if docsUI.Code != http.StatusOK || !strings.Contains(docsUI.Body.String(), "/openapi/v1.json") {
		t.Fatalf("expected versioned docs UI to reference versioned spec, got %d %q", docsUI.Code, docsUI.Body.String())
	}
}

type streamInput struct {
	Name string `form:"name"`
}

func TestSSEAndWebSocketHelpers(t *testing.T) {
	api := newTestAPI()
	r := ninja.NewRouter("/stream")

	ninja.SSE(r, "/events", func(ctx *ninja.Context, in *streamInput, stream *ninja.SSEStream) error {
		return stream.Send(ninja.SSEEvent{
			Event: "hello",
			Data:  map[string]string{"name": in.Name},
		})
	})
	ninja.WebSocket(r, "/ws", func(ctx *ninja.Context, in *streamInput, conn *ninja.WebSocketConn) error {
		message, err := conn.ReceiveText()
		if err != nil {
			return err
		}
		return conn.SendText(in.Name + ":" + message)
	})
	api.AddRouter(r)

	sse := doRequest(api, http.MethodGet, "/stream/events?name=bot", nil)
	if sse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", sse.Code, sse.Body.String())
	}
	if ct := sse.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", ct)
	}
	if body := sse.Body.String(); !strings.Contains(body, "event: hello") || !strings.Contains(body, `data: {"name":"bot"}`) {
		t.Fatalf("unexpected SSE body %q", body)
	}

	server := httptest.NewServer(api.Handler())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/stream/ws?name=bot"
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := websocket.Message.Send(conn, "ping"); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}
	var reply string
	if err := websocket.Message.Receive(conn, &reply); err != nil {
		t.Fatalf("receive websocket reply: %v", err)
	}
	if reply != "bot:ping" {
		t.Fatalf("unexpected websocket reply %q", reply)
	}

	openapi := doRequest(api, http.MethodGet, "/openapi.json", nil)
	var spec map[string]interface{}
	json.Unmarshal(openapi.Body.Bytes(), &spec) //nolint:errcheck
	paths := spec["paths"].(map[string]interface{})
	sseResponses := paths["/stream/events"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})
	if _, ok := sseResponses["200"]; !ok {
		t.Fatalf("expected SSE response documentation, got %v", sseResponses)
	}
	wsResponses := paths["/stream/ws"].(map[string]interface{})["get"].(map[string]interface{})["responses"].(map[string]interface{})
	if _, ok := wsResponses["101"]; !ok {
		t.Fatalf("expected websocket upgrade response documentation, got %v", wsResponses)
	}
}

func TestWebSocketHandlerErrorDoesNotLeakToClient(t *testing.T) {
	api := ninja.New(ninja.Config{DisableGinDefault: true})
	r := ninja.NewRouter("/stream")
	expectedErr := errors.New("handler boom")
	var loggedErr string

	api.UseGin(func(c *gin.Context) {
		c.Next()
		if errs := c.Errors.ByType(gin.ErrorTypePrivate); len(errs) > 0 {
			loggedErr = errs.String()
		}
	})

	ninja.WebSocket(r, "/ws", func(ctx *ninja.Context, in *streamInput, conn *ninja.WebSocketConn) error {
		if _, err := conn.ReceiveText(); err != nil {
			return err
		}
		return expectedErr
	})
	api.AddRouter(r)

	server := httptest.NewServer(api.Handler())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/stream/ws?name=bot"
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := websocket.Message.Send(conn, "ping"); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}
	var reply string
	if err := websocket.Message.Receive(conn, &reply); err == nil {
		t.Fatalf("expected websocket to close without sending an error payload, got %q", reply)
	}
	if !strings.Contains(loggedErr, expectedErr.Error()) {
		t.Fatalf("expected websocket handler error to be recorded privately, got %q", loggedErr)
	}
}
