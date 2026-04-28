package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/settings"
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

func TestJWTAuthWithConfigDoesNotUseGlobalSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prev := settings.GetGlobal()
	t.Cleanup(func() { settings.SetGlobal(prev) })
	settings.SetGlobal(settings.Config{JWT: settings.JWTConfig{Secret: "wrong-secret"}})

	cfg := settings.JWTConfig{Secret: "explicit-secret", ExpireHours: 1, Issuer: "explicit"}
	token, err := GenerateTokenWithConfig(7, "explicit-user", cfg)
	if err != nil {
		t.Fatalf("GenerateTokenWithConfig: %v", err)
	}

	r := gin.New()
	r.Use(JWTAuthWithConfig(cfg))
	r.GET("/", func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			t.Fatal("expected claims")
		}
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}
