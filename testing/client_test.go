package ninjatest_test

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
	ninjatest "github.com/shijl0925/gin-ninja/testing"
)

type testHelloInput struct {
	Name string `query:"name"`
}

type testHelloOutput struct {
	Message string `json:"message"`
}

func testHello(_ *ninja.Context, in *testHelloInput) (*testHelloOutput, error) {
	return &testHelloOutput{Message: "hello " + in.Name}, nil
}

func TestClientRouterTarget(t *testing.T) {
	router := ninja.NewRouter("/users")
	ninja.Get(router, "/", testHello)

	client := ninjatest.NewWithT(t, router)
	resp := client.Get("/users/", ninjatest.Query("name", "alice"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.String())
	}

	var out testHelloOutput
	if err := resp.DecodeJSON(&out); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if out.Message != "hello alice" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

type testCreateInput struct {
	Name  string `json:"name"`
	Trace string `header:"X-Trace-ID"`
	Mode  string `cookie:"mode"`
}

type testCreateOutput struct {
	Name  string `json:"name"`
	Trace string `json:"trace"`
	Mode  string `json:"mode"`
}

func testCreate(_ *ninja.Context, in *testCreateInput) (*testCreateOutput, error) {
	return &testCreateOutput{Name: in.Name, Trace: in.Trace, Mode: in.Mode}, nil
}

func TestClientJSONHeadersAndCookies(t *testing.T) {
	api := ninja.New(ninja.Config{Title: "test"})
	router := ninja.NewRouter("/users")
	ninja.Post(router, "/", testCreate)
	api.AddRouter(router)

	client := ninjatest.NewWithT(t, api, ninjatest.WithHeader("X-Trace-ID", "default"))
	resp := client.Post("/users/", map[string]string{"name": "alice"},
		ninjatest.Header("X-Trace-ID", "request"),
		ninjatest.Cookie(&http.Cookie{Name: "mode", Value: "test"}),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.String())
	}

	var out testCreateOutput
	if err := resp.JSON(&out); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if out.Name != "alice" || out.Trace != "request" || out.Mode != "test" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestClientWithHeaderOverwritesDefault(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.Header.Values("Authorization")
		if len(values) != 1 || values[0] != "Bearer B" {
			t.Fatalf("unexpected Authorization headers: %v", values)
		}
		w.WriteHeader(http.StatusNoContent)
	}), ninjatest.WithHeader("Authorization", "Bearer A"), ninjatest.WithHeader("Authorization", "Bearer B"))

	resp := client.Get("/")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

type testFormInput struct {
	Name string `form:"name"`
}

func testForm(_ *ninja.Context, in *testFormInput) (*testHelloOutput, error) {
	return &testHelloOutput{Message: "form " + in.Name}, nil
}

func TestClientFormBodyAndRawRequest(t *testing.T) {
	router := ninja.NewRouter("/forms")
	ninja.Post(router, "/", testForm)

	client := ninjatest.NewWithT(t, router)
	req := client.NewRequest(http.MethodPost, "/forms/", url.Values{"name": {"alice"}})
	resp := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.String())
	}

	var out testHelloOutput
	if err := resp.DecodeJSON(&out); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if out.Message != "form alice" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

type testUploadInput struct {
	Title string              `form:"title"`
	File  *ninja.UploadedFile `file:"file"`
}

type testUploadOutput struct {
	Title    string `json:"title"`
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

func testUpload(_ *ninja.Context, in *testUploadInput) (*testUploadOutput, error) {
	content, err := in.File.Bytes()
	if err != nil {
		return nil, err
	}
	return &testUploadOutput{
		Title:    in.Title,
		FileName: in.File.Filename,
		Content:  string(content),
	}, nil
}

func TestClientMultipartBody(t *testing.T) {
	router := ninja.NewRouter("/uploads")
	ninja.Post(router, "/", testUpload)

	client := ninjatest.NewWithT(t, router)
	resp := client.Post("/uploads/", ninjatest.Multipart(
		url.Values{"title": {"demo"}},
		ninjatest.File("file", "demo.txt", "hello multipart"),
	))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.String())
	}

	var out testUploadOutput
	if err := resp.DecodeJSON(&out); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if out.Title != "demo" || out.FileName != "demo.txt" || out.Content != "hello multipart" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestClientHTTPHandlerTarget(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Path", r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))

	resp := client.Get("/ping")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Path") != "/ping" || resp.String() != "ok" {
		t.Fatalf("unexpected response: header=%q body=%q", resp.Header.Get("X-Path"), resp.String())
	}
}

func TestClientHeadersAndMethodHelpers(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Values("X-Default"); len(got) != 2 || got[0] != "A" || got[1] != "B" {
			t.Fatalf("unexpected default headers: %v", got)
		}
		if got := r.Header.Values("X-Request"); len(got) != 2 || got[0] != "C" || got[1] != "D" {
			t.Fatalf("unexpected request headers: %v", got)
		}
		w.Header().Set("X-Method", r.Method)
		w.WriteHeader(http.StatusOK)
	}), ninjatest.WithHeaders(http.Header{"X-Default": {"A", "B"}}))

	requestHeaders := ninjatest.Headers(http.Header{"X-Request": {"C", "D"}})
	for _, tc := range []struct {
		name   string
		do     func() *ninjatest.Response
		method string
	}{
		{name: "put", do: func() *ninjatest.Response { return client.Put("/", "body", requestHeaders) }, method: http.MethodPut},
		{name: "patch", do: func() *ninjatest.Response { return client.Patch("/", "body", requestHeaders) }, method: http.MethodPatch},
		{name: "delete", do: func() *ninjatest.Response { return client.Delete("/", requestHeaders) }, method: http.MethodDelete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := tc.do()
			if resp.StatusCode != http.StatusOK || resp.Header.Get("X-Method") != tc.method {
				t.Fatalf("unexpected response: status=%d method=%q", resp.StatusCode, resp.Header.Get("X-Method"))
			}
		})
	}
}

func TestClientRawBodyVariants(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r.Body); err != nil {
			t.Fatalf("ReadFrom: %v", err)
		}
		w.Header().Set("X-Content-Type", r.Header.Get("Content-Type"))
		_, _ = w.Write(buf.Bytes())
	}))

	for _, tc := range []struct {
		name        string
		body        any
		wantBody    string
		contentType string
	}{
		{name: "bytes", body: []byte("raw bytes"), wantBody: "raw bytes"},
		{name: "string", body: "raw string", wantBody: "raw string"},
		{name: "reader", body: strings.NewReader("raw reader"), wantBody: "raw reader"},
		{name: "form", body: url.Values{"name": {"alice"}}, wantBody: "name=alice", contentType: "application/x-www-form-urlencoded"},
		{name: "scalar-json", body: 42, wantBody: "42", contentType: "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := client.Post("/", tc.body)
			if resp.String() != tc.wantBody {
				t.Fatalf("expected body %q, got %q", tc.wantBody, resp.String())
			}
			if resp.Header.Get("X-Content-Type") != tc.contentType {
				t.Fatalf("expected content type %q, got %q", tc.contentType, resp.Header.Get("X-Content-Type"))
			}
		})
	}
}

func TestClientMultipartValueBody(t *testing.T) {
	fields := url.Values{"title": {"demo"}}
	body := ninjatest.Multipart(fields,
		ninjatest.File("file", "bytes.txt", []byte("bytes")),
		ninjatest.File("file", "reader.txt", strings.NewReader("reader")),
	)
	fields.Set("title", "changed")

	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("title"); got != "demo" {
			t.Fatalf("expected cloned title value, got %q", got)
		}
		files := r.MultipartForm.File["file"]
		if len(files) != 2 {
			t.Fatalf("expected two files, got %d", len(files))
		}
		for _, file := range files {
			opened, err := file.Open()
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(opened); err != nil {
				_ = opened.Close()
				t.Fatalf("ReadFrom: %v", err)
			}
			if err := opened.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			w.Header().Add("X-File", file.Filename+":"+buf.String())
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	resp := client.Post("/", *body)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	got := resp.Header.Values("X-File")
	if len(got) != 2 || got[0] != "bytes.txt:bytes" || got[1] != "reader.txt:reader" {
		t.Fatalf("unexpected files: %v", got)
	}
}

func TestClientWithConfigRouterTarget(t *testing.T) {
	router := ninja.NewRouter("/configured")
	ninja.Get(router, "/", testHello)

	client := ninjatest.NewWithT(t, router, ninjatest.WithConfig(ninja.Config{Title: "Configured API"}))
	resp := client.Get("/configured/", ninjatest.Query("name", "alice"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.String())
	}
}

func TestResponseNilHelpers(t *testing.T) {
	var resp *ninjatest.Response
	if resp.String() != "" {
		t.Fatalf("expected empty nil response string")
	}
	if err := resp.DecodeJSON(&testHelloOutput{}); err == nil {
		t.Fatalf("expected nil response decode error")
	}
}

func TestClientPanicsForInvalidTargets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target any
	}{
		{name: "nil-api", target: (*ninja.NinjaAPI)(nil)},
		{name: "nil-router", target: (*ninja.Router)(nil)},
		{name: "unsupported", target: "unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() {
				_ = ninjatest.New(tc.target)
			})
		})
	}
}

func TestClientPanicsForInvalidRequests(t *testing.T) {
	client := ninjatest.New(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for _, tc := range []struct {
		name string
		run  func()
	}{
		{name: "nil-request", run: func() { _ = client.Do(nil) }},
		{name: "json-encode", run: func() { _ = client.Post("/", func() {}) }},
		{name: "nil-multipart", run: func() { _ = client.Post("/", (*ninjatest.MultipartBody)(nil)) }},
		{name: "empty-file-field", run: func() { _ = client.Post("/", ninjatest.Multipart(nil, ninjatest.File("", "file.txt", "body"))) }},
		{name: "nil-file-body", run: func() { _ = client.Post("/", ninjatest.Multipart(nil, ninjatest.File("file", "file.txt", nil))) }},
		{name: "unsupported-file-body", run: func() { _ = client.Post("/", ninjatest.Multipart(nil, ninjatest.File("file", "file.txt", 1))) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, tc.run)
		})
	}
}

func TestClientInvalidRequestReportsThroughTestingT(t *testing.T) {
	fake := &recordingT{}
	client := ninjatest.New(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), ninjatest.WithT(fake))

	assertPanics(t, func() {
		_ = client.Do(nil)
	})
	if !fake.helperCalled {
		t.Fatalf("expected Helper to be called")
	}
	if fake.message == "" {
		t.Fatalf("expected Fatalf message")
	}
}

func TestCookieNilIsIgnored(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.Cookies()) != 0 {
			t.Fatalf("expected no cookies, got %v", r.Cookies())
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	resp := client.Get("/", ninjatest.Cookie(nil))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

type recordingT struct {
	helperCalled bool
	message      string
}

func (r *recordingT) Helper() {
	r.helperCalled = true
}

func (r *recordingT) Fatalf(format string, args ...any) {
	r.message = format
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}
