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

	LoggerMaxSizeMB  = 100
	LoggerMaxAgeDays = 7
	LoggerMaxBackups = 3
)
