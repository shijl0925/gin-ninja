package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"golang.org/x/net/websocket"
)

func newDemoAPI() *ninja.NinjaAPI {
	gin.SetMode(gin.TestMode)

	api := ninja.New(ninja.Config{
		Title:   "Demo",
		Version: "main",
		Prefix:  "/api",
		Versions: map[string]ninja.VersionConfig{
			"v1": {
				Prefix:      "/v1",
				Description: "Current example API",
			},
			"v0": {
				Prefix:       "/v0",
				Description:  "Legacy example API",
				Deprecated:   true,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/docs/gin-ninja/v1-migration",
			},
		},
	})
	router := ninja.NewRouter(
		"/examples",
		ninja.WithTags("Examples"),
		ninja.WithTagDescription("Examples", "Framework feature demos for manual testing"),
		ninja.WithVersion("v1"),
	)
	ninja.Get(router, "/request-meta", EchoRequestMeta,
		ninja.Response(http.StatusUnauthorized, "Example unauthorized response", nil),
		ninja.Response(http.StatusNotFound, "Example detailed response", &RequestMetaOutput{}),
	)
	ninja.Get(router, "/features", ListFeatureDemos, ninja.Paginated[FeatureItemOut]())
	ninja.Get(router, "/cache", CachedFeatureDemo, ninja.Cache(time.Minute))
	ninja.Get(router, "/limited", LimitedOperation, ninja.RateLimit(1, 1))
	ninja.Get(router, "/slow", SlowOperation, ninja.Timeout(150*time.Millisecond))
	ninja.Get(router, "/hidden", HiddenOperation, ninja.ExcludeFromDocs())
	ninja.SSE(router, "/events", StreamEventsDemo)
	ninja.WebSocket(router, "/ws", WebSocketEchoDemo)
	ninja.Post(router, "/upload-single", UploadSingleDemo)
	ninja.Post(router, "/upload-many", UploadManyDemo)
	ninja.Get(router, "/download", DownloadDemo)
	ninja.Get(router, "/download-reader", DownloadReaderDemo)
	api.AddRouter(router)

	versionedV1 := ninja.NewRouter("/examples/versioned", ninja.WithTags("Examples"), ninja.WithVersion("v1"))
	ninja.Get(versionedV1, "/info", VersionedInfoV1)
	api.AddRouter(versionedV1)

	versionedV0 := ninja.NewRouter("/examples/versioned", ninja.WithTags("Examples"), ninja.WithVersion("v0"))
	ninja.Get(versionedV0, "/info", VersionedInfoV0)
	api.AddRouter(versionedV0)

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

func doMultipartDemoRequest(t *testing.T, api *ninja.NinjaAPI, path string, fields map[string]string, files map[string][]string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s): %v", key, err)
		}
	}
	for field, names := range files {
		for _, name := range names {
			part, err := writer.CreateFormFile(field, name)
			if err != nil {
				t.Fatalf("CreateFormFile(%s): %v", field, err)
			}
			if _, err := part.Write([]byte("content:" + name)); err != nil {
				t.Fatalf("Write file %s: %v", name, err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestDemoEndpoints_RequestMetaDefaultsAndOverrides(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/api/v1/examples/request-meta", nil)
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

	w = doDemoRequest(api, http.MethodGet, "/api/v1/examples/request-meta?lang=en-US&verbose=true", func(req *http.Request) {
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

func TestDemoEndpoints_HiddenRouteRemainsReachable(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/api/v1/examples/hidden", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "hidden route is reachable") {
		t.Fatalf("unexpected hidden route response: %s", w.Body.String())
	}
}

func TestDemoEndpoints_PaginatedCacheRateLimitedAndTimeout(t *testing.T) {
	api := newDemoAPI()

	w := doDemoRequest(api, http.MethodGet, "/api/v1/examples/features?search=timeout", nil)
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

	cacheDemoCounter.Store(0)
	cacheFirst := doDemoRequest(api, http.MethodGet, "/api/v1/examples/cache", nil)
	if cacheFirst.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", cacheFirst.Code, cacheFirst.Body.String())
	}
	if got := cacheFirst.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("expected cache-control header, got %q", got)
	}
	etag := cacheFirst.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	cacheSecond := doDemoRequest(api, http.MethodGet, "/api/v1/examples/cache", nil)
	if cacheSecond.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", cacheSecond.Code, cacheSecond.Body.String())
	}
	if cacheDemoCounter.Load() != 1 {
		t.Fatalf("expected cached response, counter=%d", cacheDemoCounter.Load())
	}
	if cacheSecond.Body.String() != cacheFirst.Body.String() {
		t.Fatalf("expected cached body to match, got %q vs %q", cacheSecond.Body.String(), cacheFirst.Body.String())
	}

	notModified := doDemoRequest(api, http.MethodGet, "/api/v1/examples/cache", func(req *http.Request) {
		req.Header.Set("If-None-Match", etag)
	})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d: %s", notModified.Code, notModified.Body.String())
	}

	first := doDemoRequest(api, http.MethodGet, "/api/v1/examples/limited", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first limited request to pass, got %d: %s", first.Code, first.Body.String())
	}
	second := doDemoRequest(api, http.MethodGet, "/api/v1/examples/limited", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second limited request to be rejected, got %d: %s", second.Code, second.Body.String())
	}

	slow := doDemoRequest(api, http.MethodGet, "/api/v1/examples/slow", nil)
	if slow.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d: %s", slow.Code, slow.Body.String())
	}
}

func TestDemoEndpoints_FileUploadAndDownload(t *testing.T) {
	api := newDemoAPI()

	single := doMultipartDemoRequest(t, api, "/api/v1/examples/upload-single", map[string]string{
		"title": "avatar",
	}, map[string][]string{
		"file": {"avatar.png"},
	})
	if single.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", single.Code, single.Body.String())
	}

	var singleOut UploadDemoOutput
	if err := json.Unmarshal(single.Body.Bytes(), &singleOut); err != nil {
		t.Fatalf("unmarshal single upload: %v", err)
	}
	if singleOut.Title != "avatar" || singleOut.Filename != "avatar.png" || singleOut.FileCount != 1 {
		t.Fatalf("unexpected single upload response: %+v", singleOut)
	}

	multi := doMultipartDemoRequest(t, api, "/api/v1/examples/upload-many", map[string]string{
		"category": "docs",
	}, map[string][]string{
		"files": {"a.txt", "b.txt"},
	})
	if multi.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", multi.Code, multi.Body.String())
	}

	var multiOut UploadDemoOutput
	if err := json.Unmarshal(multi.Body.Bytes(), &multiOut); err != nil {
		t.Fatalf("unmarshal multi upload: %v", err)
	}
	if multiOut.Category != "docs" || multiOut.FileCount != 2 {
		t.Fatalf("unexpected multi upload response: %+v", multiOut)
	}

	download := doDemoRequest(api, http.MethodGet, "/api/v1/examples/download", nil)
	if download.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", download.Code, download.Body.String())
	}
	if got := download.Header().Get("Content-Disposition"); got == "" || !bytes.Contains([]byte(got), []byte("demo.txt")) {
		t.Fatalf("expected attachment header, got %q", got)
	}
	if got := download.Header().Get("Content-Type"); got == "" || !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", got)
	}

	readerDownload := doDemoRequest(api, http.MethodGet, "/api/v1/examples/download-reader", nil)
	if readerDownload.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", readerDownload.Code, readerDownload.Body.String())
	}
	if got := readerDownload.Header().Get("Content-Disposition"); got == "" || !bytes.Contains([]byte(got), []byte("request.txt")) {
		t.Fatalf("expected reader download header, got %q", got)
	}
}

func TestDemoEndpoints_VersioningSSEAndWebSocket(t *testing.T) {
	api := newDemoAPI()

	v1 := doDemoRequest(api, http.MethodGet, "/api/v1/examples/versioned/info", nil)
	if v1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v1.Code, v1.Body.String())
	}
	if v1.Header().Get("Deprecation") != "" {
		t.Fatalf("did not expect deprecation header on v1, got %v", v1.Header())
	}
	if v1.Header().Get("Sunset") != "" || v1.Header().Get("Link") != "" {
		t.Fatalf("did not expect sunset/link headers on v1, got %v", v1.Header())
	}

	v0 := doDemoRequest(api, http.MethodGet, "/api/v0/examples/versioned/info", nil)
	if v0.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v0.Code, v0.Body.String())
	}
	if v0.Header().Get("Deprecation") != "true" {
		t.Fatalf("expected deprecation header, got %v", v0.Header())
	}
	if v0.Header().Get("Sunset") == "" || v0.Header().Get("Link") == "" {
		t.Fatalf("expected sunset and link headers, got %v", v0.Header())
	}

	sse := doDemoRequest(api, http.MethodGet, "/api/v1/examples/events?name=bot", nil)
	if sse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", sse.Code, sse.Body.String())
	}
	if got := sse.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", got)
	}
	if body := sse.Body.String(); !strings.Contains(body, "event: hello") || !strings.Contains(body, `"name":"bot"`) || !strings.Contains(body, `"transport":"sse"`) {
		t.Fatalf("unexpected SSE body %q", body)
	}

	server := httptest.NewServer(api.Handler())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/examples/ws?name=bot"
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := websocket.Message.Send(conn, "ping"); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}
	var message string
	if err := websocket.Message.Receive(conn, &message); err != nil {
		t.Fatalf("receive websocket message: %v", err)
	}
	if message != "bot:ping" {
		t.Fatalf("unexpected websocket message %q", message)
	}
}

func TestDemoEndpoints_OpenAPIVisibilityResponsesAndVersionedDocs(t *testing.T) {
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
	if _, ok := paths["/api/v1/examples/hidden"]; ok {
		t.Fatalf("expected hidden route to be absent from OpenAPI, got %v", paths)
	}

	requestMeta := paths["/api/v1/examples/request-meta"].(map[string]interface{})["get"].(map[string]interface{})
	responses := requestMeta["responses"].(map[string]interface{})
	if _, ok := responses["401"]; !ok {
		t.Fatalf("expected documented 401 response, got %v", responses)
	}
	if _, ok := responses["404"]; !ok {
		t.Fatalf("expected documented 404 response, got %v", responses)
	}

	cache := paths["/api/v1/examples/cache"].(map[string]interface{})["get"].(map[string]interface{})
	cacheHeaders := cache["responses"].(map[string]interface{})["200"].(map[string]interface{})["headers"].(map[string]interface{})
	if _, ok := cacheHeaders["ETag"]; !ok {
		t.Fatalf("expected ETag header in OpenAPI, got %v", cacheHeaders)
	}
	if _, ok := cacheHeaders["Cache-Control"]; !ok {
		t.Fatalf("expected Cache-Control header in OpenAPI, got %v", cacheHeaders)
	}

	uploadSingle := paths["/api/v1/examples/upload-single"].(map[string]interface{})["post"].(map[string]interface{})
	requestBody := uploadSingle["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	if _, ok := content["multipart/form-data"]; !ok {
		t.Fatalf("expected multipart form content, got %v", content)
	}

	download := paths["/api/v1/examples/download"].(map[string]interface{})["get"].(map[string]interface{})
	downloadResponses := download["responses"].(map[string]interface{})
	okResponse := downloadResponses["200"].(map[string]interface{})
	responseContent := okResponse["content"].(map[string]interface{})
	if _, ok := responseContent["application/octet-stream"]; !ok {
		t.Fatalf("expected binary response content, got %v", responseContent)
	}

	v1Docs := doDemoRequest(api, http.MethodGet, "/openapi/v1.json", nil)
	if v1Docs.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v1Docs.Code, v1Docs.Body.String())
	}
	var v1Spec map[string]interface{}
	if err := json.Unmarshal(v1Docs.Body.Bytes(), &v1Spec); err != nil {
		t.Fatalf("unmarshal v1 openapi: %v", err)
	}
	v1Paths := v1Spec["paths"].(map[string]interface{})
	if _, ok := v1Paths["/api/v1/examples/versioned/info"]; !ok {
		t.Fatalf("expected v1 versioned path, got %v", v1Paths)
	}
	if _, ok := v1Paths["/api/v0/examples/versioned/info"]; ok {
		t.Fatalf("did not expect v0 path in v1 docs, got %v", v1Paths)
	}

	v0Docs := doDemoRequest(api, http.MethodGet, "/openapi/v0.json", nil)
	if v0Docs.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", v0Docs.Code, v0Docs.Body.String())
	}
	var v0Spec map[string]interface{}
	if err := json.Unmarshal(v0Docs.Body.Bytes(), &v0Spec); err != nil {
		t.Fatalf("unmarshal v0 openapi: %v", err)
	}
	v0Paths := v0Spec["paths"].(map[string]interface{})
	if _, ok := v0Paths["/api/v0/examples/versioned/info"]; !ok {
		t.Fatalf("expected v0 versioned path, got %v", v0Paths)
	}
	if _, ok := v0Paths["/api/v1/examples/versioned/info"]; ok {
		t.Fatalf("did not expect v1 path in v0 docs, got %v", v0Paths)
	}
}
