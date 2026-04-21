package ninja

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNormalizeVersionParam(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim whitespace", input: "  v1  ", want: "v1"},
		{name: "strip json suffix", input: "v2.json", want: "v2"},
		{name: "empty", input: "   ", want: ""},
		{name: "unicode", input: "版本1.json", want: "版本1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeVersionParam(tc.input); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRequestVersionPrefersVersionAndVersionJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("version param", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/docs/v1", nil)
		c.Params = gin.Params{{Key: "version", Value: " v1 "}}
		if got := requestVersion(c); got != "v1" {
			t.Fatalf("expected version param, got %q", got)
		}
	})

	t.Run("version json param", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/openapi/v2.json", nil)
		c.Params = gin.Params{{Key: "version.json", Value: "v2.json"}}
		if got := requestVersion(c); got != "v2" {
			t.Fatalf("expected version.json param, got %q", got)
		}
	})
}

func TestVersionDeprecationMiddleware_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("deprecated version emits headers and sunset time wins", func(t *testing.T) {
		sunsetTime := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
		router := gin.New()
		router.Use(versionDeprecationMiddleware(VersionConfig{
			Deprecated:   true,
			Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
			SunsetTime:   sunsetTime,
			MigrationURL: "https://example.com/migrate",
		}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Deprecation"); got != "true" {
			t.Fatalf("expected Deprecation header, got %q", got)
		}
		if got := w.Header().Get("Sunset"); got != sunsetTime.Format(http.TimeFormat) {
			t.Fatalf("expected Sunset header from SunsetTime, got %q", got)
		}
		if got := w.Header().Get("Link"); got != `<https://example.com/migrate>; rel="deprecation"` {
			t.Fatalf("expected Link header, got %q", got)
		}
	})

	t.Run("deprecated version emits sunset header from compatibility string", func(t *testing.T) {
		router := gin.New()
		router.Use(versionDeprecationMiddleware(VersionConfig{
			Deprecated: true,
			Sunset:     "Wed, 31 Dec 2026 23:59:59 GMT",
		}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Sunset"); got != "Wed, 31 Dec 2026 23:59:59 GMT" {
			t.Fatalf("expected Sunset header from Sunset string, got %q", got)
		}
	})

	t.Run("non-deprecated version emits no deprecation headers", func(t *testing.T) {
		router := gin.New()
		router.Use(versionDeprecationMiddleware(VersionConfig{
			Deprecated:   false,
			Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
			MigrationURL: "https://example.com/migrate",
		}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Deprecation"); got != "" {
			t.Fatalf("did not expect Deprecation header, got %q", got)
		}
		if got := w.Header().Get("Sunset"); got != "" {
			t.Fatalf("did not expect Sunset header, got %q", got)
		}
		if got := w.Header().Get("Link"); got != "" {
			t.Fatalf("did not expect Link header, got %q", got)
		}
	})
}

func TestResponseHeadersForOperation_DeprecatedVersion(t *testing.T) {
	spec := newOpenAPISpec(Config{})

	t.Run("deprecated version documents deprecation headers", func(t *testing.T) {
		headers := spec.responseHeadersForOperation(&operation{
			versionInfo: &VersionConfig{
				Deprecated:   true,
				SunsetTime:   time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC),
				MigrationURL: "https://example.com/migrate",
			},
		})

		if _, ok := headers["Deprecation"]; !ok {
			t.Fatalf("expected Deprecation header, got %v", headers)
		}
		if _, ok := headers["Sunset"]; !ok {
			t.Fatalf("expected Sunset header, got %v", headers)
		}
		if _, ok := headers["Link"]; !ok {
			t.Fatalf("expected Link header, got %v", headers)
		}
	})

	t.Run("deprecated version documents sunset header from compatibility string", func(t *testing.T) {
		headers := spec.responseHeadersForOperation(&operation{
			versionInfo: &VersionConfig{
				Deprecated: true,
				Sunset:     "Wed, 31 Dec 2026 23:59:59 GMT",
			},
		})

		if _, ok := headers["Sunset"]; !ok {
			t.Fatalf("expected Sunset header, got %v", headers)
		}
	})

	t.Run("non-deprecated version does not document deprecation headers", func(t *testing.T) {
		headers := spec.responseHeadersForOperation(&operation{
			versionInfo: &VersionConfig{
				Deprecated:   false,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/migrate",
			},
		})

		if headers != nil {
			t.Fatalf("expected no documented headers, got %v", headers)
		}
	})
}

func TestNormalizeVersionConfig_NormalizesSunsetCompatibilityField(t *testing.T) {
	cfg := normalizeVersionConfig("v1", VersionConfig{
		Sunset: "Wed, 31 Dec 2026 23:59:59 GMT",
	})

	if cfg.SunsetTime.IsZero() {
		t.Fatal("expected SunsetTime to be populated from Sunset")
	}
	if got := cfg.normalizedSunsetHeaderValue(); got != cfg.Sunset {
		t.Fatalf("expected normalized sunset header %q, got %q", cfg.Sunset, got)
	}
}

func TestNormalizeVersionConfig_PrefersSunsetTime(t *testing.T) {
	sunsetTime := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
	cfg := normalizeVersionConfig("v1", VersionConfig{
		Sunset:     "Wed, 31 Dec 2026 23:59:59 GMT",
		SunsetTime: sunsetTime,
	})

	if !cfg.SunsetTime.Equal(sunsetTime) {
		t.Fatalf("expected SunsetTime to be preserved, got %v", cfg.SunsetTime)
	}
	if got := cfg.normalizedSunsetHeaderValue(); got != sunsetTime.Format(http.TimeFormat) {
		t.Fatalf("expected SunsetTime header %q, got %q", sunsetTime.Format(http.TimeFormat), got)
	}
}
