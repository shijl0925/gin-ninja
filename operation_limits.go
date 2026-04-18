package ninja

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// tokenBucket is a single token-bucket entry keyed by client IP.
type tokenBucket struct {
	tokens float64
	last   time.Time
}

const (
	rateLimiterPruneInterval = 5 * time.Minute
	rateLimiterClientTTL     = 5 * time.Minute
)

// rateLimiter manages per-client-IP token buckets so that a single client
// cannot exhaust the rate limit for all other callers.
type rateLimiter struct {
	mu        sync.Mutex
	rate      float64
	burst     float64
	clients   map[string]*tokenBucket
	lastPrune time.Time
}

func newRateLimiter(rate, burst float64) *rateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &rateLimiter{
		rate:      rate,
		burst:     burst,
		clients:   make(map[string]*tokenBucket),
		lastPrune: time.Now(),
	}
}

// allow reports whether the request from clientIP should be allowed through.
func (l *rateLimiter) allow(clientIP string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.clients[clientIP]
	if !ok {
		bucket = &tokenBucket{tokens: l.burst, last: now}
		l.clients[clientIP] = bucket
	}

	elapsed := now.Sub(bucket.last).Seconds()
	bucket.last = now
	bucket.tokens += elapsed * l.rate
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}

	allowed := bucket.tokens >= 1
	if allowed {
		bucket.tokens--
	}

	if now.Sub(l.lastPrune) > rateLimiterPruneInterval {
		l.pruneLocked(now)
	}

	return allowed
}

// pruneLocked removes client entries that have been idle longer than
// rateLimiterClientTTL.  Must be called with l.mu held.
func (l *rateLimiter) pruneLocked(now time.Time) {
	for ip, bucket := range l.clients {
		if now.Sub(bucket.last) > rateLimiterClientTTL {
			delete(l.clients, ip)
		}
	}
	l.lastPrune = now
}

func wrapRateLimit(limiter *rateLimiter, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP(), time.Now()) {
			writeError(c, &Error{
				Status:  http.StatusTooManyRequests,
				Code:    "RATE_LIMITED",
				Message: "rate limit exceeded",
			})
			c.Abort()
			return
		}
		next(c)
	}
}

func wrapTimeout(timeout time.Duration, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		recorder := newCaptureResponseWriter(c.Writer)
		copied := c.Copy()
		copied.Request = copied.Request.WithContext(reqCtx)
		copied.Writer = recorder

		resultCh := make(chan any, 1)
		go func() {
			defer func() {
				resultCh <- recover()
			}()
			next(copied)
		}()

		select {
		case panicValue := <-resultCh:
			if panicValue != nil {
				panic(panicValue)
			}
			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}
			copyHeader(c.Writer.Header(), recorder.header)
			c.Status(recorder.status)
			if len(recorder.body) > 0 && c.Request.Method != http.MethodHead {
				_, _ = c.Writer.Write(recorder.body)
			}
		case <-reqCtx.Done():
			// cancel() is called explicitly here (in addition to the deferred call)
			// to propagate the cancellation signal to the handler goroutine as early
			// as possible. Well-behaved handlers that check their context will stop
			// promptly; handlers that do not check the context will run to completion
			// on their own — Go's cooperative concurrency model does not allow
			// forceful goroutine termination.
			cancel()
			if errors.Is(reqCtx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
				writeError(c, &Error{
					Status:  http.StatusRequestTimeout,
					Code:    "REQUEST_TIMEOUT",
					Message: "request timed out",
				})
			}
			c.Abort()
		}
	}
}
