package ninja

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

/*
Hot path benchmark usage:
  - Run all hot path benchmarks:
    go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem .
  - Run a single benchmark group:
    go test -run '^$' -bench '^BenchmarkHotpathsRouting$' -benchmem .
  - Run only one sub-benchmark implementation:
    go test -run '^$' -bench '^BenchmarkHotpathsRouting/gin-ninja$' -benchmem .
  - Reduce noise when comparing results:
    go test -run '^$' -bench '^BenchmarkHotpaths' -benchmem -count=5 .
*/

type benchmarkRouteInput struct {
	ID string `path:"id"`
}

type benchmarkRouteOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type benchmarkBindingInput struct {
	Limit int    `form:"limit" binding:"required,gte=1,lte=100"`
	Name  string `json:"name" binding:"required,min=3"`
	Count int    `json:"count" binding:"required,gte=1"`
}

type benchmarkBindingOutput struct {
	Limit int    `json:"limit"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	OK    bool   `json:"ok"`
}

type benchmarkBindingQuery struct {
	Limit int `form:"limit" binding:"required,gte=1,lte=100"`
}

type benchmarkBindingBody struct {
	Name  string `json:"name" binding:"required,min=3"`
	Count int    `json:"count" binding:"required,gte=1"`
}

func BenchmarkHotpathsRouting(b *testing.B) {
	b.Run("gin-ninja", func(b *testing.B) {
		handler := benchmarkNinjaRouteHandler()
		benchmarkServeHTTP(b, handler, func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/users/42", nil)
		})
	})

	b.Run("gin", func(b *testing.B) {
		handler := benchmarkGinRouteHandler()
		benchmarkServeHTTP(b, handler, func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/users/42", nil)
		})
	})
}

func BenchmarkHotpathsBinding(b *testing.B) {
	body := []byte(`{"name":"alice","count":3}`)

	b.Run("gin-ninja", func(b *testing.B) {
		handler := benchmarkNinjaBindingHandler()
		benchmarkServeHTTP(b, handler, func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/bindings?limit=20", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			return req
		})
	})

	b.Run("gin", func(b *testing.B) {
		handler := benchmarkGinBindingHandler()
		benchmarkServeHTTP(b, handler, func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/bindings?limit=20", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			return req
		})
	})
}

func BenchmarkHotpathsCacheHit(b *testing.B) {
	b.Run("gin-ninja", func(b *testing.B) {
		handler := benchmarkNinjaCacheHandler()
		req := func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/cache/42?lang=zh", nil)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req())
		if recorder.Code != http.StatusOK {
			b.Fatalf("warm cache status = %d", recorder.Code)
		}
		benchmarkServeHTTP(b, handler, req)
	})

	b.Run("gin", func(b *testing.B) {
		handler := benchmarkGinCacheHandler()
		req := func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/cache/42?lang=zh", nil)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req())
		if recorder.Code != http.StatusOK {
			b.Fatalf("warm cache status = %d", recorder.Code)
		}
		benchmarkServeHTTP(b, handler, req)
	})
}

func BenchmarkNormalizeVersionParam(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = normalizeVersionParam(" v2026.json ")
	}
}

func BenchmarkSSEDataJSONMap(b *testing.B) {
	value := map[string]any{
		"name":  "alice",
		"count": 3,
		"ok":    true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sseData(value)
	}
}

func benchmarkServeHTTP(b *testing.B, handler http.Handler, request func() *http.Request) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request())
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status = %d", recorder.Code)
		}
	}
}

func benchmarkNinjaRouteHandler() http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("/users")
	Get(router, "/:id", func(_ *Context, input *benchmarkRouteInput) (*benchmarkRouteOutput, error) {
		return &benchmarkRouteOutput{ID: input.ID, Name: "alice"}, nil
	})
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkGinRouteHandler() http.Handler {
	router := gin.New()
	router.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, benchmarkRouteOutput{ID: c.Param("id"), Name: "alice"})
	})
	return router
}

func benchmarkNinjaBindingHandler() http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("")
	Post(router, "/bindings", func(_ *Context, input *benchmarkBindingInput) (*benchmarkBindingOutput, error) {
		return &benchmarkBindingOutput{
			Limit: input.Limit,
			Name:  input.Name,
			Count: input.Count,
			OK:    true,
		}, nil
	}, SuccessStatus(http.StatusOK))
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkGinBindingHandler() http.Handler {
	router := gin.New()
	router.POST("/bindings", func(c *gin.Context) {
		var query benchmarkBindingQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var body benchmarkBindingBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, benchmarkBindingOutput{
			Limit: query.Limit,
			Name:  body.Name,
			Count: body.Count,
			OK:    true,
		})
	})
	return router
}

func benchmarkNinjaCacheHandler() http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("/cache")
	Get(router, "/:id", func(_ *Context, input *benchmarkRouteInput) (*benchmarkRouteOutput, error) {
		return &benchmarkRouteOutput{ID: input.ID, Name: "alice"}, nil
	}, Cache(time.Minute))
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkGinCacheHandler() http.Handler {
	router := gin.New()
	store := NewMemoryCacheStore()
	ttl := time.Minute
	cacheControl := defaultCacheControl(ttl)

	router.GET("/cache/:id", benchmarkNativeCacheMiddleware(ttl, store, cacheControl), func(c *gin.Context) {
		c.JSON(http.StatusOK, benchmarkRouteOutput{ID: c.Param("id"), Name: "alice"})
	})
	return router
}

func benchmarkNativeCacheMiddleware(ttl time.Duration, store ResponseCacheStore, cacheControl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCacheableMethod(c.Request.Method) {
			c.Next()
			return
		}

		cacheKey := c.Request.Method + ":" + c.Request.URL.RequestURI()
		if cached, ok := store.Get(cacheKey); ok && !isExpiredCachedResponse(cached, time.Now()) {
			writeCachedResponse(c, cached, cacheControl)
			c.Abort()
			return
		}

		originalWriter := c.Writer
		recorder := newCaptureResponseWriter(originalWriter)
		c.Writer = recorder
		c.Next()
		c.Writer = originalWriter

		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		if cacheControl != "" && recorder.status >= 200 && recorder.status < 400 && recorder.header.Get("Cache-Control") == "" {
			recorder.header.Set("Cache-Control", cacheControl)
		}

		etag := recorder.header.Get("ETag")
		if etag == "" && recorder.status >= 200 && recorder.status < 400 && len(recorder.body) > 0 {
			etag = generateETag(recorder.body)
			recorder.header.Set("ETag", etag)
		}

		copyHeader(originalWriter.Header(), recorder.header)
		originalWriter.WriteHeader(recorder.status)
		if len(recorder.body) > 0 && c.Request.Method != http.MethodHead {
			_, _ = originalWriter.Write(recorder.body)
		}

		if recorder.status >= 200 && recorder.status < 300 {
			store.Set(cacheKey, &CachedResponse{
				Status:  recorder.status,
				Header:  cloneHeader(recorder.header),
				Body:    append([]byte(nil), recorder.body...),
				Expires: time.Now().Add(ttl),
				ETag:    etag,
			})
		}
	}
}
