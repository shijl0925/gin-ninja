package middleware

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestAuthHelperFallbacksAndCopies(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if AuthScopesKey() != authScopesKey {
		t.Fatalf("unexpected scopes key")
	}
	if principal := GetAuthPrincipal(c); principal != nil {
		t.Fatalf("expected nil principal, got %v", principal)
	}
	if scopes := GetAuthScopes(c); scopes != nil {
		t.Fatalf("expected nil scopes, got %v", scopes)
	}

	claims := &Claims{UserID: 12, Username: "fallback"}
	c.Set(claimsKey, claims)
	if principal := GetAuthPrincipal(c); principal != claims {
		t.Fatalf("expected claims fallback principal, got %v", principal)
	}

	c.Set(authScopesKey, "wrong-type")
	if scopes := GetAuthScopes(c); len(scopes) != 0 {
		t.Fatalf("expected wrong-type scopes to return empty copy, got %v", scopes)
	}

	setAuthScopes(c, []string{"read"})
	scopes := GetAuthScopes(c)
	scopes[0] = "mutated"
	if got := GetAuthScopes(c)[0]; got != "read" {
		t.Fatalf("expected scopes to be copied, got %q", got)
	}
}

func TestAuthMiddlewareFailureBranchesAndPanics(t *testing.T) {
	assertPanics(t, func() { APIKeyHeader("X-Key", nil) })
	assertPanics(t, func() { HTTPBasicAuth("", nil) })
	assertPanics(t, func() { HTTPBearerAuth(nil) })
	assertPanics(t, func() { OAuth2BearerAuthWithScopes(nil, nil) })

	t.Run("api key cookie missing", func(t *testing.T) {
		r := gin.New()
		r.Use(APIKeyCookie("api_key", func(_ *gin.Context, key string) (any, bool) {
			return "user", key == "secret"
		}))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := performRequest(r, http.MethodGet, "/", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("basic default realm invalid credentials", func(t *testing.T) {
		r := gin.New()
		r.Use(HTTPBasicAuth("", func(_ *gin.Context, username, password string) (any, bool) {
			return username, username == "alice" && password == "secret"
		}))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("alice", "wrong")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != `Basic realm="Restricted"` {
			t.Fatalf("expected default basic challenge, got %d headers=%v", w.Code, w.Header())
		}
	})

	t.Run("bearer invalid token", func(t *testing.T) {
		r := gin.New()
		r.Use(HTTPBearerAuth(func(_ *gin.Context, token string) (any, bool) {
			return "user", token == "valid"
		}))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatalf("expected bearer unauthorized, got %d headers=%v", w.Code, w.Header())
		}
	})
}

func TestOAuth2BearerAuthMissingInvalidAndEmptyScopes(t *testing.T) {
	t.Run("missing token includes required scope challenge", func(t *testing.T) {
		r := gin.New()
		r.Use(OAuth2BearerAuthWithScopes([]string{"read"}, func(*gin.Context, string) (any, []string, bool) {
			return nil, nil, false
		}))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := performRequest(r, http.MethodGet, "/", nil)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Header().Get("WWW-Authenticate"), `scope="read"`) {
			t.Fatalf("expected scope challenge, got %d headers=%v", w.Code, w.Header())
		}
	})

	t.Run("invalid token includes error challenge", func(t *testing.T) {
		r := gin.New()
		r.Use(OAuth2BearerAuthWithScopes([]string{"read"}, func(*gin.Context, string) (any, []string, bool) {
			return nil, nil, false
		}))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
			t.Fatalf("expected invalid token challenge, got %d headers=%v", w.Code, w.Header())
		}
	})

	t.Run("empty required scopes succeeds", func(t *testing.T) {
		r := gin.New()
		r.Use(OAuth2BearerAuthWithScopes(nil, func(_ *gin.Context, token string) (any, []string, bool) {
			return "user", nil, token == "valid"
		}))
		r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, GetAuthPrincipal(c).(string)) })

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != "user" {
			t.Fatalf("expected success without required scopes, got %d %s", w.Code, w.Body.String())
		}
	})
}

func TestJWTAdditionalBranches(t *testing.T) {
	assertPanics(t, func() { JWTAuthWithSecret("") })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(claimsKey, "wrong-type")
	if claims := GetClaims(c); claims != nil {
		t.Fatalf("expected nil claims for wrong type, got %v", claims)
	}

	token, err := generateToken(1, "issuer-default", "secret", time.Hour, "")
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(*jwt.Token) (any, error) {
		return []byte("secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("expected token to parse, err=%v", err)
	}
	if claims := parsed.Claims.(*Claims); claims.Issuer != "gin-ninja" {
		t.Fatalf("expected default issuer, got %q", claims.Issuer)
	}

	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{Username: "none"})
	signed, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}
	r := gin.New()
	r.Use(JWTAuthWithSecret("secret"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected non-HMAC token to be rejected, got %d", w.Code)
	}
}

func TestCORSReleaseAllowAllExplicitConfig(t *testing.T) {
	withGinMode(t, gin.ReleaseMode)

	r := gin.New()
	r.Use(CORS(&CORSConfig{AllowOrigins: []string{"*"}}))
	r.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "http://localhost/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected allow-all origin, got %q", got)
	}
}

func TestRequestIDValidationAndGetterFallbacks(t *testing.T) {
	for _, id := range []string{"bad value", strings.Repeat("a", maxRequestIDLen+1)} {
		if isValidRequestID(id) {
			t.Fatalf("expected request id %q to be invalid", id)
		}
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(requestIDKey, 123)
	if got := GetRequestID(c); got != "" {
		t.Fatalf("expected empty request id for wrong type, got %q", got)
	}

	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, GetRequestID(c)) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDKey, "bad value")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() == "bad value" || len(w.Body.String()) != 32 {
		t.Fatalf("expected generated id for invalid header, got %q", w.Body.String())
	}
}

func TestGetLocaleWrongTypeAndEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set(LocaleKey(), 123)
	if got := GetLocale(c); got != "en" {
		t.Fatalf("expected default locale for wrong type, got %q", got)
	}
	c.Set(LocaleKey(), "")
	if got := GetLocale(c); got != "en" {
		t.Fatalf("expected default locale for empty string, got %q", got)
	}
}

func TestCSRFAdditionalBranches(t *testing.T) {
	t.Run("secret generation panics when random fails", func(t *testing.T) {
		original := csrfRandRead
		csrfRandRead = func([]byte) (int, error) {
			return 0, errors.New("entropy unavailable")
		}
		t.Cleanup(func() { csrfRandRead = original })

		assertPanics(t, func() { _ = generateCSRFSecret() })
	})

	t.Run("form token fallback and custom error handler", func(t *testing.T) {
		r := gin.New()
		r.Use(CSRF(&CSRFConfig{
			Secret:    "csrf-secret",
			FieldName: "csrf",
			ErrorHandler: func(c *gin.Context) {
				c.String(http.StatusTeapot, "custom csrf")
			},
		}))
		r.GET("/token", func(c *gin.Context) { c.String(http.StatusOK, CSRFToken(c)) })
		r.POST("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		tokenResp := performRequest(r, http.MethodGet, "/token", nil)
		csrfCookie := findCookie(tokenResp, "csrf_token")
		if csrfCookie == nil {
			t.Fatal("expected csrf cookie")
		}

		formReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("csrf="+tokenResp.Body.String()))
		formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		formReq.AddCookie(csrfCookie)
		formResp := httptest.NewRecorder()
		r.ServeHTTP(formResp, formReq)
		if formResp.Code != http.StatusNoContent {
			t.Fatalf("expected form token to pass, got %d %s", formResp.Code, formResp.Body.String())
		}

		badReq := httptest.NewRequest(http.MethodPost, "/", nil)
		badResp := httptest.NewRecorder()
		r.ServeHTTP(badResp, badReq)
		if badResp.Code != http.StatusTeapot || badResp.Body.String() != "custom csrf" {
			t.Fatalf("expected custom csrf error, got %d %q", badResp.Code, badResp.Body.String())
		}
	})
}

func TestSessionInternalErrorBranches(t *testing.T) {
	t.Run("ensure saved noops", func(t *testing.T) {
		var nilWriter *sessionResponseWriter
		nilWriter.ensureSessionSaved()
		(&sessionResponseWriter{}).ensureSessionSaved()
		(&sessionResponseWriter{persisted: true, session: &Session{dirty: true}}).ensureSessionSaved()
		(&sessionResponseWriter{session: &Session{dirty: false}}).ensureSessionSaved()
	})

	t.Run("save rejects oversized cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		s := &Session{
			cfg:  &SessionConfig{CookieName: "session", Path: "/", Secret: "secret", MaxAge: 60},
			data: map[string]string{"large": strings.Repeat("x", maxCookieValueLen)},
		}
		if err := s.Save(c); err == nil {
			t.Fatal("expected oversized session save error")
		}
	})

	t.Run("set initializes nil data", func(t *testing.T) {
		s := &Session{}
		s.Set("key", "value")
		if got, ok := s.Get("key"); !ok || got != "value" {
			t.Fatalf("expected set to initialize data, got %q ok=%v", got, ok)
		}
	})

	t.Run("decode error cases", func(t *testing.T) {
		for _, value := range []string{
			"missing-separator",
			"payload.bad-signature",
			signedSessionPayload("%%%", "secret"),
			signedSessionPayload(base64.RawURLEncoding.EncodeToString([]byte("not json")), "secret"),
			signedSessionPayload(base64.RawURLEncoding.EncodeToString([]byte(`{"data":{},"expires_at":0}`)), "secret"),
		} {
			if _, err := decodeSession(value, "secret", time.Now()); err == nil {
				t.Fatalf("expected decode error for %q", value)
			}
		}
	})

	t.Run("decode nil data becomes empty map", func(t *testing.T) {
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"expires_at":4102444800}`))
		data, err := decodeSession(signedSessionPayload(payload, "secret"), "secret", time.Now())
		if err != nil {
			t.Fatalf("decodeSession: %v", err)
		}
		if data == nil || len(data) != 0 {
			t.Fatalf("expected empty data map, got %#v", data)
		}
	})
}

func TestLoggerInfoAndWarnBranches(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(Logger(logger))
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/missing", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	performRequest(r, http.MethodGet, "/ok", nil)
	performRequest(r, http.MethodGet, "/missing", nil)

	if logs.Len() != 2 {
		t.Fatalf("expected 2 log entries, got %d", logs.Len())
	}
	if logs.All()[0].Level != zap.InfoLevel {
		t.Fatalf("expected info log, got %s", logs.All()[0].Level)
	}
	if logs.All()[1].Level != zap.WarnLevel {
		t.Fatalf("expected warn log, got %s", logs.All()[1].Level)
	}
}

func TestSecureHeadersAdditionalBranches(t *testing.T) {
	if forwardedProtoIsHTTPS("") {
		t.Fatal("expected empty forwarded proto to be false")
	}
	if forwardedProtoIsHTTPS("http, ws") {
		t.Fatal("expected forwarded proto without https to be false")
	}

	r := gin.New()
	r.Use(SecureHeaders(&SecurityConfig{
		FrameOption:           "SAMEORIGIN",
		HSTSMaxAge:            10,
		HSTSIncludeSubDomains: true,
		HSTSPreload:           true,
	}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("expected SAMEORIGIN, got %q", got)
	}
	if got := w.Header().Get("X-XSS-Protection"); got != "1; mode=block" {
		t.Fatalf("expected XSS protection header, got %q", got)
	}
	if got := w.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "includeSubDomains") || !strings.Contains(got, "preload") {
		t.Fatalf("expected full HSTS header, got %q", got)
	}
}

func signedSessionPayload(payload, secret string) string {
	return payload + "." + sessionHMAC(payload, secret)
}

func performRequest(r http.Handler, method, path string, body *strings.Reader) *httptest.ResponseRecorder {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func withGinMode(t *testing.T, mode string) {
	t.Helper()
	previous := gin.Mode()
	gin.SetMode(mode)
	t.Cleanup(func() {
		gin.SetMode(previous)
	})
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
