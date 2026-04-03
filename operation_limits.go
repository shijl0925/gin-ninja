package ninja

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newRateLimiter(rate, burst float64) *rateLimiter {
	if burst < 1 {
		burst = 1
	}
	now := time.Now()
	return &rateLimiter{
		rate:   rate,
		burst:  burst,
		tokens: burst,
		last:   now,
	}
}

func (l *rateLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	elapsed := now.Sub(l.last).Seconds()
	l.last = now
	l.tokens += elapsed * l.rate
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func wrapRateLimit(limiter *rateLimiter, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.allow(time.Now()) {
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

		c.Request = c.Request.WithContext(reqCtx)
		next(c)

		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
			writeError(c, &Error{
				Status:  http.StatusRequestTimeout,
				Code:    "REQUEST_TIMEOUT",
				Message: "request timed out",
			})
			c.Abort()
		}
	}
}
