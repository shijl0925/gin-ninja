package ninja

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRedisCachePrefix = "gin-ninja:"

// RedisCacheConfig configures the built-in Redis-backed response cache store.
type RedisCacheConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
	Prefix   string
}

// RedisCacheStore stores serialized route responses in Redis.
type RedisCacheStore struct {
	client *redis.Client
	prefix string
}

// NewRedisCacheStore creates a Redis-backed response cache store.
func NewRedisCacheStore(cfg RedisCacheConfig) (*RedisCacheStore, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, errors.New("redis cache: addr is required")
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = defaultRedisCachePrefix
	}
	return NewRedisCacheStoreWithClient(redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}), prefix), nil
}

// NewRedisCacheStoreWithClient wraps an existing Redis client.
func NewRedisCacheStoreWithClient(client *redis.Client, prefix string) *RedisCacheStore {
	if strings.TrimSpace(prefix) == "" {
		prefix = defaultRedisCachePrefix
	}
	return &RedisCacheStore{
		client: client,
		prefix: prefix,
	}
}

// Client exposes the underlying Redis client for health checks and shutdown hooks.
func (s *RedisCacheStore) Client() *redis.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// Close closes the underlying Redis client.
func (s *RedisCacheStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

// Ping verifies that the configured Redis server is reachable.
func (s *RedisCacheStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("redis cache: nil client")
	}
	return s.client.Ping(ctx).Err()
}

func (s *RedisCacheStore) Get(key string) (*CachedResponse, bool) {
	return s.GetContext(context.Background(), key)
}

func (s *RedisCacheStore) GetContext(ctx context.Context, key string) (*CachedResponse, bool) {
	if s == nil || s.client == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := s.client.Get(ctx, s.cacheKey(key)).Bytes()
	if err != nil {
		return nil, false
	}
	var cached CachedResponse
	if err := json.Unmarshal(payload, &cached); err != nil {
		_ = s.client.Del(ctx, s.cacheKey(key)).Err()
		return nil, false
	}
	if !cached.Expires.IsZero() && time.Now().After(cached.Expires) {
		s.Delete(key)
		return nil, false
	}
	return cloneCachedResponse(&cached), true
}

func (s *RedisCacheStore) Set(key string, value *CachedResponse) {
	s.SetContext(context.Background(), key, value)
}

func (s *RedisCacheStore) SetContext(ctx context.Context, key string, value *CachedResponse) {
	if s == nil || s.client == nil || value == nil || strings.TrimSpace(key) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	ttl := time.Duration(0)
	if !value.Expires.IsZero() {
		ttl = time.Until(value.Expires)
		if ttl <= 0 {
			s.Delete(key)
			return
		}
	}
	_ = s.client.Set(ctx, s.cacheKey(key), payload, ttl).Err()
}

func (s *RedisCacheStore) Delete(key string) {
	if s == nil || s.client == nil || strings.TrimSpace(key) == "" {
		return
	}
	s.deleteOne(context.Background(), key)
}

func (s *RedisCacheStore) DeleteMany(keys ...string) {
	if s == nil || s.client == nil {
		return
	}
	for _, key := range normalizeCacheTags(keys) {
		s.deleteOne(context.Background(), key)
	}
}

func (s *RedisCacheStore) AddTags(key string, tags ...string) {
	if s == nil || s.client == nil || strings.TrimSpace(key) == "" {
		return
	}
	normalized := normalizeCacheTags(tags)
	if len(normalized) == 0 {
		return
	}
	ctx := context.Background()
	keyTagsKey := s.keyTagsKey(key)
	pipe := s.client.TxPipeline()
	for _, tag := range normalized {
		pipe.SAdd(ctx, s.tagKey(tag), key)
		pipe.SAdd(ctx, keyTagsKey, tag)
	}
	_, _ = pipe.Exec(ctx)
}

func (s *RedisCacheStore) InvalidateTags(tags ...string) int {
	if s == nil || s.client == nil {
		return 0
	}
	ctx := context.Background()
	keys := map[string]struct{}{}
	for _, tag := range normalizeCacheTags(tags) {
		members, err := s.client.SMembers(ctx, s.tagKey(tag)).Result()
		if err != nil {
			continue
		}
		for _, key := range members {
			keys[key] = struct{}{}
		}
		_ = s.client.Del(ctx, s.tagKey(tag)).Err()
	}
	for key := range keys {
		s.deleteOne(ctx, key)
	}
	return len(keys)
}

func (s *RedisCacheStore) AcquireLock(key string, ttl time.Duration) (func(), bool) {
	if s == nil || s.client == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	ctx := context.Background()
	token := randomLockToken()
	ok, err := s.client.SetNX(ctx, s.lockKey(key), token, ttl).Result()
	if err != nil || !ok {
		return nil, false
	}
	return func() {
		const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`
		_ = s.client.Eval(ctx, script, []string{s.lockKey(key)}, token).Err()
	}, true
}

func (s *RedisCacheStore) deleteOne(ctx context.Context, key string) {
	tags, err := s.client.SMembers(ctx, s.keyTagsKey(key)).Result()
	if err == nil {
		pipe := s.client.TxPipeline()
		for _, tag := range tags {
			pipe.SRem(ctx, s.tagKey(tag), key)
		}
		pipe.Del(ctx, s.keyTagsKey(key))
		pipe.Del(ctx, s.cacheKey(key))
		_, _ = pipe.Exec(ctx)
		return
	}
	_ = s.client.Del(ctx, s.keyTagsKey(key), s.cacheKey(key)).Err()
}

func (s *RedisCacheStore) cacheKey(key string) string {
	return s.prefix + "cache:" + key
}

func (s *RedisCacheStore) tagKey(tag string) string {
	return s.prefix + "tag:" + tag
}

func (s *RedisCacheStore) keyTagsKey(key string) string {
	return s.prefix + "keytags:" + key
}

func (s *RedisCacheStore) lockKey(key string) string {
	return s.prefix + "lock:" + key
}

// randomLockToken generates a cryptographically random 16-byte hex token for
// distributed lock ownership.  Using a random token (instead of a timestamp)
// ensures that two concurrent callers on different processes or goroutines
// never share the same token, preventing accidental lock release.
func randomLockToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is a system-level error; panic rather than fall
		// back to a predictable value that would silently break lock semantics.
		panic("redis cache: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
