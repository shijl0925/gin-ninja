package redis

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	ninja "github.com/shijl0925/gin-ninja"
)

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
		store.Set("key", &ninja.CachedResponse{})
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

	t.Run("get set delete and invalid cache entries", func(t *testing.T) {
		redisServer := miniredis.RunT(t)
		store, err := NewRedisCacheStore(RedisCacheConfig{Addr: redisServer.Addr(), Prefix: "demo:"})
		if err != nil {
			t.Fatalf("NewRedisCacheStore: %v", err)
		}
		ctx := context.Background()

		value := &ninja.CachedResponse{
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

		expired := ninja.CachedResponse{Status: http.StatusOK, Expires: time.Now().Add(-time.Minute)}
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

		store.Set("users:2", &ninja.CachedResponse{Status: http.StatusOK, Body: []byte("a"), Expires: time.Now().Add(time.Minute)})
		store.AddTags("users:2", "users")
		if removed := store.InvalidateTags("users"); removed != 1 {
			t.Fatalf("InvalidateTags() = %d, want 1", removed)
		}
		if _, ok := store.Get("users:2"); ok {
			t.Fatal("expected invalidated tag entry to be removed")
		}
	})
}

func TestNormalizeCacheTags(t *testing.T) {
	got := normalizeCacheTags([]string{" users ", "", "users", "posts", "  "})
	want := []string{"users", "posts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCacheTags() = %#v, want %#v", got, want)
	}
}

func TestCloneCachedResponse(t *testing.T) {
	if cloneCachedResponse(nil) != nil {
		t.Fatal("expected nil clone for nil response")
	}

	original := &ninja.CachedResponse{
		Status: http.StatusOK,
		Header: http.Header{
			"X-Test": []string{"one"},
		},
		Body: []byte("payload"),
	}
	cloned := cloneCachedResponse(original)
	if cloned == original {
		t.Fatal("expected distinct response pointer")
	}
	cloned.Header.Set("X-Test", "two")
	cloned.Body[0] = 'P'
	if original.Header.Get("X-Test") != "one" || string(original.Body) != "payload" {
		t.Fatal("expected cloned response to be independent")
	}
}
