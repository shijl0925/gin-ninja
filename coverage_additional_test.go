package ninja

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
)

type readErrorBody struct{}

func (readErrorBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (readErrorBody) Close() error             { return nil }

type badJSONMarshaler struct{}

func (badJSONMarshaler) MarshalJSON() ([]byte, error) { return nil, errors.New("marshal failed") }

type textValue string

func (tv textValue) MarshalText() ([]byte, error) { return []byte("txt:" + string(tv)), nil }

type badModelSchemaOut struct {
	ModelSchema[struct {
		Value badJSONMarshaler `json:"value"`
	}]
}

type testClaimsWithoutUserID struct{}

func serveAPIRequest(api *NinjaAPI, method, target string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestBindingAdditionalCoverage(t *testing.T) {
	t.Run("body read error and locale fallback", func(t *testing.T) {
		c, _ := newTestContext(http.MethodPost, "/", "")
		c.Request.Body = readErrorBody{}

		var in struct {
			Name string `json:"name"`
		}
		err := bindInput(c, http.MethodPost, &in)
		if err == nil || err.Error() != "read failed" {
			t.Fatalf("expected read error, got %v", err)
		}
		if got := localeFromContext(nil); got == "" {
			t.Fatal("expected default locale")
		}
		c.Set(contextkeys.Locale, "zh")
		if got := localeFromContext(c); got != "zh" {
			t.Fatalf("expected locale from context, got %q", got)
		}
	})

	t.Run("hasFormFields and defaults", func(t *testing.T) {
		type embedded struct {
			Flag bool `form:"flag"`
		}
		type input struct {
			embedded
			Token string `header:"X-Token" default:"guest"`
			Page  int    `form:"page" default:"7"`
		}
		if !hasFormFields(reflect.TypeOf(input{})) {
			t.Fatal("expected embedded form fields to be detected")
		}
		if hasFormFields(reflect.TypeOf(1)) {
			t.Fatal("expected non-struct to have no form fields")
		}

		c, _ := newTestContext(http.MethodGet, "/?flag=true", "")
		var in input
		if err := bindInput(c, http.MethodGet, &in); err != nil {
			t.Fatalf("bindInput: %v", err)
		}
		if in.Token != "guest" || in.Page != 7 || !in.Flag {
			t.Fatalf("unexpected defaults: %+v", in)
		}
	})

	t.Run("bad default, invalid multipart, and invalid multipart values", func(t *testing.T) {
		type badDefault struct {
			Count int `header:"X-Count" default:"nope"`
		}
		c, _ := newTestContext(http.MethodGet, "/", "")
		var bad badDefault
		err := bindInput(c, http.MethodGet, &bad)
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_DEFAULT_VALUE" {
			t.Fatalf("expected BAD_DEFAULT_VALUE, got %v", err)
		}

		c, _ = newTestContext(http.MethodPost, "/upload", "broken")
		c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
		var upload multipartBindInput
		err = bindMultipartFields(c, reflect.TypeOf(upload), reflect.ValueOf(&upload).Elem())
		if !errors.As(err, &apiErr) || apiErr.Code != "INVALID_MULTIPART" {
			t.Fatalf("expected INVALID_MULTIPART, got %v", err)
		}

		type badForm struct {
			Count int `form:"count"`
		}
		form := &multipart.Form{Value: map[string][]string{"count": {"bad"}}}
		var badFormValue badForm
		err = bindMultipartValue(reflect.TypeOf(badFormValue), reflect.ValueOf(&badFormValue).Elem(), form)
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_FORM_VALUE" {
			t.Fatalf("expected BAD_FORM_VALUE, got %v", err)
		}

		type badFile struct {
			File string `file:"file"`
		}
		form = &multipart.Form{File: map[string][]*multipart.FileHeader{"file": {&multipart.FileHeader{Filename: "a.txt"}}}}
		var badFileValue badFile
		err = bindMultipartValue(reflect.TypeOf(badFileValue), reflect.ValueOf(&badFileValue).Elem(), form)
		if !errors.As(err, &apiErr) || apiErr.Code != "BAD_FILE_FIELD" {
			t.Fatalf("expected BAD_FILE_FIELD, got %v", err)
		}
	})

	t.Run("string slice and file helper variants", func(t *testing.T) {
		var ints []int
		if err := setFieldFromStrings(reflect.ValueOf(&ints).Elem(), []string{"1", "2"}); err != nil {
			t.Fatalf("setFieldFromStrings slice: %v", err)
		}
		if len(ints) != 2 || ints[1] != 2 {
			t.Fatalf("unexpected parsed slice: %+v", ints)
		}

		var ptr *int
		if err := setFieldFromString(reflect.ValueOf(&ptr).Elem(), "3"); err != nil {
			t.Fatalf("setFieldFromString ptr: %v", err)
		}
		if ptr == nil || *ptr != 3 {
			t.Fatalf("unexpected parsed pointer: %+v", ptr)
		}

		file := &multipart.FileHeader{Filename: "a.txt"}
		var single *multipart.FileHeader
		if err := setFileField(reflect.ValueOf(&single).Elem(), []*multipart.FileHeader{file}); err != nil {
			t.Fatalf("setFileField single: %v", err)
		}
		var many []*multipart.FileHeader
		if err := setFileField(reflect.ValueOf(&many).Elem(), []*multipart.FileHeader{file}); err != nil {
			t.Fatalf("setFileField slice: %v", err)
		}

		c, _ := newTestContext(http.MethodPost, "/", "")
		c.Request.PostForm = make(map[string][]string)
		c.Request.PostForm.Set("enabled", "true")
		if !hasFormValue(c, "enabled") {
			t.Fatal("expected post form value to be detected")
		}
	})
}

func TestModelSchemaAdditionalCoverage(t *testing.T) {
	t.Run("bind model schema validation", func(t *testing.T) {
		if _, err := BindModelSchema[any](schemaModel{}); err == nil {
			t.Fatal("expected nil target type error")
		}
		if _, err := BindModelSchema[*publicSchema](schemaModel{}); err == nil {
			t.Fatal("expected pointer target error")
		}
		if _, err := BindModelSchema[int](schemaModel{}); err == nil {
			t.Fatal("expected non-struct target error")
		}
		if _, err := BindModelSchema[struct{}](schemaModel{}); err == nil {
			t.Fatal("expected missing embedded model schema error")
		}
	})

	t.Run("assign and serialize variants", func(t *testing.T) {
		type ptrSchema struct {
			*ModelSchema[schemaModel]
		}
		out, err := BindModelSchema[ptrSchema](&schemaModel{Name: "alice", Email: "a@example.com"})
		if err != nil {
			t.Fatalf("BindModelSchema: %v", err)
		}
		if out.ModelSchema == nil || out.Model.Name != "alice" {
			t.Fatalf("unexpected bound schema: %+v", out)
		}

		var ptr *schemaModel
		if err := assignModelSchemaModel(reflect.ValueOf(&ptr).Elem(), reflect.ValueOf(schemaModel{Name: "bob"})); err != nil {
			t.Fatalf("assignModelSchemaModel pointer: %v", err)
		}
		if ptr == nil || ptr.Name != "bob" {
			t.Fatalf("unexpected assigned pointer: %+v", ptr)
		}
		if err := assignModelSchemaModel(reflect.ValueOf(&ptr).Elem(), reflect.ValueOf(123)); err == nil {
			t.Fatal("expected incompatible assignment error")
		}

		if got := parseModelSchemaTag(""); got != nil {
			t.Fatalf("expected nil parseModelSchemaTag result, got %+v", got)
		}
		if got := normalizeModelSchemaNames([]string{" name ", "", "name"}); len(got) != 1 || got[0] != "name" {
			t.Fatalf("unexpected normalized names: %+v", got)
		}
		if containsModelSchemaName(nil, "name") {
			t.Fatal("expected empty name set to miss")
		}

		if value, err := serializeModelSchemaValue(reflect.ValueOf([]*schemaModel{{Name: "alice"}, nil}), modelSchemaFilter{}); err != nil || len(value.([]any)) != 2 {
			t.Fatalf("unexpected serialized slice: value=%+v err=%v", value, err)
		}
		if value, err := serializeModelSchemaValue(reflect.Value{}, modelSchemaFilter{}); err != nil || value != nil {
			t.Fatalf("unexpected invalid serialization: value=%+v err=%v", value, err)
		}
	})

	t.Run("marshal failure propagates", func(t *testing.T) {
		value := badModelSchemaOut{}
		value.Model.Value = badJSONMarshaler{}
		if _, err := value.MarshalJSON(); err == nil || !strings.Contains(err.Error(), "marshal failed") {
			t.Fatalf("expected marshal failure, got %v", err)
		}
	})

	t.Run("struct serialization helpers", func(t *testing.T) {
		type embedded struct {
			Hidden string `json:"hidden"`
		}
		type sample struct {
			embedded
			Name  string    `json:"name"`
			Empty string    `json:"empty,omitempty"`
			Text  textValue `json:"text"`
		}

		serialized, err := serializeModelSchemaStruct(reflect.ValueOf(sample{
			embedded: embedded{Hidden: "secret"},
			Name:     "alice",
			Text:     "demo",
		}), newModelSchemaFilter(nil, []string{"name"}))
		if err != nil {
			t.Fatalf("serializeModelSchemaStruct: %v", err)
		}
		if _, ok := serialized["name"]; ok {
			t.Fatalf("expected excluded field to be omitted: %+v", serialized)
		}
		if serialized["text"] == nil {
			t.Fatalf("expected embedded/custom values, got %+v", serialized)
		}

		var anyValue any = pointerMarshaler("value")
		if preserved, ok := preserveCustomJSONValue(reflect.ValueOf(anyValue)); !ok || preserved == nil {
			t.Fatalf("expected pointer receiver marshaler to be preserved, got value=%v ok=%v", preserved, ok)
		}

		field := reflect.TypeOf(sample{}).Field(2)
		if !isJSONOmitEmpty(field) {
			t.Fatal("expected omitempty tag to be detected")
		}

		filtered := newModelSchemaFilter(nil, []string{"name"})
		structField := reflect.TypeOf(sample{}).Field(1)
		if filtered.includes(structField, "name") {
			t.Fatal("expected excluded field to be rejected")
		}
		if value, err := serializeModelSchemaValue(reflect.ValueOf((*sample)(nil)), modelSchemaFilter{}); err != nil || value != nil {
			t.Fatalf("expected nil pointer serialization, got value=%v err=%v", value, err)
		}
		if value, err := serializeModelSchemaElement(reflect.Value{}, modelSchemaFilter{}); err != nil || value != nil {
			t.Fatalf("expected invalid element serialization, got value=%v err=%v", value, err)
		}
		if _, ok := customJSONValue(reflect.Value{}); ok {
			t.Fatal("expected invalid custom JSON value to be ignored")
		}
		direct := ModelSchema[badJSONMarshaler]{Model: badJSONMarshaler{}}
		if _, err := direct.MarshalJSON(); err == nil {
			t.Fatal("expected direct marshaler to fail")
		}
	})
}

func TestSchemaAdditionalCoverage(t *testing.T) {
	t.Parallel()

	registry := newSchemaRegistry()
	if got := registry.schemaForType(reflect.TypeOf(float32(0))); got.Format != "float" {
		t.Fatalf("expected float32 schema, got %+v", got)
	}
	if got := registry.schemaForType(reflect.TypeOf(float64(0))); got.Format != "double" {
		t.Fatalf("expected float64 schema, got %+v", got)
	}
	if got := registry.schemaForType(reflect.TypeOf(map[string]int{})); got.Type != "object" {
		t.Fatalf("expected map schema, got %+v", got)
	}
	if got := registry.schemaForType(reflect.TypeOf((chan int)(nil))); got.Type != "string" {
		t.Fatalf("expected fallback string schema, got %+v", got)
	}
	if got := registry.buildStructSchema(reflect.TypeOf(struct {
		private string
		Name    string  `json:",omitempty" default:"demo"`
		Enabled bool    `default:"true"`
		Count   int     `default:"7"`
		Score   uint    `default:"9"`
		Ratio   float64 `default:"1.5"`
	}{})); got.Properties["name"].Default != "demo" {
		t.Fatalf("unexpected struct schema defaults: %+v", got)
	}
	if got := modelSchemaComponentName(reflect.TypeOf(schemaModel{}), newModelSchemaFilter([]string{"name"}, []string{"password"})); !strings.Contains(got, "fields") {
		t.Fatalf("unexpected model schema component name %q", got)
	}
	if got := typeName(reflect.TypeOf(&schemaSample{})); got != "schemaSample" {
		t.Fatalf("unexpected type name %q", got)
	}
	field := reflect.TypeOf(struct {
		Value string `json:",omitempty"`
	}{}).Field(0)
	if got := jsonFieldName(field); got != "value" {
		t.Fatalf("expected fallback json field name, got %q", got)
	}
	if got := defaultJSONFieldName("URLValue"); got != "urlValue" {
		t.Fatalf("unexpected default JSON field name %q", got)
	}
	if value, ok := defaultValueForType(reflect.TypeOf(true), "nope"); ok || value != nil {
		t.Fatalf("expected invalid bool default to fail, got value=%v ok=%v", value, ok)
	}
}

func TestCacheAndInternalRouteAdditionalCoverage(t *testing.T) {
	t.Run("openapi cache and current api", func(t *testing.T) {
		api := New(Config{Title: "cache", Version: "1", Versions: map[string]VersionConfig{"v1": {Prefix: "/v1"}}})
		api.openAPICache.main = []byte("cached")
		if cached, err := api.openAPIBytes(); err != nil || string(cached) != "cached" {
			t.Fatalf("expected cached main spec, got %q err=%v", string(cached), err)
		}
		api.invalidateOpenAPICache()
		mainBytes, err := api.openAPIBytes()
		if err != nil || len(mainBytes) == 0 {
			t.Fatalf("openAPIBytes: bytes=%q err=%v", string(mainBytes), err)
		}
		if again, err := api.openAPIBytes(); err != nil || !bytes.Equal(mainBytes, again) {
			t.Fatalf("expected cached main spec, got err=%v", err)
		}
		if _, ok, err := api.versionOpenAPIBytes("missing"); err != nil || ok {
			t.Fatalf("expected missing version spec, got ok=%v err=%v", ok, err)
		}
		_ = api.versionSpec("v1")
		api.openAPICache.versions = map[string][]byte{"v1": []byte("cached-version")}
		if cached, ok, err := api.versionOpenAPIBytes("v1"); err != nil || !ok || string(cached) != "cached-version" {
			t.Fatalf("expected cached version spec, got %q ok=%v err=%v", string(cached), ok, err)
		}
		api.invalidateOpenAPICache()
		versionBytes, ok, err := api.versionOpenAPIBytes("v1")
		if err != nil || !ok || len(versionBytes) == 0 {
			t.Fatalf("versionOpenAPIBytes: ok=%v err=%v", ok, err)
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		if got, ok := currentAPI(nil); ok || got != nil {
			t.Fatalf("expected nil currentAPI result, got api=%v ok=%v", got, ok)
		}
		api.attachContext()(c)
		if got, ok := currentAPI(c); !ok || got != api {
			t.Fatalf("expected attached api, got api=%v ok=%v", got, ok)
		}
		c.Set(ninjaAPIContextKey, "bad")
		if got, ok := currentAPI(c); ok || got != nil {
			t.Fatalf("expected bad type to be ignored, got api=%v ok=%v", got, ok)
		}
	})

	t.Run("typed middleware, versioned docs, and cache helpers", func(t *testing.T) {
		api := New(Config{Title: "routes", Version: "1", Versions: map[string]VersionConfig{"v1": {Prefix: "/v1"}}})
		router := NewRouter("/items", WithVersion("v1"))
		router.Use(func(ctx *Context) error {
			return NewError(http.StatusTeapot, "blocked")
		})
		Get(router, "/", func(ctx *Context, in *struct{}) (*struct{}, error) {
			return &struct{}{}, nil
		})
		api.AddRouter(router)

		if w := serveAPIRequest(api, http.MethodGet, "/v1/items/", nil, nil); w.Code != http.StatusTeapot {
			t.Fatalf("expected middleware error response, got %d", w.Code)
		}
		if w := serveAPIRequest(api, http.MethodGet, "/docs/v1", nil, nil); w.Code != http.StatusOK {
			t.Fatalf("expected versioned docs, got %d", w.Code)
		}
		if w := serveAPIRequest(api, http.MethodGet, "/openapi/v1.json", nil, nil); w.Code != http.StatusOK {
			t.Fatalf("expected versioned openapi, got %d", w.Code)
		}
		if w := serveAPIRequest(api, http.MethodGet, "/docs/v2", nil, nil); w.Code != http.StatusNotFound {
			t.Fatalf("expected missing docs 404, got %d", w.Code)
		}

		store := NewMemoryCacheStoreWithLimit(1)
		store.Set("old", &CachedResponse{Status: http.StatusOK, Body: []byte("old")})
		store.Set("new", &CachedResponse{Status: http.StatusOK, Body: []byte("new")})
		if _, ok := store.Get("old"); ok {
			t.Fatal("expected oldest cache entry to be evicted")
		}
		store.Set("expired", &CachedResponse{Expires: time.Now().Add(-time.Second)})
		if _, ok := store.Get("expired"); ok {
			t.Fatal("expected expired cache entry to be dropped")
		}
		if got := defaultCacheKey(nil); got != "" {
			t.Fatalf("expected empty cache key for nil context, got %q", got)
		}
		if !matchesETag(`"a", "b"`, `"b"`) {
			t.Fatal("expected ETag match")
		}
		if !matchesETag(`"tag"`, `W/"tag"`) {
			t.Fatal("expected strong If-None-Match to match weak cached ETag")
		}
		if len(splitCommaValues(" a, ,b ")) != 2 {
			t.Fatal("expected split comma values to trim empties")
		}
		if cloneCachedResponse(nil) != nil {
			t.Fatal("expected nil cached response clone")
		}

		c, w := newTestContext(http.MethodHead, "/", "")
		writeCachedResponse(c, &CachedResponse{
			Status: http.StatusOK,
			Header: http.Header{"ETag": []string{`"tag"`}},
			Body:   []byte("payload"),
		}, "public, max-age=60")
		if w.Code != http.StatusOK || w.Body.Len() != 0 {
			t.Fatalf("expected HEAD cached response without body, got status=%d body=%q", w.Code, w.Body.String())
		}

		c, w = newTestContext(http.MethodGet, "/", "")
		writeCachedResponse(c, nil, "")
		if c.Writer.Status() != http.StatusNoContent || w.Code != http.StatusOK {
			t.Fatalf("expected nil cached response status 204, got writer=%d recorder=%d", c.Writer.Status(), w.Code)
		}
	})
}

func TestCoreHelperAdditionalCoverage(t *testing.T) {
	t.Run("context request id and user id fallbacks", func(t *testing.T) {
		c, _ := newTestContext(http.MethodGet, "/", "")
		ctx := newContext(c)
		if got := ctx.RequestID(); got != "" {
			t.Fatalf("expected empty request id, got %q", got)
		}
		c.Set(requestIDContextKey, "req-1")
		if got := ctx.RequestID(); got != "req-1" {
			t.Fatalf("RequestID() = %q, want req-1", got)
		}

		if got := ctx.GetUserID(); got != 0 {
			t.Fatalf("expected empty user id, got %d", got)
		}
		c.Set(contextkeys.JWTClaims, testClaimsWithoutUserID{})
		if got := ctx.GetUserID(); got != 0 {
			t.Fatalf("expected unsupported claims to return 0, got %d", got)
		}
	})

	t.Run("memory cache delete and cache helpers", func(t *testing.T) {
		store := NewMemoryCacheStore()
		store.Set("users:1", &CachedResponse{Status: http.StatusOK, Body: []byte("one")})
		store.AddTags("users:1", "users")
		store.Delete("")
		store.Delete("users:1")
		if _, ok := store.Get("users:1"); ok {
			t.Fatal("expected Delete to remove item")
		}
		if removed := store.InvalidateTags("users"); removed != 0 {
			t.Fatalf("expected no tagged items after delete, got %d", removed)
		}

		if key, cacheStore := cacheLookup(&operation{}, nil); key != "" || cacheStore != nil {
			t.Fatalf("expected empty cache lookup, got key=%q store=%v", key, cacheStore)
		}
		op := &operation{cache: &routeCacheConfig{ttl: time.Minute, keyFn: func(*Context) string { return "k" }, store: store}}
		if key, cacheStore := cacheLookup(op, nil); key != "k" || cacheStore != store {
			t.Fatalf("unexpected cache lookup result key=%q store=%v", key, cacheStore)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ginCtx.Request = req
		ctx := &Context{Context: ginCtx}
		cacheStoreSet(ctx, store, "users:2", &CachedResponse{Status: http.StatusCreated, Body: []byte("two")})
		value, ok := cacheStoreGet(ctx, store, "users:2")
		if !ok || value == nil || string(value.Body) != "two" {
			t.Fatalf("cacheStoreGet() = (%+v, %v), want cached value", value, ok)
		}
		if value, ok := cacheStoreGet(nil, nil, "users:2"); ok || value != nil {
			t.Fatalf("expected nil store lookup miss, got value=%v ok=%v", value, ok)
		}
		cacheStoreSet(nil, nil, "users:3", &CachedResponse{Status: http.StatusOK})
	})

	t.Run("capture writer and upload helper", func(t *testing.T) {
		c, _ := newTestContext(http.MethodGet, "/", "")
		recorder := newCaptureResponseWriter(c.Writer)
		recorder.WriteHeader(http.StatusAccepted)
		recorder.Flush()
		if recorder.Status() != http.StatusAccepted {
			t.Fatalf("Status() = %d, want %d", recorder.Status(), http.StatusAccepted)
		}

		if newUploadedFile(nil) != nil {
			t.Fatal("expected nil uploaded file wrapper")
		}
	})

	t.Run("version spec caching and rate limit disable", func(t *testing.T) {
		api := New(Config{
			Title:    "versions",
			Version:  "1.0.0",
			Versions: map[string]VersionConfig{"v1": {Prefix: "/v1"}},
		})
		spec := api.versionSpec("v1")
		if spec == nil {
			t.Fatal("expected version spec")
		}
		if cached, ok := api.lookupVersionSpec("v1"); !ok || cached != spec {
			t.Fatalf("expected cached version spec, got spec=%v ok=%v", cached, ok)
		}

		op := &operation{}
		RateLimit(0)(op)
		if op.rateLimit != nil {
			t.Fatal("expected RateLimit(0) to disable limiter")
		}
	})

	t.Run("shutdown without active server is a no-op", func(t *testing.T) {
		api := New(Config{Title: "shutdown"})
		called := false
		api.OnShutdown(func(ctx context.Context, api *NinjaAPI) error {
			called = true
			return nil
		})
		if err := api.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown(): %v", err)
		}
		if called {
			t.Fatal("expected shutdown hooks to be skipped before startup")
		}
	})
}

func TestOperationLimitAdditionalCoverage(t *testing.T) {
	t.Run("rate limiting and timeout", func(t *testing.T) {
		limiter := newRateLimiter(1, 0)
		now := time.Now()
		ip := "192.0.2.1"
		if !limiter.allow(ip, now) {
			t.Fatal("expected first request to be allowed")
		}
		if limiter.allow(ip, now) {
			t.Fatal("expected second request to be limited")
		}

		router := gin.New()
		router.GET("/limited", wrapRateLimit(newRateLimiter(1, 1), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		}))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected first limited request to pass, got %d", w.Code)
		}
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/limited", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected second limited request to fail, got %d", w.Code)
		}

		router = gin.New()
		router.GET("/fast", wrapTimeout(50*time.Millisecond, func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		}))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fast", nil))
		if w.Code != http.StatusOK || w.Body.String() != "ok" {
			t.Fatalf("expected timeout wrapper success, got status=%d body=%q", w.Code, w.Body.String())
		}

		router = gin.New()
		router.GET("/slow", wrapTimeout(10*time.Millisecond, func(c *gin.Context) {
			time.Sleep(30 * time.Millisecond)
			c.String(http.StatusOK, "late")
		}))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slow", nil))
		if w.Code != http.StatusRequestTimeout {
			t.Fatalf("expected timeout response, got %d", w.Code)
		}
	})
}

// TestWrapTimeoutGoroutineExitsAfterTimeout verifies that the handler goroutine
// spawned by wrapTimeout exits after the timeout fires, once the handler
// honours context cancellation. This guards against goroutines that run
// indefinitely after the HTTP response has already been sent.
func TestWrapTimeoutGoroutineExitsAfterTimeout(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(1)

	router := gin.New()
	router.GET("/ctx-aware", wrapTimeout(10*time.Millisecond, func(c *gin.Context) {
		defer wg.Done()
		// Simulate a context-aware handler: block until context is cancelled.
		<-c.Request.Context().Done()
	}))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ctx-aware", nil))
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d", w.Code)
	}

	// The goroutine must exit within a generous grace period after the timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// goroutine exited as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handler goroutine did not exit within 500ms after timeout")
	}
}

// TestWrapTimeoutContextCancelledInHandler verifies that the context passed to
// the handler is already cancelled when the timeout fires, so the handler can
// detect it and stop early.
func TestWrapTimeoutContextCancelledInHandler(t *testing.T) {
	t.Parallel()

	var capturedCtx context.Context
	var mu sync.Mutex

	router := gin.New()
	router.GET("/capture", wrapTimeout(10*time.Millisecond, func(c *gin.Context) {
		mu.Lock()
		capturedCtx = c.Request.Context()
		mu.Unlock()
		time.Sleep(30 * time.Millisecond)
	}))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/capture", nil))
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d", w.Code)
	}

	// Allow the background goroutine to complete so capturedCtx is set.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	ctx := capturedCtx
	mu.Unlock()

	if ctx == nil {
		t.Fatal("expected handler to capture context, got nil")
	}
	if ctx.Err() == nil {
		t.Fatal("expected handler context to be cancelled after timeout, but Err() is nil")
	}
}

func TestNinjaAdditionalBranches(t *testing.T) {
	t.Parallel()

	api := New(Config{Title: "branches", Version: "1"})
	api.RegisterErrorMapper(nil)
	if api.mapError(nil) != nil {
		t.Fatal("expected nil mapError result")
	}
	if got := api.lookupVersion("v2"); got.Prefix != "/v2" {
		t.Fatalf("expected default version prefix, got %+v", got)
	}
}

func TestAPIMergesCurrentGlobalErrorMappers(t *testing.T) {
	errorMappersMu.Lock()
	original := append([]ErrorMapper(nil), errorMappers...)
	errorMappers = defaultErrorMappers()
	errorMappersMu.Unlock()
	defer func() {
		errorMappersMu.Lock()
		errorMappers = original
		errorMappersMu.Unlock()
	}()

	api := New(Config{Title: "branches", Version: "1"})
	RegisterErrorMapper(func(err error) error {
		if errors.Is(err, errBadRequest) {
			return NewError(http.StatusTeapot, "late global mapper")
		}
		return nil
	})

	mapped := api.mapError(errBadRequest)
	if !errors.Is(mapped, NewError(http.StatusTeapot, "late global mapper")) {
		t.Fatalf("expected late global mapper to apply, got %v", mapped)
	}
}

func TestRunHandlesInterruptSignal(t *testing.T) {
	api := New(Config{Title: "signal", Version: "1"})
	done := make(chan error, 1)
	go func() {
		done <- api.Run("127.0.0.1:0")
	}()

	time.Sleep(100 * time.Millisecond)
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to stop")
	}
}

func TestCaptureResponseWriterAndMultipartHelpers(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	writer := newCaptureResponseWriter(c.Writer)
	writer.WriteHeaderNow()
	if writer.Status() != http.StatusOK {
		t.Fatalf("expected WriteHeaderNow to default to 200, got %d", writer.Status())
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("title", "demo"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !strings.HasPrefix(mw.FormDataContentType(), "multipart/form-data") {
		t.Fatal("expected multipart content type")
	}
}
