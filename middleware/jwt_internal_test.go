package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		header string
		want   string
	}{
		{name: "standard", header: "Bearer token", want: "token"},
		{name: "mixed case and extra spaces", header: "bEaReR   token", want: "token"},
		{name: "tab separator", header: "Bearer\tvalue", want: "value"},
		{name: "missing token", header: "Bearer", want: ""},
		{name: "wrong scheme", header: "Basic abc", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Authorization", tc.header)
			if got := extractBearerToken(c); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
