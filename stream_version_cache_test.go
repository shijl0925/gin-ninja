package ninja

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type sseStringer string

func (s sseStringer) String() string { return string(s) }

type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *failingResponseWriter) WriteHeader(statusCode int) {}

func (w *failingResponseWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (w *failingResponseWriter) Flush() {}

func TestWebSocketConnNilHelpersReturnInternalError(t *testing.T) {
	t.Parallel()

	var conn *WebSocketConn

	if err := conn.SendJSON(map[string]string{"ok": "1"}); !IsInternal(err) {
		t.Fatalf("SendJSON() error = %v, want internal error", err)
	}
	if err := conn.ReceiveJSON(&map[string]string{}); !IsInternal(err) {
		t.Fatalf("ReceiveJSON() error = %v, want internal error", err)
	}
	if err := conn.SendText("ping"); !IsInternal(err) {
		t.Fatalf("SendText() error = %v, want internal error", err)
	}
	if _, err := conn.ReceiveText(); !IsInternal(err) {
		t.Fatalf("ReceiveText() error = %v, want internal error", err)
	}
}

func TestWebSocketConnJSONHelpers(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		wrapped := &WebSocketConn{Conn: conn}
		var input map[string]string
		if err := wrapped.ReceiveJSON(&input); err != nil {
			t.Errorf("ReceiveJSON() error = %v", err)
			return
		}
		if err := wrapped.SendJSON(map[string]string{"reply": input["name"] + "-ok"}); err != nil {
			t.Errorf("SendJSON() error = %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	client := &WebSocketConn{Conn: conn}
	if err := client.SendJSON(map[string]string{"name": "bot"}); err != nil {
		t.Fatalf("SendJSON() error = %v", err)
	}

	var reply map[string]string
	if err := client.ReceiveJSON(&reply); err != nil {
		t.Fatalf("ReceiveJSON() error = %v", err)
	}
	if reply["reply"] != "bot-ok" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestSSEDataFormatsCommonTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "string", value: "hello", want: "hello"},
		{name: "bytes", value: []byte("world"), want: "world"},
		{name: "stringer", value: sseStringer("fmt"), want: "fmt"},
		{name: "duration", value: 1500 * time.Millisecond, want: "1.5s"},
		{name: "json", value: map[string]int{"count": 2}, want: `{"count":2}`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sseData(tc.value); got != tc.want {
				t.Fatalf("sseData(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestSSEStreamSend(t *testing.T) {
	t.Run("writes framed event", func(t *testing.T) {
		c, w := newTestContext(http.MethodGet, "/events", "")
		stream := &SSEStream{c: c}

		err := stream.Send(SSEEvent{
			ID:    "42",
			Event: "update",
			Data:  "first\nsecond",
			Retry: 1500 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		if !stream.sent {
			t.Fatal("expected stream to be marked sent")
		}
		if got := w.Body.String(); got != "id: 42\nevent: update\nretry: 1500\ndata: first\ndata: second\n\n" {
			t.Fatalf("unexpected SSE frame %q", got)
		}
	})

	t.Run("nil stream returns internal error", func(t *testing.T) {
		var stream *SSEStream
		if err := stream.Send(SSEEvent{Data: "ignored"}); !IsInternal(err) {
			t.Fatalf("Send() error = %v, want internal error", err)
		}
	})

	t.Run("write failure propagates", func(t *testing.T) {
		base := &failingResponseWriter{}
		c, _ := gin.CreateTestContext(base)
		c.Request = httptest.NewRequest(http.MethodGet, "/events", nil)

		stream := &SSEStream{c: c}
		if err := stream.Send(SSEEvent{Data: "boom"}); err == nil || err.Error() != "write failed" {
			t.Fatalf("expected write failure, got %v", err)
		}
	})
}

func TestVersioningHelpersAndNotFound(t *testing.T) {
	t.Parallel()

	if got := versionedDocsPattern("/docs/"); got != "/docs/:version" {
		t.Fatalf("versionedDocsPattern() = %q", got)
	}
	if got := versionedDocsPath("/docs/", "v1"); got != "/docs/v1" {
		t.Fatalf("versionedDocsPath() = %q", got)
	}
	if got := versionedOpenAPIPattern("/openapi.json"); got != "/openapi/:version.json" {
		t.Fatalf("versionedOpenAPIPattern() = %q", got)
	}
	if got := versionedOpenAPIPath("/openapi.json", "v2"); got != "/openapi/v2.json" {
		t.Fatalf("versionedOpenAPIPath() = %q", got)
	}
	if root, ext := splitPathExt("/nested/spec.yaml"); root != "/nested/spec" || ext != ".yaml" {
		t.Fatalf("splitPathExt() = (%q, %q)", root, ext)
	}
	if got := normalizeVersionParam("v1.json"); got != "v1" {
		t.Fatalf("normalizeVersionParam() = %q", got)
	}
	if got := versionedDocsPattern(""); got != "" {
		t.Fatalf("versionedDocsPattern(empty) = %q", got)
	}
	if got := versionedOpenAPIPath("", "v2"); got != "" {
		t.Fatalf("versionedOpenAPIPath(empty) = %q", got)
	}

	docsCtx, _ := newTestContext(http.MethodGet, "/docs/v3", "")
	docsCtx.Params = gin.Params{{Key: "version", Value: "v3"}}
	if got := requestVersion(docsCtx); got != "v3" {
		t.Fatalf("requestVersion(docs) = %q", got)
	}

	openapiCtx, _ := newTestContext(http.MethodGet, "/openapi/v4.json", "")
	openapiCtx.Params = gin.Params{{Key: "version.json", Value: "v4.json"}}
	if got := requestVersion(openapiCtx); got != "v4" {
		t.Fatalf("requestVersion(openapi) = %q", got)
	}

	notFoundCtx, notFoundWriter := newTestContext(http.MethodGet, "/openapi/missing.json", "")
	versionNotFound(notFoundCtx)
	if notFoundWriter.Code != http.StatusNotFound || !strings.Contains(notFoundWriter.Body.String(), "API version not found") {
		t.Fatalf("unexpected versionNotFound response: %d %s", notFoundWriter.Code, notFoundWriter.Body.String())
	}
}

func TestVersionConfigHelpers(t *testing.T) {
	t.Parallel()

	cfg := normalizeVersionConfig("v1", VersionConfig{})
	if cfg.Prefix != "/v1" {
		t.Fatalf("normalizeVersionConfig() prefix = %q", cfg.Prefix)
	}

	cfg = normalizeVersionConfig("v2", VersionConfig{Prefix: "api/v2"})
	if cfg.Prefix != "/api/v2" {
		t.Fatalf("normalizeVersionConfig(custom) prefix = %q", cfg.Prefix)
	}

	spec := versionSpecConfig(Config{
		Title:       "Demo API",
		Description: "base",
	}, "v3", VersionConfig{Description: "versioned"})
	if spec.Title != "Demo API (v3)" || spec.Version != "v3" {
		t.Fatalf("versionSpecConfig() = %+v", spec)
	}
	if spec.Description != "base\n\nversioned" {
		t.Fatalf("versionSpecConfig() description = %q", spec.Description)
	}
	if got := joinDescription(" first ", "", "second"); got != "first\n\nsecond" {
		t.Fatalf("joinDescription() = %q", got)
	}
}

func TestVersionDeprecationMiddlewareHeaders(t *testing.T) {
	t.Parallel()

	cfg := VersionConfig{
		Deprecated:      true,
		DeprecatedSince: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		SunsetTime:      time.Date(2026, time.June, 7, 8, 9, 10, 0, time.UTC),
		MigrationURL:    "https://example.com/migrate",
	}

	r := gin.New()
	r.Use(versionDeprecationMiddleware(cfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Deprecation"); got != cfg.DeprecatedSince.Format(http.TimeFormat) {
		t.Fatalf("Deprecation header = %q", got)
	}
	if got := w.Header().Get("Sunset"); got != cfg.SunsetTime.Format(http.TimeFormat) {
		t.Fatalf("Sunset header = %q", got)
	}
	if got := w.Header().Get("Link"); !strings.Contains(got, cfg.MigrationURL) {
		t.Fatalf("Link header = %q", got)
	}
}

func TestCaptureResponseWriterWriteHeaderNowAndHelpers(t *testing.T) {
	t.Parallel()

	c, _ := newTestContext(http.MethodGet, "/cache", "")
	recorder := newCaptureResponseWriter(c.Writer)

	recorder.WriteHeaderNow()
	if recorder.Status() != http.StatusOK {
		t.Fatalf("Status() = %d, want %d", recorder.Status(), http.StatusOK)
	}
	if !recorder.Written() {
		t.Fatal("expected recorder to be marked written after WriteHeaderNow")
	}
	if _, err := recorder.WriteString("cached"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if recorder.Size() != len("cached") {
		t.Fatalf("Size() = %d, want %d", recorder.Size(), len("cached"))
	}
	if string(recorder.body) != "cached" {
		t.Fatalf("body = %q, want %q", string(recorder.body), "cached")
	}
	recorder.Flush()
}

func TestMemoryCacheStoreRemovesExpiredEntriesAndClonesValues(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStoreWithLimit(2)
	store.Set("expired", &CachedResponse{Status: http.StatusAccepted, Expires: time.Now().Add(-time.Second)})
	store.Set("fresh", &CachedResponse{
		Status: http.StatusCreated,
		Header: http.Header{"X-Test": []string{"value"}},
		Body:   []byte("ok"),
	})

	if _, ok := store.Get("expired"); ok {
		t.Fatal("expected expired cache entry to be removed")
	}
	if _, exists := store.items["expired"]; exists {
		t.Fatal("expected expired cache entry to be deleted from store")
	}

	value, ok := store.Get("fresh")
	if !ok {
		t.Fatal("expected fresh cache entry")
	}
	value.Header.Set("X-Test", "changed")
	value.Body[0] = 'X'

	again, ok := store.Get("fresh")
	if !ok {
		t.Fatal("expected fresh cache entry on second read")
	}
	if again.Header.Get("X-Test") != "value" {
		t.Fatalf("expected cached header clone, got %q", again.Header.Get("X-Test"))
	}
	if string(again.Body) != "ok" {
		t.Fatalf("expected cached body clone, got %q", string(again.Body))
	}
}

func TestMemoryCacheStoreDefaultsAndUpdatesExistingKeys(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStoreWithLimit(0)
	if store.maxEntries != defaultMemoryCacheMaxEntries {
		t.Fatalf("maxEntries = %d, want %d", store.maxEntries, defaultMemoryCacheMaxEntries)
	}

	store.Set("ignored", nil)
	if len(store.items) != 0 || len(store.order) != 0 {
		t.Fatalf("expected nil cache writes to be ignored, got items=%d order=%d", len(store.items), len(store.order))
	}

	store.Set("shared", &CachedResponse{Status: http.StatusAccepted, Body: []byte("first")})
	store.Set("shared", &CachedResponse{Status: http.StatusCreated, Body: []byte("second")})
	if len(store.order) != 1 {
		t.Fatalf("expected existing key updates not to duplicate order, got %v", store.order)
	}

	value, ok := store.Get("shared")
	if !ok {
		t.Fatal("expected updated cache entry")
	}
	if value.Status != http.StatusCreated || string(value.Body) != "second" {
		t.Fatalf("unexpected updated cache value: %+v", value)
	}
}

func TestOpenAPICacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	api := New(Config{
		Title:   "cache",
		Version: "1.0.0",
		Versions: map[string]VersionConfig{
			"v1": {Prefix: "/v1"},
		},
	})
	router := NewRouter("/items", WithVersion("v1"))
	Get(router, "/", func(ctx *Context, in *struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})
	api.AddRouter(router)

	const workers = 24
	start := make(chan struct{})
	errs := make(chan error, workers*2)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			got, err := api.openAPIBytes()
			if err != nil {
				errs <- err
				return
			}
			if len(got) == 0 {
				errs <- errors.New("empty main openapi bytes")
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			got, ok, err := api.versionOpenAPIBytes("v1")
			if err != nil {
				errs <- err
				return
			}
			if !ok {
				errs <- errors.New("expected version spec")
				return
			}
			if len(got) == 0 {
				errs <- errors.New("empty version openapi bytes")
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent openapi access failed: %v", err)
	}

	api.invalidateOpenAPICache()
	if got, err := api.openAPIBytes(); err != nil || len(got) == 0 {
		t.Fatalf("openAPIBytes() after invalidation = %q, %v", string(got), err)
	}
}

func TestMemoryCacheStoreConcurrentLockingAndBoundaryInputs(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	store.Set("users:1", &CachedResponse{Status: http.StatusOK, Expires: time.Now().Add(time.Minute)})
	store.AddTags("users:1", "", "users", " users ", "users")

	if removed := store.InvalidateTags("", " ", "users", "users"); removed != 1 {
		t.Fatalf("InvalidateTags() removed %d keys, want 1", removed)
	}
	if _, ok := store.Get("users:1"); ok {
		t.Fatal("expected tagged key to be deleted")
	}
	if _, ok := store.AcquireLock("   ", 0); ok {
		t.Fatal("expected blank cache key lock acquisition to fail")
	}
	if _, ok := store.AcquireLock("", 0); ok {
		t.Fatal("expected empty cache key lock acquisition to fail")
	}

	const contenders = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	var wins int32
	unlocks := make(chan func(), contenders)

	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			unlock, ok := store.AcquireLock("shared", 0)
			if !ok {
				return
			}
			atomic.AddInt32(&wins, 1)
			unlocks <- unlock
		}()
	}

	close(start)
	wg.Wait()
	close(unlocks)

	if got := atomic.LoadInt32(&wins); got != 1 {
		t.Fatalf("expected one lock winner, got %d", got)
	}

	unlock := <-unlocks
	unlock()
	if unlock, ok := store.AcquireLock("shared", 0); !ok || unlock == nil {
		t.Fatal("expected lock acquisition to succeed after releasing default-ttl lock")
	}
}

type hijackableResponseRecorder struct {
	*httptest.ResponseRecorder
}

func (w *hijackableResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	server, client := net.Pipe()
	reader := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
	return client, reader, nil
}

func TestCaptureResponseWriterHijackFallbackAndDelegate(t *testing.T) {
	t.Parallel()

	c, _ := newTestContext(http.MethodGet, "/cache", "")
	recorder := newCaptureResponseWriter(c.Writer)
	if _, _, err := recorder.Hijack(); err != http.ErrNotSupported {
		t.Fatalf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
	}

	base := &hijackableResponseRecorder{ResponseRecorder: httptest.NewRecorder()}
	c2, _ := gin.CreateTestContext(base)
	c2.Request = httptest.NewRequest(http.MethodGet, "/cache", nil)
	recorder = newCaptureResponseWriter(c2.Writer)
	conn, rw, err := recorder.Hijack()
	if err != nil || conn == nil || rw == nil {
		t.Fatalf("Hijack() = (%v, %v, %v)", conn, rw, err)
	}
	_ = conn.Close()
}
