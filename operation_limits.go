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
			//
			// resultCh has capacity 1, so the goroutine can always write to it
			// without blocking and will be eligible for GC as soon as it returns.
			// The background drain below makes that contract explicit and avoids
			// any confusion if the channel capacity is ever changed.
			cancel()
			go func() { <-resultCh }()
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
