package ninja

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
)

func TestCacheInvalidatorAndCaptureWriterAdditionalCoverage(t *testing.T) {
	t.Run("invalidator delete and tag", func(t *testing.T) {
		store := NewMemoryCacheStore()
		store.Set("users:1", &CachedResponse{Status: http.StatusOK, Body: []byte("cached")})

		invalidator := NewCacheInvalidator(store)
		if removed := invalidator.Delete("", "users:1", "users:1"); removed != 1 {
			t.Fatalf("Delete() removed %d, want 1", removed)
		}
		if _, ok := store.Get("users:1"); ok {
			t.Fatal("expected deleted cache entry to be removed")
		}

		store.Set("users:2", &CachedResponse{Status: http.StatusOK, Body: []byte("cached")})
		if !invalidator.Tag("users:2", " users ", "", "users") {
			t.Fatal("expected Tag() to succeed")
		}
		if removed := invalidator.InvalidateTags("users"); removed != 1 {
			t.Fatalf("InvalidateTags() removed %d, want 1", removed)
		}
		if _, ok := store.Get("users:2"); ok {
			t.Fatal("expected tagged cache entry to be invalidated")
		}
	})

	t.Run("nil invalidator and capture writer", func(t *testing.T) {
		var invalidator *CacheInvalidator
		if got := invalidator.Delete("users:1"); got != 0 {
			t.Fatalf("Delete() = %d, want 0", got)
		}
		if invalidator.Tag("users:1", "users") {
			t.Fatal("expected Tag() to fail for nil invalidator")
		}

		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		writer := newCaptureResponseWriter(ctx.Writer, -1)
		writer.Flush()
		if _, _, err := writer.Hijack(); !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
		}
	})
}

func TestRedisCacheStoreAdditionalCoverage(t *testing.T) {
	t.Run("nil store guards", func(t *testing.T) {
		var store *RedisCacheStore
		if got := store.Client(); got != nil {
			t.Fatalf("Client() = %v, want nil", got)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
		if err := store.Ping(context.Background()); err == nil {
			t.Fatal("expected Ping() to fail for nil store")
		}
		if value, ok := store.Get("key"); ok || value != nil {
			t.Fatalf("Get() = (%v, %v), want (nil, false)", value, ok)
		}
		store.Set("key", &CachedResponse{})
		store.Delete("")
		store.DeleteMany("key")
		store.AddTags("key", "users")
		if removed := store.InvalidateTags("users"); removed != 0 {
			t.Fatalf("InvalidateTags() = %d, want 0", removed)
		}
		if unlock, ok := store.AcquireLock("key", time.Second); ok || unlock != nil {
			t.Fatalf("AcquireLock() unexpected result: unlock nil=%t ok=%v", unlock == nil, ok)
		}
	})

	t.Run("constructor and lifecycle", func(t *testing.T) {
		if _, err := NewRedisCacheStore(RedisCacheConfig{}); err == nil {
			t.Fatal("expected missing addr error")
		}

		redisServer := miniredis.RunT(t)
		store, err := NewRedisCacheStore(RedisCacheConfig{Addr: redisServer.Addr(), Prefix: " "})
		if err != nil {
			t.Fatalf("NewRedisCacheStore: %v", err)
		}
		if store.Client() == nil {
			t.Fatal("expected redis client")
		}
		if err := store.Ping(context.Background()); err != nil {
			t.Fatalf("Ping(): %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
	})

	t.Run("get set delete and invalid data", func(t *testing.T) {
		redisServer := miniredis.RunT(t)
		store, err := NewRedisCacheStore(RedisCacheConfig{Addr: redisServer.Addr(), Prefix: "demo:"})
		if err != nil {
			t.Fatalf("NewRedisCacheStore: %v", err)
		}
		ctx := context.Background()

		value := &CachedResponse{
			Status:  http.StatusCreated,
			Header:  http.Header{"X-Test": []string{"value"}},
			Body:    []byte("payload"),
			Expires: time.Now().Add(time.Minute),
		}
		store.SetContext(ctx, "users:1", value)
		got, ok := store.GetContext(ctx, "users:1")
		if !ok || got == nil || got.Status != http.StatusCreated || string(got.Body) != "payload" {
			t.Fatalf("GetContext() = (%+v, %v), want cached payload", got, ok)
		}

		store.AddTags("users:1", "users", "users", "")
		store.DeleteMany("users:1", "", "users:1")
		if _, ok := store.Get("users:1"); ok {
			t.Fatal("expected DeleteMany() to remove cached item")
		}

		redisServer.Set(store.cacheKey("broken"), "{not-json")
		if value, ok := store.Get("broken"); ok || value != nil {
			t.Fatalf("Get() = (%v, %v), want invalid payload miss", value, ok)
		}
		if redisServer.Exists(store.cacheKey("broken")) {
			t.Fatal("expected invalid payload to be deleted")
		}

		expired := CachedResponse{Status: http.StatusOK, Expires: time.Now().Add(-time.Minute)}
		payload, err := json.Marshal(expired)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		redisServer.Set(store.cacheKey("expired"), string(payload))
		if value, ok := store.Get("expired"); ok || value != nil {
			t.Fatalf("Get(expired) = (%v, %v), want miss", value, ok)
		}
		if redisServer.Exists(store.cacheKey("expired")) {
			t.Fatal("expected expired payload to be deleted")
		}

		store.SetContext(context.Background(), "expired-on-set", &CachedResponse{Expires: time.Now().Add(-time.Second)})
		if redisServer.Exists(store.cacheKey("expired-on-set")) {
			t.Fatal("expected already-expired value to be removed on SetContext")
		}

		store.Set("users:2", &CachedResponse{Status: http.StatusOK, Body: []byte("a"), Expires: time.Now().Add(time.Minute)})
		store.AddTags("users:2", "users")
		if removed := store.InvalidateTags("users"); removed != 1 {
			t.Fatalf("InvalidateTags() = %d, want 1", removed)
		}
		if _, ok := store.Get("users:2"); ok {
			t.Fatal("expected invalidated tag entry to be removed")
		}
	})

	t.Run("tag metadata follows cache ttl", func(t *testing.T) {
		redisServer := miniredis.RunT(t)
		store, err := NewRedisCacheStore(RedisCacheConfig{Addr: redisServer.Addr(), Prefix: "ttl:"})
		if err != nil {
			t.Fatalf("NewRedisCacheStore: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })

		store.AddTags("missing", "users")
		if redisServer.Exists(store.tagKey("users")) || redisServer.Exists(store.keyTagsKey("missing")) {
			t.Fatal("expected missing cache key to avoid creating tag metadata")
		}

		store.Set("long", &CachedResponse{Status: http.StatusOK, Expires: time.Now().Add(time.Minute)})
		store.AddTags("long", "users")
		store.Set("short", &CachedResponse{Status: http.StatusOK, Expires: time.Now().Add(time.Second)})
		store.AddTags("short", "users")

		if ttl := redisServer.TTL(store.keyTagsKey("short")); ttl <= 0 || ttl > time.Second {
			t.Fatalf("short key tag TTL = %v, want bounded by cache TTL", ttl)
		}
		if ttl := redisServer.TTL(store.tagKey("users")); ttl <= 30*time.Second {
			t.Fatalf("shared tag TTL = %v, want to preserve longer cache TTL", ttl)
		}

		redisServer.FastForward(2 * time.Second)
		if redisServer.Exists(store.cacheKey("short")) || redisServer.Exists(store.keyTagsKey("short")) {
			t.Fatal("expected short cache entry and key tag metadata to expire together")
		}
		if !redisServer.Exists(store.tagKey("users")) {
			t.Fatal("expected shared tag metadata to remain for longer-lived cache entry")
		}
		if removed := store.InvalidateTags("users"); removed != 1 {
			t.Fatalf("InvalidateTags() = %d, want only the live tagged key", removed)
		}
		if redisServer.Exists(store.cacheKey("long")) {
			t.Fatal("expected live tagged cache entry to be invalidated")
		}
	})
}
