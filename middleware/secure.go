package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds individual security-header settings.
// All fields are opt-in: zero/false values disable the corresponding header.
type SecurityConfig struct {
	// ContentTypeNoSniff sets `X-Content-Type-Options: nosniff`.
	ContentTypeNoSniff bool
	// FrameOption controls the `X-Frame-Options` header.
	//   "DENY"        – no framing at all (most restrictive)
	//   "SAMEORIGIN"  – allow framing from the same origin
	//   ""            – header is not emitted
	FrameOption string
	// XSSProtection sets `X-XSS-Protection: 1; mode=block`.
	// Deprecated: X-XSS-Protection is no longer supported by modern browsers
	// and has been removed from Chrome, Firefox, and Edge.  It can introduce
	// security vulnerabilities in older browsers.  Use ContentSecurityPolicy
	// instead.  This field defaults to false in DefaultSecurityConfig.
	XSSProtection bool
	// ReferrerPolicy sets the `Referrer-Policy` header.
	// Common values: "no-referrer", "strict-origin-when-cross-origin".
	// If empty the header is not emitted.
	ReferrerPolicy string
	// HSTSMaxAge sets the `Strict-Transport-Security` max-age in seconds.
	// A value of 0 disables the header.
	HSTSMaxAge int
	// HSTSIncludeSubDomains appends `; includeSubDomains` to the HSTS header.
	HSTSIncludeSubDomains bool
	// HSTSPreload appends `; preload` to the HSTS header.
	HSTSPreload bool
	// TrustForwardedProto controls whether the X-Forwarded-Proto request header
	// is trusted when deciding to emit the Strict-Transport-Security header.
	// Set this to true only when the application is deployed behind a trusted
	// reverse proxy that sets the header reliably.  Defaults to false.
	// When false, HSTS is only emitted for connections where TLS is terminated
	// by this process (c.Request.TLS != nil).
	TrustForwardedProto bool
	// ContentSecurityPolicy sets the `Content-Security-Policy` header value.
	// Empty string disables it.
	ContentSecurityPolicy string
	// PermissionsPolicy sets the `Permissions-Policy` header value.
	// Empty string disables it.
	PermissionsPolicy string
}

// DefaultSecurityConfig returns a SecurityConfig with sensible defaults
// suitable for most web APIs.
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ContentTypeNoSniff:    true,
		FrameOption:           "DENY",
		XSSProtection:         false, // deprecated; use ContentSecurityPolicy instead
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            0, // disabled by default; set to 31536000 for production HTTPS
		HSTSIncludeSubDomains: false,
		HSTSPreload:           false,
		TrustForwardedProto:   false,
	}
}

// SecureHeaders returns a gin middleware that sets security-focused HTTP
// response headers according to the supplied configuration.  Pass nil to use
// DefaultSecurityConfig().
//
//	api.UseGin(middleware.SecureHeaders(nil))
//
//	api.UseGin(middleware.SecureHeaders(&middleware.SecurityConfig{
//	    ContentTypeNoSniff:    true,
//	    FrameOption:           "SAMEORIGIN",
//	    HSTSMaxAge:            31536000,
//	    ContentSecurityPolicy: "default-src 'self'",
//	}))
func SecureHeaders(cfg *SecurityConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultSecurityConfig()
	}

	// Pre-compute the HSTS header value once.
	hstsValue := ""
	if cfg.HSTSMaxAge > 0 {
		hstsValue = fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubDomains {
			hstsValue += "; includeSubDomains"
		}
		if cfg.HSTSPreload {
			hstsValue += "; preload"
		}
	}

	return func(c *gin.Context) {
		w := c.Writer

		if cfg.ContentTypeNoSniff {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		if fo := cfg.FrameOption; fo == "DENY" || fo == "SAMEORIGIN" {
			w.Header().Set("X-Frame-Options", fo)
		}

		if cfg.XSSProtection {
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		if cfg.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
		}

		if hstsValue != "" {
			// Only emit HSTS over HTTPS connections.  When TrustForwardedProto is
			// enabled the X-Forwarded-Proto header from a trusted reverse proxy is
			// also accepted; leave it disabled (default) to prevent HSTS poisoning
			// by unauthenticated clients that can set arbitrary request headers.
			if c.Request.TLS != nil || (cfg.TrustForwardedProto && forwardedProtoIsHTTPS(c.GetHeader("X-Forwarded-Proto"))) {
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}
		}

		if cfg.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
		}

		if cfg.PermissionsPolicy != "" {
			w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
		}

		c.Next()
	}
}

// forwardedProtoIsHTTPS reports whether a proxy chain contains HTTPS in a
// comma-separated X-Forwarded-Proto header, ignoring case and whitespace.
func forwardedProtoIsHTTPS(value string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "https") {
			return true
		}
	}
	return false
}

// SecureHeadersStrict is a convenience wrapper that returns SecureHeaders with
// a strict configuration suitable for production HTTPS deployments:
//   - All basic security headers enabled
//   - HSTS with a 1-year max-age + includeSubDomains
//   - TrustForwardedProto enabled (assumes a trusted reverse proxy in front)
func SecureHeadersStrict() gin.HandlerFunc {
	return SecureHeaders(&SecurityConfig{
		ContentTypeNoSniff:    true,
		FrameOption:           "DENY",
		XSSProtection:         false,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubDomains: true,
		TrustForwardedProto:   true,
	})
}
