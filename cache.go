package ninja

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
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

// ResponseCacheDeleteStore optionally supports cache-key invalidation.
type ResponseCacheDeleteStore interface {
	Delete(key string)
	DeleteMany(keys ...string)
}

// ResponseCacheTagStore optionally supports assigning tags to keys and invalidating by tag.
type ResponseCacheTagStore interface {
	AddTags(key string, tags ...string)
	InvalidateTags(tags ...string) int
}

// ResponseCacheLockStore optionally supports short-lived distributed or local locks.
type ResponseCacheLockStore interface {
	AcquireLock(key string, ttl time.Duration) (unlock func(), ok bool)
}

type CacheKeyFunc func(*Context) string
type CacheTagFunc func(*Context) []string

type CacheInvalidator struct {
	store ResponseCacheStore
}

type routeCacheConfig struct {
	ttl   time.Duration
	store ResponseCacheStore
	keyFn CacheKeyFunc
	tagFn CacheTagFunc
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
	tags       map[string]map[string]struct{}
	keyTags    map[string]map[string]struct{}
	locks      map[string]memoryCacheLock
	maxEntries int
	lockSeq    uint64
}

const defaultMemoryCacheMaxEntries = 1024

type memoryCacheLock struct {
	token   uint64
	expires time.Time
}

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

// CacheWithTags assigns one or more tags to stored responses so they can be invalidated later.
func CacheWithTags(fn CacheTagFunc) CacheOption {
	return func(cfg *routeCacheConfig) {
		if fn != nil {
			cfg.tagFn = fn
		}
	}
}

// NewCacheInvalidator provides a unified invalidation entry point for any cache store.
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
	normalized := normalizeCacheTags(keys)
	if len(normalized) == 0 {
		return 0
	}
	store.DeleteMany(normalized...)
	return len(normalized)
}

// Tag associates one cache key with one or more invalidation tags.
func (i *CacheInvalidator) Tag(key string, tags ...string) bool {
	if i == nil || i.store == nil || strings.TrimSpace(key) == "" {
		return false
	}
	store, ok := i.store.(ResponseCacheTagStore)
	if !ok {
		return false
	}
	store.AddTags(key, tags...)
	return true
}

// InvalidateTags removes all keys currently associated with the provided tags.
func (i *CacheInvalidator) InvalidateTags(tags ...string) int {
	if i == nil || i.store == nil {
		return 0
	}
	store, ok := i.store.(ResponseCacheTagStore)
	if !ok {
		return 0
	}
	return store.InvalidateTags(tags...)
}

// AcquireLock tries to obtain a short-lived lock for the given cache key.
func (i *CacheInvalidator) AcquireLock(key string, ttl time.Duration) (func(), bool) {
	if i == nil || i.store == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	store, ok := i.store.(ResponseCacheLockStore)
	if !ok {
		return nil, false
	}
	return store.AcquireLock(key, ttl)
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
		tags:       map[string]map[string]struct{}{},
		keyTags:    map[string]map[string]struct{}{},
		locks:      map[string]memoryCacheLock{},
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

func (s *MemoryCacheStore) Delete(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	s.mu.Lock()
	s.deleteKeyLocked(key)
	s.mu.Unlock()
}

func (s *MemoryCacheStore) DeleteMany(keys ...string) {
	normalized := normalizeCacheTags(keys)
	if len(normalized) == 0 {
		return
	}
	s.mu.Lock()
	for _, key := range normalized {
		s.deleteKeyLocked(key)
	}
	s.mu.Unlock()
}

func (s *MemoryCacheStore) AddTags(key string, tags ...string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	normalized := normalizeCacheTags(tags)
	if len(normalized) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[key]; !ok {
		return
	}
	for _, tag := range normalized {
		if s.tags[tag] == nil {
			s.tags[tag] = map[string]struct{}{}
		}
		s.tags[tag][key] = struct{}{}
		if s.keyTags[key] == nil {
			s.keyTags[key] = map[string]struct{}{}
		}
		s.keyTags[key][tag] = struct{}{}
	}
}

func (s *MemoryCacheStore) InvalidateTags(tags ...string) int {
	normalized := normalizeCacheTags(tags)
	if len(normalized) == 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := map[string]struct{}{}
	for _, tag := range normalized {
		for key := range s.tags[tag] {
			keys[key] = struct{}{}
		}
		delete(s.tags, tag)
	}
	for key := range keys {
		s.deleteKeyLocked(key)
	}
	return len(keys)
}

func (s *MemoryCacheStore) AcquireLock(key string, ttl time.Duration) (func(), bool) {
	if strings.TrimSpace(key) == "" {
		return nil, false
	}
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	now := time.Now()
	token := atomic.AddUint64(&s.lockSeq, 1)

	s.mu.Lock()
	if existing, ok := s.locks[key]; ok && now.Before(existing.expires) {
		s.mu.Unlock()
		return nil, false
	}
	s.locks[key] = memoryCacheLock{
		token:   token,
		expires: now.Add(ttl),
	}
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		if existing, ok := s.locks[key]; ok && existing.token == token {
			delete(s.locks, key)
		}
		s.mu.Unlock()
	}, true
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
	s.deleteKeyTagsLocked(key)
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

func (s *MemoryCacheStore) deleteKeyTagsLocked(key string) {
	tags := s.keyTags[key]
	delete(s.keyTags, key)
	for tag := range tags {
		keys := s.tags[tag]
		delete(keys, key)
		if len(keys) == 0 {
			delete(s.tags, tag)
		}
	}
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
				if !isExpiredCachedResponse(cached, time.Now()) {
					writeCachedResponse(c, cached, op.cacheControl)
					return
				}
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
			if tagStore, ok := cacheStore.(ResponseCacheTagStore); ok && op.cache.tagFn != nil {
				tagStore.AddTags(cacheKey, op.cache.tagFn(ctx)...)
			}
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

func normalizeCacheTags(values []string) []string {
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
