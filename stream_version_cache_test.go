package ninja

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

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
	recorder := newCaptureResponseWriter(c.Writer, -1)

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

func TestMemoryCacheStoreDefaultsAndUpdatesExistingKeys(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStoreWithLimit(0)
	if store.maxEntries != defaultMemoryCacheMaxEntries {
		t.Fatalf("maxEntries = %d, want %d", store.maxEntries, defaultMemoryCacheMaxEntries)
	}

	store.Set("ignored", nil)
	if len(store.items) != 0 || store.order.Len() != 0 || len(store.entries) != 0 {
		t.Fatalf("expected nil cache writes to be ignored, got items=%d order=%d entries=%d", len(store.items), store.order.Len(), len(store.entries))
	}

	store.Set("shared", &CachedResponse{Status: http.StatusAccepted, Body: []byte("first")})
	store.Set("shared", &CachedResponse{Status: http.StatusCreated, Body: []byte("second")})
	if store.order.Len() != 1 || len(store.entries) != 1 {
		t.Fatalf("expected existing key updates not to duplicate order, got order=%d entries=%d", store.order.Len(), len(store.entries))
	}

	value, ok := store.Get("shared")
	if !ok {
		t.Fatal("expected updated cache entry")
	}
	if value.Status != http.StatusCreated || string(value.Body) != "second" {
		t.Fatalf("unexpected updated cache value: %+v", value)
	}
}

func TestMemoryCacheStoreLRUUsesConstantTimeBookkeeping(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStoreWithLimit(2)
	store.Set("old", &CachedResponse{Status: http.StatusOK, Body: []byte("old")})
	store.Set("hot", &CachedResponse{Status: http.StatusOK, Body: []byte("hot")})
	if _, ok := store.Get("old"); !ok {
		t.Fatal("expected old entry before eviction")
	}
	store.Set("new", &CachedResponse{Status: http.StatusOK, Body: []byte("new")})

	if _, ok := store.Get("hot"); ok {
		t.Fatal("expected least recently used key to be evicted")
	}
	for _, key := range []string{"old", "new"} {
		if _, ok := store.Get(key); !ok {
			t.Fatalf("expected %q to remain in cache", key)
		}
		if store.entries[key] == nil {
			t.Fatalf("expected %q to have an LRU list entry", key)
		}
	}
	if store.order.Len() != len(store.items) || len(store.entries) != len(store.items) {
		t.Fatalf("LRU indexes out of sync: order=%d entries=%d items=%d", store.order.Len(), len(store.entries), len(store.items))
	}
}

func TestWrapCacheStreamsAndSkipsOversizedResponses(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	op := &operation{
		method:       http.MethodGet,
		outputType:   reflect.TypeOf(struct{}{}),
		cache:        newRouteCacheConfig(time.Minute),
		cacheControl: defaultCacheControl(time.Minute),
		etagEnabled:  true,
	}
	op.cache.store = store
	op.cache.maxBodyBytes = 4

	c, w := newTestContext(http.MethodGet, "/large", "")
	handler := wrapCache(op, func(c *gin.Context) {
		c.Header("X-Test", "large")
		_, _ = c.Writer.Write([]byte("too-large"))
	})

	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Body.String(); got != "too-large" {
		t.Fatalf("body = %q, want too-large", got)
	}
	if got := w.Header().Get("X-Test"); got != "large" {
		t.Fatalf("X-Test header = %q, want large", got)
	}
	if etag := w.Header().Get("ETag"); etag != "" {
		t.Fatalf("expected oversized response to skip ETag generation, got %q", etag)
	}
	if _, ok := store.Get("GET:/large"); ok {
		t.Fatal("expected oversized response not to be cached")
	}
}

func TestMemoryCacheStoreEvictionCleansTagIndexes(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStoreWithLimit(1)
	store.Set("old", &CachedResponse{Status: http.StatusOK})
	store.AddTags("old", "users")
	store.Set("new", &CachedResponse{Status: http.StatusCreated})

	store.mu.RLock()
	defer store.mu.RUnlock()
	if _, ok := store.items["old"]; ok {
		t.Fatal("expected old cache entry to be evicted")
	}
	if _, ok := store.keyTags["old"]; ok {
		t.Fatal("expected evicted key tags to be removed")
	}
	if len(store.tags["users"]) != 0 {
		t.Fatalf("expected evicted tag index to be empty, got %+v", store.tags["users"])
	}
}

type contextAwareStore struct {
	ctx      context.Context
	response *CachedResponse
}

func (s *contextAwareStore) Get(key string) (*CachedResponse, bool) { return nil, false }
func (s *contextAwareStore) Set(key string, value *CachedResponse)  {}
func (s *contextAwareStore) GetContext(ctx context.Context, key string) (*CachedResponse, bool) {
	s.ctx = ctx
	return s.response, s.response != nil
}
func (s *contextAwareStore) SetContext(ctx context.Context, key string, value *CachedResponse) {
	s.ctx = ctx
	s.response = value
}

func TestCacheStoreHelpersPreferRequestContext(t *testing.T) {
	t.Parallel()

	c, _ := newTestContext(http.MethodGet, "/cache", "")
	reqCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c.Request = c.Request.WithContext(reqCtx)
	ctx := newContext(c)

	store := &contextAwareStore{response: &CachedResponse{Status: http.StatusAccepted}}
	if cached, ok := cacheStoreGet(ctx, store, "users:1"); !ok || cached.Status != http.StatusAccepted {
		t.Fatalf("cacheStoreGet() = (%+v, %v)", cached, ok)
	}
	if store.ctx != reqCtx {
		t.Fatal("expected cacheStoreGet to receive request context")
	}

	cacheStoreSet(ctx, store, "users:1", &CachedResponse{Status: http.StatusCreated})
	if store.ctx != reqCtx || store.response == nil || store.response.Status != http.StatusCreated {
		t.Fatalf("expected cacheStoreSet to receive request context and value, got ctx=%v response=%+v", store.ctx, store.response)
	}
}

func TestMemoryCacheStoreDeleteExpiredIfMatchLockedPreservesReplacedValue(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	store.Set("shared", &CachedResponse{Status: http.StatusAccepted, Expires: time.Now().Add(-time.Second), Body: []byte("stale")})

	store.mu.RLock()
	stale := store.items["shared"]
	store.mu.RUnlock()
	if stale == nil {
		t.Fatal("expected stale cache entry")
	}

	freshExpiry := time.Now().Add(time.Minute)
	store.Set("shared", &CachedResponse{Status: http.StatusCreated, Expires: freshExpiry, Body: []byte("fresh")})

	store.mu.Lock()
	store.deleteExpiredIfMatchLocked("shared", stale, time.Now())
	store.mu.Unlock()

	value, ok := store.Get("shared")
	if !ok {
		t.Fatal("expected replaced cache entry to remain available")
	}
	if value.Status != http.StatusCreated || string(value.Body) != "fresh" {
		t.Fatalf("unexpected cache value after guarded delete: %+v", value)
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
	recorder := newCaptureResponseWriter(c.Writer, -1)
	if _, _, err := recorder.Hijack(); err != http.ErrNotSupported {
		t.Fatalf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
	}

	base := &hijackableResponseRecorder{ResponseRecorder: httptest.NewRecorder()}
	c2, _ := gin.CreateTestContext(base)
	c2.Request = httptest.NewRequest(http.MethodGet, "/cache", nil)
	recorder = newCaptureResponseWriter(c2.Writer, -1)
	conn, rw, err := recorder.Hijack()
	if err != nil || conn == nil || rw == nil {
		t.Fatalf("Hijack() = (%v, %v, %v)", conn, rw, err)
	}
	_ = conn.Close()
}
