package ninja

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterPruneLockedRemovesIdleClients(t *testing.T) {
	now := time.Now()
	limiter := newRateLimiter(1, 1)
	limiter.clients["expired"] = &tokenBucket{tokens: 1, last: now.Add(-rateLimiterClientTTL - time.Second)}
	limiter.clients["active"] = &tokenBucket{tokens: 1, last: now.Add(-rateLimiterClientTTL + time.Second)}

	limiter.mu.Lock()
	limiter.pruneLocked(now)
	limiter.mu.Unlock()

	if _, ok := limiter.clients["expired"]; ok {
		t.Fatal("expected expired client bucket to be pruned")
	}
	if _, ok := limiter.clients["active"]; !ok {
		t.Fatal("expected active client bucket to remain")
	}
	if !limiter.lastPrune.Equal(now) {
		t.Fatalf("expected lastPrune to be updated to %s, got %s", now, limiter.lastPrune)
	}
}

func TestRateLimiterAllowCapsTokensAndPrunes(t *testing.T) {
	now := time.Now()
	limiter := newRateLimiter(10, 2)
	limiter.clients["client"] = &tokenBucket{tokens: 1, last: now.Add(-time.Second)}
	limiter.clients["expired"] = &tokenBucket{tokens: 1, last: now.Add(-rateLimiterClientTTL - time.Second)}
	limiter.lastPrune = now.Add(-rateLimiterPruneInterval - time.Second)

	if !limiter.allow("client", now) {
		t.Fatal("expected request to be allowed")
	}
	if got := limiter.clients["client"].tokens; got != 1 {
		t.Fatalf("expected tokens to be capped at burst then consumed, got %v", got)
	}
	if _, ok := limiter.clients["expired"]; ok {
		t.Fatal("expected allow to prune expired client")
	}
}

func TestWrapCooperativeTimeoutWritesTimeoutWhenHandlerCooperates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := wrapCooperativeTimeout(time.Nanosecond, func(c *gin.Context) {
		<-c.Request.Context().Done()
	})
	handler(c)

	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected timeout status %d, got %d", http.StatusRequestTimeout, w.Code)
	}
	if !c.IsAborted() {
		t.Fatal("expected context to be aborted after timeout")
	}
}

func TestWrapCooperativeTimeoutPreservesWrittenResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := wrapCooperativeTimeout(time.Nanosecond, func(c *gin.Context) {
		<-c.Request.Context().Done()
		c.String(http.StatusAccepted, "accepted")
	})
	handler(c)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected written status %d, got %d", http.StatusAccepted, w.Code)
	}
	if w.Body.String() != "accepted" {
		t.Fatalf("expected written body to be preserved, got %q", w.Body.String())
	}
}

func TestWrapTimeoutDefaultsOKWhenHandlerWritesNothing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := wrapTimeout(time.Second, func(c *gin.Context) {})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected default status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWrapTimeoutPropagatesHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	defer func() {
		if got := recover(); got != "boom" {
			t.Fatalf("expected propagated panic %q, got %v", "boom", got)
		}
	}()

	handler := wrapTimeout(time.Second, func(c *gin.Context) {
		panic("boom")
	})
	handler(c)
}

func TestTimeoutCaptureResponseWriterMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(baseRecorder)
	writer := newTimeoutCaptureResponseWriter(c.Writer, 8)

	writer.Header().Set("X-Test", "yes")
	if got := writer.Header().Get("X-Test"); got != "yes" {
		t.Fatalf("expected captured header, got %q", got)
	}
	if writer.Written() {
		t.Fatal("expected fresh capture writer to be unwritten")
	}

	writer.WriteHeaderNow()
	if writer.Status() != http.StatusOK {
		t.Fatalf("expected default status %d, got %d", http.StatusOK, writer.Status())
	}
	writer.WriteHeader(http.StatusCreated)
	if writer.Status() != http.StatusOK {
		t.Fatalf("expected first status to win, got %d", writer.Status())
	}

	n, err := writer.WriteString("hello")
	if err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if n != len("hello") {
		t.Fatalf("expected WriteString to report %d bytes, got %d", len("hello"), n)
	}
	if writer.Size() != len("hello") {
		t.Fatalf("expected captured size %d, got %d", len("hello"), writer.Size())
	}
	if !writer.Written() {
		t.Fatal("expected capture writer to report written")
	}

	n, err = writer.Write([]byte(" world"))
	if err != nil {
		t.Fatalf("Write overflow: %v", err)
	}
	if n != len(" world") {
		t.Fatalf("expected overflow write to report %d bytes, got %d", len(" world"), n)
	}
	if string(writer.body) != "hello wo" {
		t.Fatalf("expected body capped at max bytes, got %q", string(writer.body))
	}
	if !writer.overflowed {
		t.Fatal("expected overflow flag after capped write")
	}

	writer.Flush()
	if !writer.streamed || !writer.Written() {
		t.Fatal("expected Flush to mark response as streamed and written")
	}
	if conn, rw, err := writer.Hijack(); conn != nil || rw != nil || !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("expected unsupported hijack, got conn=%v rw=%v err=%v", conn, rw, err)
	}
	if pusher := writer.Pusher(); pusher != nil {
		t.Fatalf("expected nil pusher, got %v", pusher)
	}
}

func TestTimeoutCaptureResponseWriterOverflowVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(baseRecorder)

	writer := newTimeoutCaptureResponseWriter(c.Writer, 0)
	n, err := writer.Write([]byte("full"))
	if err != nil {
		t.Fatalf("Write with no remaining capacity: %v", err)
	}
	if n != len("full") {
		t.Fatalf("expected no-capacity write to report %d bytes, got %d", len("full"), n)
	}
	if !writer.overflowed {
		t.Fatal("expected no-capacity write to mark overflow")
	}

	n, err = writer.Write([]byte("ignored"))
	if err != nil {
		t.Fatalf("Write after overflow: %v", err)
	}
	if n != len("ignored") {
		t.Fatalf("expected post-overflow write to report %d bytes, got %d", len("ignored"), n)
	}
}
