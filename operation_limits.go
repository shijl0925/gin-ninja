package ninja

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/defaults"
)

// tokenBucket is a single token-bucket entry keyed by client IP.
type tokenBucket struct {
	tokens float64
	last   time.Time
}

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

	if now.Sub(l.lastPrune) > defaults.RateLimiterPruneInterval {
		l.pruneLocked(now)
	}

	return allowed
}

// pruneLocked removes client entries that have been idle longer than
// defaults.RateLimiterClientTTL. Must be called with l.mu held.
func (l *rateLimiter) pruneLocked(now time.Time) {
	for ip, bucket := range l.clients {
		if now.Sub(bucket.last) > defaults.RateLimiterClientTTL {
			delete(l.clients, ip)
		}
	}
	l.lastPrune = now
}

func wrapRateLimit(limiter *rateLimiter, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP(), time.Now()) {
			WriteError(c, &Error{
				Code:    http.StatusTooManyRequests,
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

		recorder := newTimeoutCaptureResponseWriter(c.Writer, defaults.TimeoutMaxBodyBytes)
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
			if recorder.overflowed {
				WriteError(c, &Error{
					Code:    http.StatusInternalServerError,
					Message: "response exceeded timeout capture limit",
				})
				return
			}
			if recorder.streamed {
				WriteError(c, &Error{
					Code:    http.StatusInternalServerError,
					Message: "response could not be captured by timeout wrapper",
				})
				return
			}
			copyHeader(c.Writer.Header(), recorder.header)
			c.Status(recorder.status)
			c.Writer.WriteHeaderNow()
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
				WriteError(c, &Error{
					Code:    http.StatusRequestTimeout,
					Message: "request timed out",
				})
			}
			c.Abort()
			go drainTimeoutResult(resultCh)
		}
	}
}

func wrapCooperativeTimeout(timeout time.Duration, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(reqCtx)
		next(c)

		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
			WriteError(c, &Error{
				Code:    http.StatusRequestTimeout,
				Message: "request timed out",
			})
			c.Abort()
		}
	}
}

func drainTimeoutResult(resultCh <-chan any) {
	timer := time.NewTimer(defaults.TimeoutDrainDuration)
	defer timer.Stop()

	select {
	case panicValue := <-resultCh:
		if panicValue != nil {
			logTimeoutPanic(panicValue)
		}
	case <-timer.C:
	}
}

func logTimeoutPanic(panicValue any) {
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[GIN-NINJA] panic after timeout: %v\n", panicValue)
}

type timeoutCaptureResponseWriter struct {
	gin.ResponseWriter
	header       http.Header
	body         []byte
	status       int
	maxBodyBytes int
	overflowed   bool
	streamed     bool
}

func newTimeoutCaptureResponseWriter(base gin.ResponseWriter, maxBodyBytes int) *timeoutCaptureResponseWriter {
	return &timeoutCaptureResponseWriter{
		ResponseWriter: base,
		header:         http.Header{},
		maxBodyBytes:   maxBodyBytes,
	}
}

func (w *timeoutCaptureResponseWriter) Header() http.Header {
	return w.header
}

func (w *timeoutCaptureResponseWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
}

func (w *timeoutCaptureResponseWriter) WriteHeaderNow() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
}

func (w *timeoutCaptureResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.overflowed {
		// Report bytes as consumed so gin.ResponseWriter callers are not exposed
		// to an artificial capture-only write failure after the cap is reached.
		return len(data), nil
	}
	remaining := w.maxBodyBytes - len(w.body)
	if remaining <= 0 {
		w.overflowed = true
		return len(data), nil
	}
	if len(data) > remaining {
		w.body = append(w.body, data[:remaining]...)
		w.overflowed = true
		return len(data), nil
	}
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *timeoutCaptureResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *timeoutCaptureResponseWriter) Status() int {
	return w.status
}

func (w *timeoutCaptureResponseWriter) Size() int {
	return len(w.body)
}

func (w *timeoutCaptureResponseWriter) Written() bool {
	return w.status != 0 || len(w.body) > 0 || w.overflowed || w.streamed
}

func (w *timeoutCaptureResponseWriter) Flush() {
	w.streamed = true
}

func (w *timeoutCaptureResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.streamed = true
	return nil, nil, http.ErrNotSupported
}

func (w *timeoutCaptureResponseWriter) Pusher() http.Pusher {
	return nil
}
