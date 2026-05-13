package ninja

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCacheInvalidatorAndCaptureWriterAdditionalCoverage(t *testing.T) {
	t.Run("invalidator delete and tag", func(t *testing.T) {
		store := NewMemoryCacheStore()
		store.Set("users:1", &CachedResponse{Status: http.StatusOK, Body: []byte("cached")})

		invalidator := NewCacheInvalidator(store)
		if removed := invalidator.Delete("", "users:1", "users:1"); removed != 1 {
			t.Fatalf("Delete() removed %d, want 1", removed)
		}
		if _, ok := store.Get("users:1"); ok {
			t.Fatal("expected deleted cache entry to be removed")
		}

		store.Set("users:2", &CachedResponse{Status: http.StatusOK, Body: []byte("cached")})
		if !invalidator.Tag("users:2", " users ", "", "users") {
			t.Fatal("expected Tag() to succeed")
		}
		if removed := invalidator.InvalidateTags("users"); removed != 1 {
			t.Fatalf("InvalidateTags() removed %d, want 1", removed)
		}
		if _, ok := store.Get("users:2"); ok {
			t.Fatal("expected tagged cache entry to be invalidated")
		}
	})

	t.Run("nil invalidator and capture writer", func(t *testing.T) {
		var invalidator *CacheInvalidator
		if got := invalidator.Delete("users:1"); got != 0 {
			t.Fatalf("Delete() = %d, want 0", got)
		}
		if invalidator.Tag("users:1", "users") {
			t.Fatal("expected Tag() to fail for nil invalidator")
		}

		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		writer := newCaptureResponseWriter(ctx.Writer)
		writer.Flush()
		if _, _, err := writer.Hijack(); !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
		}
	})
}
