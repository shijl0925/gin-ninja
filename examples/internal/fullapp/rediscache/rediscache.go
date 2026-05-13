// Package rediscache wires Redis response caching into the full example when imported.
package rediscache

import (
	"context"
	"log"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/cache/redis"
	"github.com/shijl0925/gin-ninja/examples/internal/fullapp"
	"github.com/shijl0925/gin-ninja/settings"
)

func init() {
	fullapp.RegisterCacheStoreFactory(NewCacheStore)
}

// NewCacheStore creates a Redis-backed cache store, falling back to memory when Redis is unavailable.
func NewCacheStore(cfg settings.Config) (ninja.ResponseCacheStore, func(context.Context) error) {
	cacheStore := ninja.ResponseCacheStore(ninja.NewMemoryCacheStore())
	var cacheStoreShutdown func(context.Context) error
	redisStore, err := redis.NewRedisCacheStore(redis.RedisCacheConfig{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Prefix:   cfg.Redis.Prefix,
	})
	if err != nil {
		log.Printf("cache: falling back to in-memory store: %v", err)
	} else if err := redisStore.Ping(context.Background()); err != nil {
		log.Printf("cache: redis unavailable, falling back to in-memory store: %v", err)
		_ = redisStore.Close()
	} else {
		cacheStore = redisStore
		cacheStoreShutdown = func(context.Context) error { return redisStore.Close() }
		log.Printf("cache: using redis store at %s", cfg.Redis.Addr)
	}
	return cacheStore, cacheStoreShutdown
}
