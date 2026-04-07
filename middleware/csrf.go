package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var csrfRandRead = rand.Read

// CSRFConfig holds configuration for the CSRF middleware.
type CSRFConfig struct {
	// CookieName is the name of the cookie that stores the CSRF token.
	// Defaults to "csrf_token".
	CookieName string
	// HeaderName is the request header name the client must send the token in.
	// Defaults to "X-CSRF-Token".
	HeaderName string
	// FieldName is the form field name the client may alternatively use.
	// Defaults to "csrf_token".
	FieldName string
	// CookieMaxAge is the cookie lifetime in seconds. Defaults to 43200 (12 h).
	CookieMaxAge int
	// CookieSecure marks the cookie as Secure (HTTPS only).
	CookieSecure bool
	// CookieHTTPOnly controls whether the cookie is accessible to JavaScript.
	// Note: set to false if the client reads the token via JavaScript (common
	// for SPA + same-origin API setups).  Defaults to false.
	CookieHTTPOnly bool
	// CookieSameSite controls the SameSite attribute of the token cookie.
	// Defaults to http.SameSiteStrictMode.
	CookieSameSite http.SameSite
	// IgnoreMethods lists HTTP methods that do not require a CSRF token.
	// Defaults to GET, HEAD, OPTIONS, TRACE.
	IgnoreMethods []string
	// ErrorHandler is an optional custom handler invoked when the token is
	// invalid.  The default handler writes HTTP 403 with a JSON body.
	ErrorHandler gin.HandlerFunc
}

func (cfg *CSRFConfig) withDefaults() *CSRFConfig {
	out := *cfg
	if out.CookieName == "" {
		out.CookieName = "csrf_token"
	}
	if out.HeaderName == "" {
		out.HeaderName = "X-CSRF-Token"
	}
	if out.FieldName == "" {
		out.FieldName = "csrf_token"
	}
	if out.CookieMaxAge == 0 {
		out.CookieMaxAge = 43200
	}
	if out.CookieSameSite == 0 {
		out.CookieSameSite = http.SameSiteStrictMode
	}
	if len(out.IgnoreMethods) == 0 {
		out.IgnoreMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
			http.MethodTrace,
		}
	}
	if out.ErrorHandler == nil {
		out.ErrorHandler = defaultCSRFErrorHandler
	}
	return &out
}

func defaultCSRFErrorHandler(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"code":    "CSRF_TOKEN_INVALID",
			"message": "invalid or missing CSRF token",
		},
	})
}

// csrfTokenKey is the gin context key under which the current CSRF token is stored.
const csrfTokenKey = "gin_ninja_csrf_token"

// CSRF returns a gin middleware that implements the double-submit cookie
// pattern for CSRF protection.
//
// For safe methods (GET, HEAD, OPTIONS, TRACE) a fresh token is generated and
// set as a cookie so the client can read it for subsequent state-changing
// requests.
//
// For all other methods the middleware validates that the cookie value matches
// the token submitted in the X-CSRF-Token request header (or the configured
// form field).  If the token is missing or invalid the request is aborted with
// HTTP 403.
//
//	api.UseGin(middleware.CSRF(nil))                      // defaults
//	api.UseGin(middleware.CSRF(&middleware.CSRFConfig{    // custom
//	    CookieSecure: true,
//	}))
//
// Read the active token in a handler or template:
//
//	token := middleware.CSRFToken(c)
func CSRF(cfg *CSRFConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = &CSRFConfig{}
	}
	cfg = cfg.withDefaults()

	ignoredSet := make(map[string]bool, len(cfg.IgnoreMethods))
	for _, m := range cfg.IgnoreMethods {
		ignoredSet[strings.ToUpper(m)] = true
	}

	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)

		// Read (or generate) the token from the cookie.
		token, err := c.Cookie(cfg.CookieName)
		if err != nil || token == "" {
			token = generateCSRFToken()
			setCSRFCookie(c, cfg, token)
		}

		c.Set(csrfTokenKey, token)

		// Safe methods: just ensure the cookie exists, then proceed.
		if ignoredSet[method] {
			c.Next()
			return
		}

		// State-changing methods: validate the submitted token.
		submitted := c.GetHeader(cfg.HeaderName)
		if submitted == "" {
			// Try form field as fallback.
			submitted = c.PostForm(cfg.FieldName)
		}

		if !csrfTokensEqual(token, submitted) {
			cfg.ErrorHandler(c)
			return
		}

		c.Next()
	}
}

// CSRFToken returns the CSRF token stored by the CSRF middleware for this
// request.  Returns an empty string if the middleware is not registered.
func CSRFToken(c *gin.Context) string {
	v, _ := c.Get(csrfTokenKey)
	s, _ := v.(string)
	return s
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := csrfRandRead(b); err != nil {
		panic("csrf: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func setCSRFCookie(c *gin.Context, cfg *CSRFConfig, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cfg.CookieMaxAge,
		Secure:   cfg.CookieSecure,
		HttpOnly: cfg.CookieHTTPOnly,
		SameSite: cfg.CookieSameSite,
	})
}

// csrfTokensEqual performs a constant-time comparison to prevent timing attacks.
func csrfTokensEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
