package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
)

func newDemoAPI() *ninja.NinjaAPI {
	gin.SetMode(gin.TestMode)

	api := ninja.New(ninja.Config{Title: "Demo", Version: "0.0.1"})
	router := ninja.NewRouter(
		"/examples",
		ninja.WithTags("Examples"),
		ninja.WithTagDescription("Examples", "Framework feature demos for manual testing"),
	)
	ninja.Get(router, "/request-meta", EchoRequestMeta,
		ninja.Response(http.StatusUnauthorized, "Example unauthorized response", nil),
		ninja.Response(http.StatusNotFound, "Example detailed response", &RequestMetaOutput{}),
	)
	ninja.Get(router, "/features", ListFeatureDemos, ninja.Paginated[FeatureItemOut]())
	ninja.Get(router, "/limited", LimitedOperation, ninja.RateLimit(1, 1))
	ninja.Get(router, "/slow", SlowOperation, ninja.Timeout(150*time.Millisecond))
	ninja.Get(router, "/hidden", HiddenOperation, ninja.ExcludeFromDocs())
	api.AddRouter(router)
	return api
}

func doDemoRequest(api *ninja.NinjaAPI, method, path string, configure func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if configure != nil {
		configure(req)
	}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestDemoEndpoints_RequestMetaDefaultsAndOverrides(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/examples/request-meta", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out RequestMetaOutput
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Session != "guest-session" || out.TraceID != "trace-demo" || out.Lang != "zh-CN" || out.Verbose {
		t.Fatalf("unexpected defaults: %+v", out)
	}

	w = doDemoRequest(api, http.MethodGet, "/examples/request-meta?lang=en-US&verbose=true", func(req *http.Request) {
		req.Header.Set("X-Trace-ID", "trace-override")
		req.AddCookie(&http.Cookie{Name: "session", Value: "sess-123"})
	})
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal override response: %v", err)
	}
	if out.Session != "sess-123" || out.TraceID != "trace-override" || out.Lang != "en-US" || !out.Verbose {
		t.Fatalf("unexpected overridden values: %+v", out)
	}
}

func TestDemoEndpoints_PaginatedRateLimitedAndTimeout(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/examples/features?search=timeout", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal paginated response: %v", err)
	}
	if _, ok := page["items"]; !ok {
		t.Fatalf("expected standardized page items, got %v", page)
	}

	first := doDemoRequest(api, http.MethodGet, "/examples/limited", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first limited request to pass, got %d: %s", first.Code, first.Body.String())
	}
	second := doDemoRequest(api, http.MethodGet, "/examples/limited", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second limited request to be rejected, got %d: %s", second.Code, second.Body.String())
	}

	slow := doDemoRequest(api, http.MethodGet, "/examples/slow", nil)
	if slow.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d: %s", slow.Code, slow.Body.String())
	}
}

func TestDemoEndpoints_OpenAPIVisibilityAndResponses(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/openapi.json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}

	tags := spec["tags"].([]interface{})
	tag := tags[0].(map[string]interface{})
	if tag["name"] != "Examples" || tag["description"] != "Framework feature demos for manual testing" {
		t.Fatalf("unexpected tag metadata: %v", tag)
	}

	paths := spec["paths"].(map[string]interface{})
	if _, ok := paths["/examples/hidden"]; ok {
		t.Fatalf("expected hidden route to be absent from OpenAPI, got %v", paths)
	}

	requestMeta := paths["/examples/request-meta"].(map[string]interface{})["get"].(map[string]interface{})
	responses := requestMeta["responses"].(map[string]interface{})
	if _, ok := responses["401"]; !ok {
		t.Fatalf("expected documented 401 response, got %v", responses)
	}
	if _, ok := responses["404"]; !ok {
		t.Fatalf("expected documented 404 response, got %v", responses)
	}
}
