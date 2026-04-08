package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UploadConfig holds configuration for the upload-limiting middleware.
type UploadConfig struct {
	// MaxSize is the maximum allowed Content-Length in bytes.
	// Requests that declare a larger Content-Length are rejected with 413.
	// Requests without a Content-Length header are allowed through but the body
	// is wrapped with http.MaxBytesReader so the limit is still enforced.
	// Defaults to 10 MiB (10 << 20) when 0.
	MaxSize int64
	// AllowedMIMETypes is an optional whitelist of accepted Content-Type values
	// for requests that carry a body (POST, PUT, PATCH).
	// Each entry is matched as a prefix, so "image/" matches "image/jpeg",
	// "image/png", etc.
	// A nil or empty slice disables content-type checking.
	AllowedMIMETypes []string
	// ErrorHandler is an optional custom handler invoked when the request is
	// rejected.  The default handler writes an appropriate JSON error.
	ErrorHandler func(c *gin.Context, status int, code, message string)
}

const defaultMaxUploadSize = 10 << 20 // 10 MiB

// UploadLimit returns a gin middleware that enforces upload size limits and an
// optional content-type whitelist.
//
//	api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
//	    MaxSize:          5 << 20,   // 5 MiB
//	    AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
//	}))
//
// Pass nil to use defaults (10 MiB limit, no content-type checking).
func UploadLimit(cfg *UploadConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = &UploadConfig{}
	}
	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = defaultMaxUploadSize
	}

	allowedTypes := make([]string, len(cfg.AllowedMIMETypes))
	for i, t := range cfg.AllowedMIMETypes {
		allowedTypes[i] = strings.ToLower(strings.TrimSpace(t))
	}

	errHandler := cfg.ErrorHandler
	if errHandler == nil {
		errHandler = defaultUploadErrorHandler
	}

	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)
		hasBody := method == http.MethodPost ||
			method == http.MethodPut ||
			method == http.MethodPatch

		if !hasBody {
			c.Next()
			return
		}

		// 1. Check declared Content-Length.
		if cl := c.Request.ContentLength; cl > maxSize {
			errHandler(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				fmt.Sprintf("request body must not exceed %d bytes", maxSize))
			return
		}

		// 2. Wrap body with MaxBytesReader to enforce the limit even when no
		//    Content-Length header is present.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		// 3. Content-type whitelist check.
		if len(allowedTypes) > 0 {
			ct := strings.ToLower(c.ContentType())
			// Strip parameters (e.g. charset=utf-8).
			if idx := strings.IndexByte(ct, ';'); idx >= 0 {
				ct = strings.TrimSpace(ct[:idx])
			}
			if !isAllowedMIMEType(ct, allowedTypes) {
				errHandler(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
					fmt.Sprintf("content type %q is not allowed", ct))
				return
			}
		}

		c.Next()
	}
}

func defaultUploadErrorHandler(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// isAllowedMIMEType checks whether ct is covered by any of the allowed type
// patterns.  A pattern ending with "/" is treated as a prefix (wildcard).
func isAllowedMIMEType(ct string, allowed []string) bool {
	for _, a := range allowed {
		if strings.HasSuffix(a, "/") {
			// Prefix match: "image/" matches "image/jpeg", "image/png" etc.
			if strings.HasPrefix(ct, a) {
				return true
			}
		} else {
			if ct == a {
				return true
			}
		}
	}
	return false
}
