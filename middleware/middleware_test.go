package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/settings"
)

func init() { gin.SetMode(gin.TestMode) }

// ---------------------------------------------------------------------------
// RequestID middleware
// ---------------------------------------------------------------------------

func TestRequestID_GeneratesID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		id := middleware.GetRequestID(c)
		if id == "" {
			t.Error("expected request ID to be set")
		}
		c.String(http.StatusOK, id)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
}

func TestRequestID_ReusesClientID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, middleware.GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "my-custom-id" {
		t.Errorf("expected my-custom-id, got %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// JWT middleware
// ---------------------------------------------------------------------------

func newJWTEngine(secret string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWTAuthWithSecret(secret))
	r.GET("/protected", func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no claims"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID, "username": claims.Username})
	})
	return r
}

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-secret-key-123"
	token, err := middleware.GenerateTokenWithSecret(42, "alice", secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	r := newJWTEngine(secret)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body) //nolint:errcheck
	if uint(body["user_id"].(float64)) != 42 {
		t.Errorf("expected user_id=42, got %v", body["user_id"])
	}
}

func TestJWTAuth_MissingToken(t *testing.T) {
	r := newJWTEngine("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	r := newJWTEngine("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	token, _ := middleware.GenerateTokenWithSecret(1, "bob", "correct-secret", time.Hour)
	r := newJWTEngine("wrong-secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CORS middleware
// ---------------------------------------------------------------------------

func TestCORS_DefaultAllowsAll(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS(nil))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Use http.NewRequest so the Host header is set correctly.
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Errorf("expected Access-Control-Allow-Origin header in response, got headers: %v", w.Header())
	}
}

func TestCORS_CustomConfig(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS(&middleware.CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Authorization"},
		AllowCredentials: true,
		MaxAgeSecs:       60,
	}))
	r.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req, _ := http.NewRequest(http.MethodOptions, "http://localhost/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header, got %q", got)
	}
}

func TestJWTAuth_UsesGlobalSettings(t *testing.T) {
	prev := settings.Global.JWT
	t.Cleanup(func() { settings.Global.JWT = prev })

	settings.Global.JWT.Secret = "global-secret"
	settings.Global.JWT.ExpireHours = 1
	settings.Global.JWT.Issuer = "test-issuer"

	token, err := middleware.GenerateToken(99, "global-user")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	r := gin.New()
	r.Use(middleware.JWTAuth())
	r.GET("/protected", func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": claims.UserID,
			"key":     middleware.ClaimsKey(),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), middleware.ClaimsKey()) {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}

func TestGenerateTokenWithSecret_EmptySecret(t *testing.T) {
	if _, err := middleware.GenerateTokenWithSecret(1, "alice", "", time.Hour); err == nil {
		t.Fatal("expected empty secret error")
	}
}

func TestGenerateTokenWithSecret_DoesNotUseGlobalIssuer(t *testing.T) {
	prev := settings.Global.JWT
	t.Cleanup(func() { settings.Global.JWT = prev })

	settings.Global.JWT.Issuer = "global-issuer"
	token, err := middleware.GenerateTokenWithSecret(1, "alice", "secret", time.Hour)
	if err != nil {
		t.Fatalf("GenerateTokenWithSecret: %v", err)
	}

	parsed, err := jwt.ParseWithClaims(token, &middleware.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("expected token to parse, err=%v valid=%v", err, parsed != nil && parsed.Valid)
	}

	claims, ok := parsed.Claims.(*middleware.Claims)
	if !ok {
		t.Fatalf("unexpected claims type: %T", parsed.Claims)
	}
	if claims.Issuer != "gin-ninja" {
		t.Fatalf("expected default issuer, got %q", claims.Issuer)
	}
}

func TestJWTAuth_RequireRolesPermissionsAndScopes(t *testing.T) {
	secret := "authz-secret"
	token, err := middleware.GenerateTokenWithSecretAndClaims(middleware.Claims{
		UserID:      7,
		Username:    "admin",
		Roles:       []string{"admin", "editor"},
		Permissions: []string{"users:read", "users:write"},
		Scopes:      []string{"profile", "users"},
	}, secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateTokenWithSecretAndClaims: %v", err)
	}

	r := gin.New()
	r.Use(middleware.JWTAuthWithSecret(secret))
	r.Use(middleware.RequireRoles("admin"))
	r.Use(middleware.RequirePermissions("users:read"))
	r.Use(middleware.RequireScopes("users"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTAuth_RequireRolesPermissionsAndScopes_Denied(t *testing.T) {
	secret := "authz-secret"
	token, err := middleware.GenerateTokenWithSecretAndClaims(middleware.Claims{
		UserID:      7,
		Username:    "member",
		Roles:       []string{"member"},
		Permissions: []string{"users:read"},
		Scopes:      []string{"profile"},
	}, secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateTokenWithSecretAndClaims: %v", err)
	}

	tests := []struct {
		name     string
		use      gin.HandlerFunc
		wantCode int
	}{
		{name: "missing claims", use: middleware.RequireRoles("admin"), wantCode: http.StatusUnauthorized},
		{name: "missing role", use: middleware.RequireRoles("admin"), wantCode: http.StatusForbidden},
		{name: "missing permission", use: middleware.RequirePermissions("users:write"), wantCode: http.StatusForbidden},
		{name: "missing scope", use: middleware.RequireScopes("users"), wantCode: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			if tc.name != "missing claims" {
				r.Use(middleware.JWTAuthWithSecret(secret))
			}
			r.Use(tc.use)
			r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.name != "missing claims" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}
