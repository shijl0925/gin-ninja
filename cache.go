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
type ResponseCacheStore interface {
	Get(key string) (*cachedResponse, bool)
	Set(key string, value *cachedResponse)
}

type CacheKeyFunc func(*Context) string

type routeCacheConfig struct {
	ttl    time.Duration
	store  ResponseCacheStore
	keyFn  CacheKeyFunc
}

type cachedResponse struct {
	status  int
	header  http.Header
	body    []byte
	expires time.Time
	etag    string
}

type MemoryCacheStore struct {
	mu    sync.RWMutex
	items map[string]*cachedResponse
}

var defaultResponseCacheStore = NewMemoryCacheStore()

func newRouteCacheConfig(ttl time.Duration) *routeCacheConfig {
	return &routeCacheConfig{
		ttl:   ttl,
		store: defaultResponseCacheStore,
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
	return &MemoryCacheStore{items: map[string]*cachedResponse{}}
}

func (s *MemoryCacheStore) Get(key string) (*cachedResponse, bool) {
	s.mu.RLock()
	value, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !value.expires.IsZero() && time.Now().After(value.expires) {
		s.mu.Lock()
		delete(s.items, key)
		s.mu.Unlock()
		return nil, false
	}
	return cloneCachedResponse(value), true
}

func (s *MemoryCacheStore) Set(key string, value *cachedResponse) {
	if value == nil {
		return
	}
	s.mu.Lock()
	s.items[key] = cloneCachedResponse(value)
	s.mu.Unlock()
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

		if cacheStore != nil && cacheKey != "" && recorder.status >= 200 && recorder.status < 300 {
			cacheStore.Set(cacheKey, &cachedResponse{
				status:  recorder.status,
				header:  cloneHeader(recorder.header),
				body:    append([]byte(nil), recorder.body...),
				expires: time.Now().Add(op.cache.ttl),
				etag:    etag,
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

func writeCachedResponse(c *gin.Context, cached *cachedResponse, cacheControl string) {
	if cached == nil {
		c.Status(http.StatusNoContent)
		return
	}
	header := cloneHeader(cached.header)
	if cacheControl != "" && header.Get("Cache-Control") == "" {
		header.Set("Cache-Control", cacheControl)
	}
	if etag := header.Get("ETag"); etag != "" && matchesETag(c.GetHeader("If-None-Match"), etag) {
		copyHeader(c.Writer.Header(), header)
		c.Status(http.StatusNotModified)
		return
	}
	copyHeader(c.Writer.Header(), header)
	c.Status(cached.status)
	if len(cached.body) > 0 && c.Request.Method != http.MethodHead {
		_, _ = c.Writer.Write(cached.body)
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

func cloneCachedResponse(in *cachedResponse) *cachedResponse {
	if in == nil {
		return nil
	}
	return &cachedResponse{
		status:  in.status,
		header:  cloneHeader(in.header),
		body:    append([]byte(nil), in.body...),
		expires: in.expires,
		etag:    in.etag,
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

func (w *captureResponseWriter) WriteHeaderNow() {}

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

func (w *captureResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}
