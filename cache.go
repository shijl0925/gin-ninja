package ninja

import (
	"bufio"
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/defaults"
)

type CacheOption func(*routeCacheConfig)

// ResponseCacheStore stores serialized route responses for cacheable endpoints.
// Implementations receive fully-exported CachedResponse values so applications
// can provide a small custom store when the built-in memory store is not enough.
type ResponseCacheStore interface {
	Get(key string) (*CachedResponse, bool)
	Set(key string, value *CachedResponse)
}

// ResponseCacheDeleteStore optionally supports cache-key invalidation.
type ResponseCacheDeleteStore interface {
	Delete(key string)
	DeleteMany(keys ...string)
}

type CacheKeyFunc func(*Context) string

var defaultCacheVaryHeaders = []string{"Authorization", "Accept-Language"}

type CacheInvalidator struct {
	store ResponseCacheStore
}

type routeCacheConfig struct {
	ttl          time.Duration
	store        ResponseCacheStore
	keyFn        CacheKeyFunc
	maxBodyBytes int64
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
	order      *list.List
	entries    map[string]*list.Element
	maxEntries int
}

type memoryCacheEntry struct {
	key string
}

func newRouteCacheConfig(ttl time.Duration) *routeCacheConfig {
	return &routeCacheConfig{
		ttl:          ttl,
		store:        NewMemoryCacheStore(),
		keyFn:        defaultCacheKey,
		maxBodyBytes: defaults.CacheMaxBodyBytes,
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

// CacheWithMaxBodyBytes limits how much response body data a cached route will
// buffer for ETag generation and cache storage. Responses larger than max are
// streamed to the client and are not cached. Use a negative value to disable the
// limit for trusted small-response routes.
func CacheWithMaxBodyBytes(max int64) CacheOption {
	return func(cfg *routeCacheConfig) {
		cfg.maxBodyBytes = max
	}
}

// NewCacheInvalidator provides explicit cache-key invalidation for stores that
// implement ResponseCacheDeleteStore.
func NewCacheInvalidator(store ResponseCacheStore) *CacheInvalidator {
	return &CacheInvalidator{store: store}
}

// Delete removes one or more cached keys when the underlying store supports invalidation.
func (i *CacheInvalidator) Delete(keys ...string) int {
	if i == nil || i.store == nil || len(keys) == 0 {
		return 0
	}
	store, ok := i.store.(ResponseCacheDeleteStore)
	if !ok {
		return 0
	}
	normalized := normalizeCacheKeys(keys)
	if len(normalized) == 0 {
		return 0
	}
	store.DeleteMany(normalized...)
	return len(normalized)
}

// NewMemoryCacheStore creates an in-memory route cache store.
func NewMemoryCacheStore() *MemoryCacheStore {
	return NewMemoryCacheStoreWithLimit(defaults.MemoryCacheMaxEntries)
}

// NewMemoryCacheStoreWithLimit creates an in-memory route cache store with a bounded size.
func NewMemoryCacheStoreWithLimit(maxEntries int) *MemoryCacheStore {
	if maxEntries <= 0 {
		maxEntries = defaults.MemoryCacheMaxEntries
	}
	return &MemoryCacheStore{
		items:      map[string]*CachedResponse{},
		order:      list.New(),
		entries:    map[string]*list.Element{},
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
	now := time.Now()
	if !value.Expires.IsZero() && now.After(value.Expires) {
		s.mu.Lock()
		s.deleteExpiredIfMatchLocked(key, value, now)
		s.mu.Unlock()
		return nil, false
	}
	// Promote the key to the back of the eviction order so that recently
	// accessed entries are evicted last (LRU semantics).
	s.mu.Lock()
	s.promoteKeyLocked(key)
	s.mu.Unlock()
	return cloneCachedResponse(value), true
}

func (s *MemoryCacheStore) deleteExpiredIfMatchLocked(key string, expected *CachedResponse, now time.Time) {
	current, ok := s.items[key]
	if !ok || current != expected {
		return
	}
	if current.Expires.IsZero() || !now.After(current.Expires) {
		return
	}
	s.deleteKeyLocked(key)
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
		s.entries[key] = s.order.PushBack(memoryCacheEntry{key: key})
	} else {
		s.promoteKeyLocked(key)
	}
	s.items[key] = cloneCachedResponse(value)
	s.mu.Unlock()
}

func (s *MemoryCacheStore) Delete(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	s.mu.Lock()
	s.deleteKeyLocked(key)
	s.mu.Unlock()
}

func (s *MemoryCacheStore) DeleteMany(keys ...string) {
	normalized := normalizeCacheKeys(keys)
	if len(normalized) == 0 {
		return
	}
	s.mu.Lock()
	for _, key := range normalized {
		s.deleteKeyLocked(key)
	}
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
	for s.order.Len() > 0 {
		element := s.order.Front()
		entry, _ := element.Value.(memoryCacheEntry)
		key := entry.key
		s.order.Remove(element)
		delete(s.entries, key)
		if _, ok := s.items[key]; ok {
			s.deleteKeyLocked(key)
			return
		}
	}
}

func (s *MemoryCacheStore) deleteKeyLocked(key string) {
	delete(s.items, key)
	if element := s.entries[key]; element != nil {
		s.order.Remove(element)
		delete(s.entries, key)
	}
}

// promoteKeyLocked moves key to the back of s.order so that eviction targets
// the least-recently-used entry.
// Must be called with s.mu held for writing.
func (s *MemoryCacheStore) promoteKeyLocked(key string) {
	if element := s.entries[key]; element != nil {
		s.order.MoveToBack(element)
	}
}

func wrapCache(op *operation, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCacheableMethod(c.Request.Method) || op.stream.config != nil {
			next(c)
			return
		}

		ctx := newContext(c)
		cacheKey, cacheStore := cacheLookup(op, ctx)
		if cacheStore != nil && cacheKey != "" {
			if cached, ok := cacheStoreGet(cacheStore, cacheKey); ok {
				if !isExpiredCachedResponse(cached, time.Now()) {
					writeCachedResponse(c, cached, op.cache.control, defaultCacheVaryHeaders...)
					return
				}
			}
		}

		if isDownloadType(op.route.outputType) {
			next(c)
			return
		}

		maxBodyBytes := defaults.CacheMaxBodyBytes
		if op.cache.config != nil {
			maxBodyBytes = op.cache.config.maxBodyBytes
		}
		originalWriter := c.Writer
		recorder := newCaptureResponseWriter(originalWriter, maxBodyBytes)
		c.Writer = recorder
		next(c)
		c.Writer = originalWriter
		if recorder.passthrough {
			return
		}

		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		if op.cache.control != "" && recorder.status >= 200 && recorder.status < 400 && recorder.header.Get("Cache-Control") == "" {
			recorder.header.Set("Cache-Control", op.cache.control)
		}
		if op.cache.config != nil && recorder.status >= 200 && recorder.status < 400 {
			addVary(recorder.header, defaultCacheVaryHeaders...)
		}

		etag := recorder.header.Get("ETag")
		if op.cache.etagEnabled && etag == "" && recorder.status >= 200 && recorder.status < 400 && len(recorder.body) > 0 {
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

		if cacheStore != nil && cacheKey != "" && op.cache.config != nil && recorder.status >= 200 && recorder.status < 300 {
			cacheStoreSet(cacheStore, cacheKey, &CachedResponse{
				Status:  recorder.status,
				Header:  cloneHeader(recorder.header),
				Body:    append([]byte(nil), recorder.body...),
				Expires: time.Now().Add(op.cache.config.ttl),
				ETag:    etag,
			})
		}
	}
}

func cacheLookup(op *operation, ctx *Context) (string, ResponseCacheStore) {
	if op.cache.config == nil || op.cache.config.ttl <= 0 {
		return "", nil
	}
	keyFn := op.cache.config.keyFn
	if keyFn == nil {
		keyFn = defaultCacheKey
	}
	return keyFn(ctx), op.cache.config.store
}

func cacheStoreGet(store ResponseCacheStore, key string) (*CachedResponse, bool) {
	if store == nil {
		return nil, false
	}
	return store.Get(key)
}

func cacheStoreSet(store ResponseCacheStore, key string, value *CachedResponse) {
	if store == nil {
		return
	}
	store.Set(key, value)
}

func writeCachedResponse(c *gin.Context, cached *CachedResponse, cacheControl string, vary ...string) {
	if cached == nil {
		c.Status(http.StatusNoContent)
		return
	}
	header := cloneHeader(cached.Header)
	if cacheControl != "" && header.Get("Cache-Control") == "" {
		header.Set("Cache-Control", cacheControl)
	}
	addVary(header, vary...)
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
	return fmt.Sprintf("private, max-age=%d", seconds)
}

func defaultCacheKey(ctx *Context) string {
	if ctx == nil || ctx.Request == nil || ctx.Request.URL == nil {
		return ""
	}
	key := ctx.Request.Method + ":" + ctx.Request.URL.RequestURI()
	for _, header := range defaultCacheVaryHeaders {
		if value := ctx.Request.Header.Get(header); value != "" {
			key += "|" + http.CanonicalHeaderKey(header) + "=" + hashCacheKeyValue(value)
		}
	}
	return key
}

func hashCacheKeyValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func addVary(header http.Header, fields ...string) {
	if header == nil {
		return
	}
	values := splitCommaValues(header.Get("Vary"))
	for _, value := range values {
		if value == "*" {
			return
		}
	}
	for _, field := range fields {
		if field == "" {
			continue
		}
		exists := false
		for _, value := range values {
			if strings.EqualFold(value, field) {
				exists = true
				break
			}
		}
		if !exists {
			values = append(values, http.CanonicalHeaderKey(field))
		}
	}
	if len(values) > 0 {
		header.Set("Vary", strings.Join(values, ", "))
	}
}

func generateETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func matchesETag(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	normalizedETag := normalizeWeakETag(etag)
	for _, candidate := range splitCommaValues(ifNoneMatch) {
		if candidate == "*" || normalizeWeakETag(candidate) == normalizedETag {
			return true
		}
	}
	return false
}

// normalizeWeakETag strips the weak validator prefix so GET/HEAD conditional
// requests use weak comparison semantics when matching ETags.
func normalizeWeakETag(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.EqualFold(value[:2], "W/") {
		return value[2:]
	}
	return value
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

func normalizeCacheKeys(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
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

// isExpiredCachedResponse reports whether a cached response has a non-zero
// expiry time that is already in the past; nil entries and zero expiries are
// treated as not expired so callers can safely skip extra nil checks.
func isExpiredCachedResponse(value *CachedResponse, now time.Time) bool {
	return value != nil && !value.Expires.IsZero() && now.After(value.Expires)
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
	header       http.Header
	body         []byte
	status       int
	maxBodyBytes int64
	passthrough  bool
}

func newCaptureResponseWriter(base gin.ResponseWriter, maxBodyBytes int64) *captureResponseWriter {
	return &captureResponseWriter{
		ResponseWriter: base,
		header:         http.Header{},
		maxBodyBytes:   maxBodyBytes,
	}
}

func (w *captureResponseWriter) Header() http.Header {
	if w.passthrough {
		return w.ResponseWriter.Header()
	}
	return w.header
}

func (w *captureResponseWriter) WriteHeader(statusCode int) {
	if w.passthrough {
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}
	if w.status == 0 {
		w.status = statusCode
	}
}

func (w *captureResponseWriter) WriteHeaderNow() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.passthrough {
		w.ResponseWriter.WriteHeaderNow()
	}
}
func (w *captureResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.passthrough {
		return w.ResponseWriter.Write(data)
	}
	if w.maxBodyBytes >= 0 && int64(len(w.body))+int64(len(data)) > w.maxBodyBytes {
		return w.switchToPassthrough(data)
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
	if w.passthrough {
		return w.ResponseWriter.Size()
	}
	return len(w.body)
}

func (w *captureResponseWriter) Written() bool {
	return w.passthrough || w.status != 0 || len(w.body) > 0
}

func (w *captureResponseWriter) Flush() {
	if !w.passthrough {
		_, _ = w.switchToPassthrough(nil)
	}
	w.ResponseWriter.Flush()
}

func (w *captureResponseWriter) switchToPassthrough(data []byte) (int, error) {
	if w.passthrough {
		return w.ResponseWriter.Write(data)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	copyHeader(w.ResponseWriter.Header(), w.header)
	w.ResponseWriter.WriteHeader(w.status)
	w.passthrough = true
	if len(w.body) > 0 {
		if _, err := w.ResponseWriter.Write(w.body); err != nil {
			w.body = nil
			return 0, err
		}
		w.body = nil
	}
	if len(data) == 0 {
		return 0, nil
	}
	return w.ResponseWriter.Write(data)
}

func (w *captureResponseWriter) Hijack() (conn net.Conn, rw *bufio.ReadWriter, err error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	if unwrapper, ok := w.ResponseWriter.(interface{ Unwrap() http.ResponseWriter }); ok &&
		isGinResponseWriter(w.ResponseWriter) &&
		!supportsHijacker(unwrapper.Unwrap()) {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func supportsHijacker(writer any) bool {
	_, ok := writer.(http.Hijacker)
	return ok
}

func isGinResponseWriter(writer any) bool {
	typ := reflect.TypeOf(writer)
	if typ == nil {
		return false
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.PkgPath() == "github.com/gin-gonic/gin" && typ.Name() == "responseWriter"
}
