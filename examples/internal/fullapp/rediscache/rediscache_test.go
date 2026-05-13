package rediscache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/cache/redis"
	"github.com/shijl0925/gin-ninja/settings"
)

func TestNewCacheStore(t *testing.T) {
	redisServer := miniredis.RunT(t)
	cfg := settings.Config{
		Redis: settings.RedisConfig{
			Enabled: true,
			Addr:    redisServer.Addr(),
			Prefix:  "fullapp:",
		},
	}

	store, shutdown := NewCacheStore(cfg)
	if _, ok := store.(*redis.RedisCacheStore); !ok {
		t.Fatalf("expected redis cache store, got %T", store)
	}
	if shutdown == nil {
		t.Fatal("expected redis shutdown hook")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown(): %v", err)
	}

	cfg.Redis.Addr = ""
	store, shutdown = NewCacheStore(cfg)
	if _, ok := store.(*ninja.MemoryCacheStore); !ok {
		t.Fatalf("expected fallback cache store, got %T", store)
	}
	if shutdown != nil {
		t.Fatalf("expected nil shutdown after fallback, got nil=%t", shutdown == nil)
	}

	cfg.Redis.Addr = "127.0.0.1:1"
	store, shutdown = NewCacheStore(cfg)
	if _, ok := store.(*ninja.MemoryCacheStore); !ok {
		t.Fatalf("expected ping failure fallback store, got %T", store)
	}
	if shutdown != nil {
		t.Fatalf("expected nil shutdown after ping failure fallback, got nil=%t", shutdown == nil)
	}
}
