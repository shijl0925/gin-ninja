package ninja

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CacheOption func(*routeCacheConfig)

// ResponseCacheStore stores serialized route responses for cacheable endpoints.
// Implementations receive fully-exported CachedResponse values and may store
// them in any backend (in-process memory, Redis, Memcached, etc.).
type ResponseCacheStore interface {
	Get(key string) (*CachedResponse, bool)
	Set(key string, value *CachedResponse)
}

type CacheKeyFunc func(*Context) string

type routeCacheConfig struct {
	ttl   time.Duration
	store ResponseCacheStore
	keyFn CacheKeyFunc
}

// CachedResponse is the serialized representation of a cached HTTP response.
// All fields are exported so that external ResponseCacheStore implementations
// can read and write them without relying on internal package types.
type CachedResponse struct {
	Status  int
	Header  http.Header
	Body    []byte
	Expires time.Time
	ETag    string
}

type MemoryCacheStore struct {
	mu         sync.RWMutex
	items      map[string]*CachedResponse
	order      []string
	maxEntries int
}

const defaultMemoryCacheMaxEntries = 1024

func newRouteCacheConfig(ttl time.Duration) *routeCacheConfig {
	return &routeCacheConfig{
		ttl:   ttl,
		store: NewMemoryCacheStore(),
		keyFn: defaultCacheKey,
	}
}

// CacheWithStore overrides the cache backend for a route.
func CacheWithStore(store ResponseCacheStore) CacheOption {
	return func(cfg *routeCacheConfig) {
		if store != nil {
			cfg.store = store
		}
	}
}

// CacheWithKey customizes the cache key for a route.
func CacheWithKey(fn CacheKeyFunc) CacheOption {
	return func(cfg *routeCacheConfig) {
		if fn != nil {
			cfg.keyFn = fn
		}
	}
}

// NewMemoryCacheStore creates an in-memory route cache store.
func NewMemoryCacheStore() *MemoryCacheStore {
	return NewMemoryCacheStoreWithLimit(defaultMemoryCacheMaxEntries)
}

// NewMemoryCacheStoreWithLimit creates an in-memory route cache store with a bounded size.
func NewMemoryCacheStoreWithLimit(maxEntries int) *MemoryCacheStore {
	if maxEntries <= 0 {
		maxEntries = defaultMemoryCacheMaxEntries
	}
	return &MemoryCacheStore{
		items:      map[string]*CachedResponse{},
		order:      []string{},
		maxEntries: maxEntries,
	}
}

func (s *MemoryCacheStore) Get(key string) (*CachedResponse, bool) {
	s.mu.RLock()
	value, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !value.Expires.IsZero() && time.Now().After(value.Expires) {
		s.mu.Lock()
		s.deleteKeyLocked(key)
		s.mu.Unlock()
		return nil, false
	}
	return cloneCachedResponse(value), true
}

func (s *MemoryCacheStore) Set(key string, value *CachedResponse) {
	if value == nil {
		return
	}
	s.mu.Lock()
	if _, exists := s.items[key]; !exists {
		s.pruneExpiredLocked(time.Now())
		if len(s.items) >= s.maxEntries {
			s.evictOldestLocked()
		}
		s.order = append(s.order, key)
	}
	s.items[key] = cloneCachedResponse(value)
	s.mu.Unlock()
}

func (s *MemoryCacheStore) pruneExpiredLocked(now time.Time) {
	for key, value := range s.items {
		if value != nil && !value.Expires.IsZero() && now.After(value.Expires) {
			s.deleteKeyLocked(key)
		}
	}
}

func (s *MemoryCacheStore) evictOldestLocked() {
	for len(s.order) > 0 {
		key := s.order[0]
		s.order = s.order[1:]
		if _, ok := s.items[key]; ok {
			delete(s.items, key)
			return
		}
	}
}

func (s *MemoryCacheStore) deleteKeyLocked(key string) {
	delete(s.items, key)
	if len(s.order) == 0 {
		return
	}
	filtered := s.order[:0]
	for _, existing := range s.order {
		if existing != key {
			filtered = append(filtered, existing)
		}
	}
	s.order = filtered
}

func wrapCache(op *operation, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCacheableMethod(c.Request.Method) || op.stream != nil {
			next(c)
			return
		}

		ctx := newContext(c)
		cacheKey, cacheStore := cacheLookup(op, ctx)
		if cacheStore != nil && cacheKey != "" {
			if cached, ok := cacheStore.Get(cacheKey); ok {
				writeCachedResponse(c, cached, op.cacheControl)
				return
			}
		}

		originalWriter := c.Writer
		recorder := newCaptureResponseWriter(originalWriter)
		c.Writer = recorder
		next(c)
		c.Writer = originalWriter

		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		if op.cacheControl != "" && recorder.status >= 200 && recorder.status < 400 && recorder.header.Get("Cache-Control") == "" {
			recorder.header.Set("Cache-Control", op.cacheControl)
		}

		etag := recorder.header.Get("ETag")
		if op.etagEnabled && etag == "" && recorder.status >= 200 && recorder.status < 400 && len(recorder.body) > 0 {
			etag = generateETag(recorder.body)
			recorder.header.Set("ETag", etag)
		}

		if etag != "" && matchesETag(c.GetHeader("If-None-Match"), etag) {
			copyHeader(originalWriter.Header(), recorder.header)
			originalWriter.WriteHeader(http.StatusNotModified)
			return
		}

		copyHeader(originalWriter.Header(), recorder.header)
		originalWriter.WriteHeader(recorder.status)
		if len(recorder.body) > 0 && c.Request.Method != http.MethodHead {
			_, _ = originalWriter.Write(recorder.body)
		}

		if cacheStore != nil && cacheKey != "" && op.cache != nil && recorder.status >= 200 && recorder.status < 300 {
			cacheStore.Set(cacheKey, &CachedResponse{
				Status:  recorder.status,
				Header:  cloneHeader(recorder.header),
				Body:    append([]byte(nil), recorder.body...),
				Expires: time.Now().Add(op.cache.ttl),
				ETag:    etag,
			})
		}
	}
}

func cacheLookup(op *operation, ctx *Context) (string, ResponseCacheStore) {
	if op.cache == nil || op.cache.ttl <= 0 {
		return "", nil
	}
	keyFn := op.cache.keyFn
	if keyFn == nil {
		keyFn = defaultCacheKey
	}
	return keyFn(ctx), op.cache.store
}

func writeCachedResponse(c *gin.Context, cached *CachedResponse, cacheControl string) {
	if cached == nil {
		c.Status(http.StatusNoContent)
		return
	}
	header := cloneHeader(cached.Header)
	if cacheControl != "" && header.Get("Cache-Control") == "" {
		header.Set("Cache-Control", cacheControl)
	}
	if etag := header.Get("ETag"); etag != "" && matchesETag(c.GetHeader("If-None-Match"), etag) {
		copyHeader(c.Writer.Header(), header)
		c.Status(http.StatusNotModified)
		return
	}
	copyHeader(c.Writer.Header(), header)
	c.Status(cached.Status)
	if len(cached.Body) > 0 && c.Request.Method != http.MethodHead {
		_, _ = c.Writer.Write(cached.Body)
	}
}

func defaultCacheControl(ttl time.Duration) string {
	seconds := int(ttl / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("public, max-age=%d", seconds)
}

func defaultCacheKey(ctx *Context) string {
	if ctx == nil || ctx.Request == nil || ctx.Request.URL == nil {
		return ""
	}
	return ctx.Request.Method + ":" + ctx.Request.URL.RequestURI()
}

func generateETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func matchesETag(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	for _, candidate := range splitCommaValues(ifNoneMatch) {
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

func splitCommaValues(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isCacheableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func cloneCachedResponse(in *CachedResponse) *CachedResponse {
	if in == nil {
		return nil
	}
	return &CachedResponse{
		Status:  in.Status,
		Header:  cloneHeader(in.Header),
		Body:    append([]byte(nil), in.Body...),
		Expires: in.Expires,
		ETag:    in.ETag,
	}
}

func cloneHeader(in http.Header) http.Header {
	if len(in) == 0 {
		return http.Header{}
	}
	out := make(http.Header, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func copyHeader(dst, src http.Header) {
	for key := range dst {
		delete(dst, key)
	}
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
}

type captureResponseWriter struct {
	gin.ResponseWriter
	header http.Header
	body   []byte
	status int
}

func newCaptureResponseWriter(base gin.ResponseWriter) *captureResponseWriter {
	return &captureResponseWriter{
		ResponseWriter: base,
		header:         http.Header{},
	}
}

func (w *captureResponseWriter) Header() http.Header {
	return w.header
}

func (w *captureResponseWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
}

func (w *captureResponseWriter) WriteHeaderNow() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
}
func (w *captureResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *captureResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *captureResponseWriter) Status() int {
	return w.status
}

func (w *captureResponseWriter) Size() int {
	return len(w.body)
}

func (w *captureResponseWriter) Written() bool {
	return w.status != 0 || len(w.body) > 0
}

func (w *captureResponseWriter) Flush() {}

func (w *captureResponseWriter) Hijack() (conn net.Conn, rw *bufio.ReadWriter, err error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	defer func() {
		if recover() != nil {
			conn = nil
			rw = nil
			err = http.ErrNotSupported
		}
	}()
	return hijacker.Hijack()
}
