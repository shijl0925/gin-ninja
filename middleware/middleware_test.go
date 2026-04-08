package middleware_test

import (
	"crypto/tls"
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

func TestCORS_CustomConfigDefaults(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS(&middleware.CORSConfig{
		AllowCredentials: true,
	}))
	r.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "http://localhost/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected default allow-origin header, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "PATCH") {
		t.Fatalf("expected default methods, got %q", got)
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

// ---------------------------------------------------------------------------
// I18n middleware
// ---------------------------------------------------------------------------

func TestI18n_SetsLocaleFromHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.I18n())
	r.GET("/", func(c *gin.Context) {
		locale := middleware.GetLocale(c)
		c.String(http.StatusOK, locale)
	})

	cases := []struct {
		header string
		want   string
	}{
		{"en", "en"},
		{"en-US,en;q=0.9", "en"},
		{"zh-CN,zh;q=0.9", "zh"},
		{"zh", "zh"},
		{"fr", "en"}, // unsupported → fallback
		{"", "en"},   // missing → fallback
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.header != "" {
			req.Header.Set("Accept-Language", tc.header)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Body.String() != tc.want {
			t.Errorf("Accept-Language=%q: expected locale %q, got %q", tc.header, tc.want, w.Body.String())
		}
	}
}

func TestI18n_GetLocaleDefault(t *testing.T) {
	r := gin.New()
	// No I18n middleware registered.
	r.GET("/", func(c *gin.Context) {
		locale := middleware.GetLocale(c)
		c.String(http.StatusOK, locale)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "en" {
		t.Errorf("expected default locale en, got %q", w.Body.String())
	}
}


// ---------------------------------------------------------------------------
// SessionMiddleware
// ---------------------------------------------------------------------------

func TestSession_SetAndGet(t *testing.T) {
r := gin.New()
r.Use(middleware.SessionMiddleware(&middleware.SessionConfig{Secret: "test-secret"}))
r.POST("/set", func(c *gin.Context) {
s := middleware.GetSession(c)
s.Set("user_id", "42")
c.Status(http.StatusNoContent)
})
r.GET("/get", func(c *gin.Context) {
s := middleware.GetSession(c)
v, _ := s.Get("user_id")
c.String(http.StatusOK, v)
})

req := httptest.NewRequest(http.MethodPost, "/set", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Code != http.StatusNoContent {
t.Fatalf("expected 204, got %d", w.Code)
}

var sessionCookie *http.Cookie
for _, c := range w.Result().Cookies() {
if c.Name == "session" {
sessionCookie = c
}
}
if sessionCookie == nil {
t.Fatal("expected session cookie to be set")
}
if !sessionCookie.HttpOnly {
	t.Fatal("expected session cookie to default to HttpOnly")
}

req = httptest.NewRequest(http.MethodGet, "/get", nil)
req.AddCookie(sessionCookie)
w = httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Code != http.StatusOK {
t.Fatalf("expected 200, got %d", w.Code)
}
if w.Body.String() != "42" {
t.Errorf("expected user_id=42, got %q", w.Body.String())
}
}

func TestSession_TamperedCookieIsIgnored(t *testing.T) {
r := gin.New()
r.Use(middleware.SessionMiddleware(&middleware.SessionConfig{Secret: "test-secret"}))
r.GET("/", func(c *gin.Context) {
s := middleware.GetSession(c)
v, ok := s.Get("admin")
if ok && v == "true" {
c.String(http.StatusOK, "admin")
} else {
c.String(http.StatusOK, "user")
}
})

req := httptest.NewRequest(http.MethodGet, "/", nil)
req.AddCookie(&http.Cookie{Name: "session", Value: "tampered.invalidsig"})
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Body.String() != "user" {
t.Errorf("tampered session should be ignored, got %q", w.Body.String())
}
}

func TestSession_HTTPOnlyCanBeExplicitlyDisabled(t *testing.T) {
r := gin.New()
r.Use(middleware.SessionMiddleware(&middleware.SessionConfig{
Secret:      "test-secret",
HTTPOnly:    false,
HTTPOnlySet: true,
}))
r.POST("/", func(c *gin.Context) {
s := middleware.GetSession(c)
s.Set("user_id", "42")
c.Status(http.StatusNoContent)
})

req := httptest.NewRequest(http.MethodPost, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

var sessionCookie *http.Cookie
for _, c := range w.Result().Cookies() {
if c.Name == "session" {
sessionCookie = c
}
}
if sessionCookie == nil {
t.Fatal("expected session cookie to be set")
}
if sessionCookie.HttpOnly {
t.Fatal("expected session cookie HttpOnly to remain disabled")
}
}

func TestSession_NilMiddleware_GetSessionReturnsNil(t *testing.T) {
r := gin.New()
r.GET("/", func(c *gin.Context) {
s := middleware.GetSession(c)
if s != nil {
c.String(http.StatusOK, "not-nil")
} else {
c.String(http.StatusOK, "nil")
}
})
req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Body.String() != "nil" {
t.Errorf("expected nil session when middleware is absent")
}
}

func TestSession_PanicOnEmptySecret(t *testing.T) {
defer func() {
if recover() == nil {
t.Fatal("expected panic on empty secret")
}
}()
middleware.SessionMiddleware(&middleware.SessionConfig{Secret: ""})
}

// ---------------------------------------------------------------------------
// CSRF middleware
// ---------------------------------------------------------------------------

func TestCSRF_SafeMethodsSetsToken(t *testing.T) {
r := gin.New()
r.Use(middleware.CSRF(nil))
r.GET("/", func(c *gin.Context) {
token := middleware.CSRFToken(c)
c.String(http.StatusOK, token)
})

req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("expected 200, got %d", w.Code)
}
token := w.Body.String()
if len(token) < 16 {
t.Errorf("expected a token in the response body, got %q", token)
}
var csrfCookie *http.Cookie
for _, c := range w.Result().Cookies() {
if c.Name == "csrf_token" {
csrfCookie = c
}
}
if csrfCookie == nil {
t.Fatal("expected csrf_token cookie to be set")
}
}

func TestCSRF_PostWithValidToken(t *testing.T) {
r := gin.New()
r.Use(middleware.CSRF(nil))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

token := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
req := httptest.NewRequest(http.MethodPost, "/", nil)
req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
req.Header.Set("X-CSRF-Token", token)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
}
}

func TestCSRF_PostWithMissingToken(t *testing.T) {
r := gin.New()
r.Use(middleware.CSRF(nil))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodPost, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusForbidden {
t.Errorf("expected 403 for missing CSRF token, got %d", w.Code)
}
}

func TestCSRF_PostWithWrongToken(t *testing.T) {
r := gin.New()
r.Use(middleware.CSRF(nil))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodPost, "/", nil)
req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "correct-token-12345678901234567890"})
req.Header.Set("X-CSRF-Token", "wrong-token-12345678901234567890")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusForbidden {
t.Errorf("expected 403 for wrong CSRF token, got %d", w.Code)
}
}

// ---------------------------------------------------------------------------
// SecureHeaders middleware
// ---------------------------------------------------------------------------

func TestSecureHeaders_Defaults(t *testing.T) {
r := gin.New()
r.Use(middleware.SecureHeaders(nil))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
t.Errorf("X-Content-Type-Options: expected nosniff, got %q", got)
}
if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
t.Errorf("X-Frame-Options: expected DENY, got %q", got)
}
if got := w.Header().Get("X-XSS-Protection"); got != "1; mode=block" {
t.Errorf("X-XSS-Protection: expected '1; mode=block', got %q", got)
}
if got := w.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
t.Errorf("Referrer-Policy: expected strict-origin-when-cross-origin, got %q", got)
}
}

func TestSecureHeaders_CSP(t *testing.T) {
r := gin.New()
r.Use(middleware.SecureHeaders(&middleware.SecurityConfig{
ContentSecurityPolicy: "default-src 'self'",
}))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if got := w.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
t.Errorf("CSP header: got %q", got)
}
}

func TestSecureHeaders_HSTS_HTTP(t *testing.T) {
r := gin.New()
r.Use(middleware.SecureHeaders(&middleware.SecurityConfig{
HSTSMaxAge: 31536000,
}))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if got := w.Header().Get("Strict-Transport-Security"); got != "" {
t.Errorf("HSTS should not be emitted over HTTP, got %q", got)
}
}

func TestSecureHeaders_HSTS_ForwardedHTTPS(t *testing.T) {
r := gin.New()
r.Use(middleware.SecureHeaders(&middleware.SecurityConfig{
HSTSMaxAge:            31536000,
HSTSIncludeSubDomains: true,
}))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
req.Header.Set("X-Forwarded-Proto", "https")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

got := w.Header().Get("Strict-Transport-Security")
if got == "" {
t.Error("expected HSTS header when X-Forwarded-Proto is https")
}
if !strings.Contains(got, "includeSubDomains") {
t.Errorf("expected includeSubDomains in HSTS header, got %q", got)
}
}

func TestSecureHeaders_TLSAndStrict(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(middleware.SecureHeadersStrict())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
req.TLS = &tls.ConnectionState{}
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
t.Errorf("expected strict frame option DENY, got %q", got)
}
if got := w.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=31536000") || !strings.Contains(got, "includeSubDomains") {
t.Errorf("expected strict HSTS header, got %q", got)
}
}

func TestSecureHeaders_InvalidFrameOptionAndPermissions(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(middleware.SecureHeaders(&middleware.SecurityConfig{
FrameOption:           "ALLOWALL",
PermissionsPolicy:     "geolocation=()",
ContentTypeNoSniff:    false,
XSSProtection:         false,
ReferrerPolicy:        "",
HSTSMaxAge:            31536000,
HSTSPreload:           true,
ContentSecurityPolicy: "default-src 'none'",
}))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
req.TLS = &tls.ConnectionState{}
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if got := w.Header().Get("X-Frame-Options"); got != "" {
t.Errorf("expected invalid frame option to be ignored, got %q", got)
}
if got := w.Header().Get("Permissions-Policy"); got != "geolocation=()" {
t.Errorf("expected permissions policy header, got %q", got)
}
if got := w.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "preload") {
t.Errorf("expected preload in HSTS header, got %q", got)
}
if got := w.Header().Get("Content-Security-Policy"); got != "default-src 'none'" {
t.Errorf("expected CSP header, got %q", got)
}
if got := w.Header().Get("X-Content-Type-Options"); got != "" {
t.Errorf("expected nosniff to be disabled, got %q", got)
}
}

// ---------------------------------------------------------------------------
// UploadLimit middleware
// ---------------------------------------------------------------------------

func TestUploadLimit_DefaultAllows(t *testing.T) {
r := gin.New()
r.Use(middleware.UploadLimit(nil))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

body := strings.NewReader(`{"key":"value"}`)
req := httptest.NewRequest(http.MethodPost, "/", body)
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("expected 200, got %d", w.Code)
}
}

func TestUploadLimit_ExceedsDeclaredSize(t *testing.T) {
r := gin.New()
r.Use(middleware.UploadLimit(&middleware.UploadConfig{MaxSize: 10}))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

body := strings.NewReader(`{"key":"value_that_is_too_long"}`)
req := httptest.NewRequest(http.MethodPost, "/", body)
req.Header.Set("Content-Type", "application/json")
req.ContentLength = 9999
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusRequestEntityTooLarge {
t.Errorf("expected 413, got %d", w.Code)
}
}

func TestUploadLimit_ContentTypeNotAllowed(t *testing.T) {
r := gin.New()
r.Use(middleware.UploadLimit(&middleware.UploadConfig{
MaxSize:          10 << 20,
AllowedMIMETypes: []string{"image/jpeg", "image/png"},
}))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

body := strings.NewReader(`{"key":"value"}`)
req := httptest.NewRequest(http.MethodPost, "/", body)
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusUnsupportedMediaType {
t.Errorf("expected 415, got %d", w.Code)
}
}

func TestUploadLimit_ContentTypeAllowed(t *testing.T) {
r := gin.New()
r.Use(middleware.UploadLimit(&middleware.UploadConfig{
MaxSize:          10 << 20,
AllowedMIMETypes: []string{"application/json", "image/"},
}))
r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

body := strings.NewReader(`{}`)
req := httptest.NewRequest(http.MethodPost, "/", body)
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Code != http.StatusOK {
t.Errorf("expected 200 for allowed type, got %d", w.Code)
}

req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))
req.Header.Set("Content-Type", "image/png")
w = httptest.NewRecorder()
r.ServeHTTP(w, req)
if w.Code != http.StatusOK {
t.Errorf("expected 200 for image/png via prefix, got %d", w.Code)
}
}

func TestUploadLimit_GetRequestNotAffected(t *testing.T) {
r := gin.New()
r.Use(middleware.UploadLimit(&middleware.UploadConfig{
MaxSize:          1,
AllowedMIMETypes: []string{"text/plain"},
}))
r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

req := httptest.NewRequest(http.MethodGet, "/", nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("GET should not be affected by upload limit, got %d", w.Code)
}
}
