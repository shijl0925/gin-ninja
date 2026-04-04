package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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
	ninja.Post(router, "/upload-single", UploadSingleDemo)
	ninja.Post(router, "/upload-many", UploadManyDemo)
	ninja.Get(router, "/download", DownloadDemo)
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

func TestDemoEndpoints_FileUploadAndDownload(t *testing.T) {
	api := newDemoAPI()

	single := doMultipartDemoRequest(t, api, "/examples/upload-single", map[string]string{
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

	multi := doMultipartDemoRequest(t, api, "/examples/upload-many", map[string]string{
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

	download := doDemoRequest(api, http.MethodGet, "/examples/download", nil)
	if download.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", download.Code, download.Body.String())
	}
	if got := download.Header().Get("Content-Disposition"); got == "" || !bytes.Contains([]byte(got), []byte("demo.txt")) {
		t.Fatalf("expected attachment header, got %q", got)
	}
	if got := download.Header().Get("Content-Type"); got == "" || got[:10] != "text/plain" {
		t.Fatalf("expected text/plain content type, got %q", got)
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

	uploadSingle := paths["/examples/upload-single"].(map[string]interface{})["post"].(map[string]interface{})
	requestBody := uploadSingle["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	if _, ok := content["multipart/form-data"]; !ok {
		t.Fatalf("expected multipart form content, got %v", content)
	}

	download := paths["/examples/download"].(map[string]interface{})["get"].(map[string]interface{})
	downloadResponses := download["responses"].(map[string]interface{})
	okResponse := downloadResponses["200"].(map[string]interface{})
	responseContent := okResponse["content"].(map[string]interface{})
	if _, ok := responseContent["application/octet-stream"]; !ok {
		t.Fatalf("expected binary response content, got %v", responseContent)
	}
}
