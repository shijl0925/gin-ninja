package defaults

import "time"

const (
	RedisCachePrefix            = "gin-ninja:"
	MemoryCacheMaxEntries       = 1024
	CacheMaxBodyBytes     int64 = 1 << 20

	// Keep timeout captures large enough for typical JSON responses while
	// bounding memory held by handlers that keep writing after a timeout.
	TimeoutMaxBodyBytes = 32 << 20
	// Avoid creating an extra permanent drain goroutine for handlers that never
	// finish; panics after this window cannot be observed by the timeout wrapper.
	TimeoutDrainDuration = time.Minute

	RateLimiterPruneInterval = 5 * time.Minute
	RateLimiterClientTTL     = 5 * time.Minute

	MaxUploadSize int64 = 10 << 20

	CORSMaxAgeSecs = 43200

	LoggerMaxSizeMB  = 100
	LoggerMaxAgeDays = 7
	LoggerMaxBackups = 3
)

func CORSAllowOrigins() []string {
	return []string{"http://localhost:3000", "http://localhost:5173"}
}

func CORSAllowMethods() []string {
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
}

func CORSAllowHeaders() []string {
	return []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
}
