package ninja

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
