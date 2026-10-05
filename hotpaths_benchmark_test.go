package ninja

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/defaults"
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
	Limit int    `query:"limit" binding:"required,gte=1,lte=100"`
	Name  string `json:"name" binding:"required,min=3"`
	Count int    `json:"count" binding:"required,gte=1"`
}

type benchmarkBindingOutput struct {
	Limit int    `json:"limit"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	OK    bool   `json:"ok"`
}

type benchmarkGinBindingForm struct {
	Limit int `form:"limit" binding:"required,gte=1,lte=100"`
}

type benchmarkBindingBody struct {
	Name  string `json:"name" binding:"required,min=3"`
	Count int    `json:"count" binding:"required,gte=1"`
}

type benchmarkBindingMultiSourceInput struct {
	ID     string `path:"id"`
	Limit  int    `query:"limit" binding:"required,gte=1,lte=100"`
	Page   int    `query:"page" default:"1"`
	Search string `query:"search"`
	Token  string `header:"X-Token"`
	Locale string `cookie:"locale"`
	Name   string `json:"name" binding:"required,min=3"`
	Count  int    `json:"count" binding:"required,gte=1"`
}

type benchmarkLargeResponseOutput struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
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

func BenchmarkHotpathsOpenAPI(b *testing.B) {
	const routeCount = 200

	b.Run("cold", func(b *testing.B) {
		api := benchmarkLargeOpenAPI(routeCount)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			api.invalidateOpenAPICache()
			spec, err := api.openAPIBytes()
			if err != nil {
				b.Fatalf("OpenAPI() error = %v", err)
			}
			if len(spec) == 0 {
				b.Fatal("OpenAPI() returned empty spec")
			}
		}
	})

	b.Run("warm", func(b *testing.B) {
		api := benchmarkLargeOpenAPI(routeCount)
		if _, err := api.openAPIBytes(); err != nil {
			b.Fatalf("warm OpenAPI() error = %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			spec, err := api.openAPIBytes()
			if err != nil {
				b.Fatalf("OpenAPI() error = %v", err)
			}
			if len(spec) == 0 {
				b.Fatal("OpenAPI() returned empty spec")
			}
		}
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

func BenchmarkHotpathsBindingMultiSource(b *testing.B) {
	body := []byte(`{"name":"alice","count":3}`)
	handler := benchmarkNinjaBindingMultiSourceHandler()
	benchmarkServeHTTP(b, handler, func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/bindings/42?limit=20&search=abc", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Token", "token")
		req.AddCookie(&http.Cookie{Name: "locale", Value: "en"})
		return req
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

func BenchmarkHotpathsCacheMiss(b *testing.B) {
	handler := benchmarkNinjaCacheHandler()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/cache/42?lang=zh&miss=%d", i), nil)
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Fatalf("unexpected status = %d", recorder.Code)
		}
	}
}

func BenchmarkHotpathsCacheETag(b *testing.B) {
	handler := benchmarkNinjaCacheHandler()
	req := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/cache/42?lang=zh", nil)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req())
	if recorder.Code != http.StatusOK {
		b.Fatalf("warm cache status = %d", recorder.Code)
	}
	etag := recorder.Header().Get("ETag")
	if etag == "" {
		b.Fatal("warm cache response did not include ETag")
	}
	benchmarkServeHTTPStatus(b, handler, http.StatusNotModified, func() *http.Request {
		request := req()
		request.Header.Set("If-None-Match", etag)
		return request
	})
}

func BenchmarkHotpathsCacheLargeBody(b *testing.B) {
	handler := benchmarkNinjaLargeCacheHandler(64 * 1024)
	req := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/cache-large/42", nil)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req())
	if recorder.Code != http.StatusOK {
		b.Fatalf("warm large cache status = %d", recorder.Code)
	}
	benchmarkServeHTTP(b, handler, req)
}

func BenchmarkHotpathsMiddlewareDepth(b *testing.B) {
	for _, depth := range []int{0, 1, 5, 10, 20} {
		b.Run(fmt.Sprintf("depth-%d", depth), func(b *testing.B) {
			handler := benchmarkNinjaMiddlewareDepthHandler(depth)
			benchmarkServeHTTP(b, handler, func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/middleware/42", nil)
			})
		})
	}
}

func BenchmarkHotpathsParallelRouting(b *testing.B) {
	handler := benchmarkNinjaRouteHandler()
	benchmarkServeHTTPParallel(b, handler, http.StatusOK, func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/users/42", nil)
	})
}

func BenchmarkHotpathsParallelBinding(b *testing.B) {
	body := []byte(`{"name":"alice","count":3}`)
	handler := benchmarkNinjaBindingHandler()
	benchmarkServeHTTPParallel(b, handler, http.StatusOK, func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/bindings?limit=20", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	})
}

func BenchmarkHotpathsParallelCacheHit(b *testing.B) {
	handler := benchmarkNinjaCacheHandler()
	req := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/cache/42?lang=zh", nil)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req())
	if recorder.Code != http.StatusOK {
		b.Fatalf("warm cache status = %d", recorder.Code)
	}
	benchmarkServeHTTPParallel(b, handler, http.StatusOK, req)
}

func BenchmarkHotpathsRedisTagInvalidation(b *testing.B) {
	for _, keyCount := range []int{100, 1000, 10000, 100000} {
		b.Run(fmt.Sprintf("keys-%d", keyCount), func(b *testing.B) {
			redisServer := miniredis.RunT(b)
			store, err := NewRedisCacheStore(RedisCacheConfig{Addr: redisServer.Addr(), Prefix: "bench:"})
			if err != nil {
				b.Fatalf("NewRedisCacheStore() error = %v", err)
			}
			defer func() { _ = store.Close() }()

			ctx := context.Background()
			b.ReportAllocs()
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				if err := store.client.FlushDB(ctx).Err(); err != nil {
					b.Fatalf("FlushDB() error = %v", err)
				}
				tag := fmt.Sprintf("users:%d", i)
				benchmarkSeedRedisTaggedKeys(b, store, tag, keyCount)

				b.StartTimer()
				removed := store.InvalidateTags(tag)
				b.StopTimer()

				if removed != keyCount {
					b.Fatalf("InvalidateTags() removed %d keys, want %d", removed, keyCount)
				}
			}
		})
	}
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
	benchmarkServeHTTPStatus(b, handler, http.StatusOK, request)
}

func benchmarkServeHTTPStatus(b *testing.B, handler http.Handler, status int, request func() *http.Request) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request())
		if recorder.Code != status {
			b.Fatalf("unexpected status = %d", recorder.Code)
		}
	}
}

func benchmarkServeHTTPParallel(b *testing.B, handler http.Handler, status int, request func() *http.Request) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request())
			if recorder.Code != status {
				b.Fatalf("unexpected status = %d", recorder.Code)
			}
		}
	})
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

func benchmarkLargeOpenAPI(routeCount int) *NinjaAPI {
	api := New(Config{
		DisableGinDefault: true,
		Title:             "Benchmark API",
		Version:           "v1",
	})
	for group := 0; group < 10; group++ {
		router := NewRouter(fmt.Sprintf("/groups/%d", group),
			WithTags(fmt.Sprintf("Group%d", group)),
			WithTagDescription(fmt.Sprintf("Group%d", group), "Synthetic benchmark routes"),
		)
		for route := 0; route < routeCount/10; route++ {
			path := fmt.Sprintf("/items/%d/:id", route)
			operationID := fmt.Sprintf("benchmarkGroup%dRoute%d", group, route)
			Get(router, path, func(_ *Context, input *benchmarkRouteInput) (*benchmarkRouteOutput, error) {
				return &benchmarkRouteOutput{ID: input.ID, Name: "alice"}, nil
			},
				OperationID(operationID),
				Summary("Fetch benchmark item"),
				Description("Synthetic endpoint used to benchmark OpenAPI generation."),
				Response(http.StatusNotFound, "Item not found", nil),
			)
		}
		api.AddRouter(router)
	}
	return api
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

func benchmarkNinjaBindingMultiSourceHandler() http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("")
	Post(router, "/bindings/:id", func(_ *Context, input *benchmarkBindingMultiSourceInput) (*benchmarkBindingOutput, error) {
		return &benchmarkBindingOutput{
			Limit: input.Limit,
			Name:  input.Name,
			Count: input.Count,
			OK:    input.ID != "" && input.Page == 1 && input.Token != "" && input.Locale != "",
		}, nil
	}, SuccessStatus(http.StatusOK))
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkGinBindingHandler() http.Handler {
	router := gin.New()
	router.POST("/bindings", func(c *gin.Context) {
		var query benchmarkGinBindingForm
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

func benchmarkNinjaLargeCacheHandler(size int) http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("/cache-large")
	payload := strings.Repeat("x", size)
	Get(router, "/:id", func(_ *Context, input *benchmarkRouteInput) (*benchmarkLargeResponseOutput, error) {
		return &benchmarkLargeResponseOutput{ID: input.ID, Payload: payload}, nil
	}, Cache(time.Minute, CacheWithMaxBodyBytes(int64(size*2))))
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkNinjaMiddlewareDepthHandler(depth int) http.Handler {
	api := New(Config{DisableGinDefault: true})
	router := NewRouter("/middleware")
	for i := 0; i < depth; i++ {
		router.UseGin(func(c *gin.Context) {
			c.Next()
		})
	}
	Get(router, "/:id", func(_ *Context, input *benchmarkRouteInput) (*benchmarkRouteOutput, error) {
		return &benchmarkRouteOutput{ID: input.ID, Name: "alice"}, nil
	})
	api.AddRouter(router)
	return api.Handler()
}

func benchmarkNativeCacheMiddleware(ttl time.Duration, store ResponseCacheStore, cacheControl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCacheableMethod(c.Request.Method) {
			c.Next()
			return
		}

		cacheKey := c.Request.Method + ":" + c.Request.URL.RequestURI()
		if cached, ok := store.Get(cacheKey); ok && !isExpiredCachedResponse(cached, time.Now()) {
			writeCachedResponse(c, cached, cacheControl, defaultCacheVaryHeaders...)
			c.Abort()
			return
		}

		originalWriter := c.Writer
		recorder := newCaptureResponseWriter(originalWriter, defaults.CacheMaxBodyBytes)
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

func benchmarkSeedRedisTaggedKeys(b *testing.B, store *RedisCacheStore, tag string, count int) {
	b.Helper()
	expires := time.Now().Add(time.Hour)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%s:key:%d", tag, i)
		store.Set(key, &CachedResponse{
			Status:  http.StatusOK,
			Header:  http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"ok":true}`),
			Expires: expires,
		})
		store.AddTags(key, tag)
	}
}
