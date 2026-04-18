package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"unicode"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "X-Request-ID"

// maxRequestIDLen is the maximum number of characters accepted from a
// client-supplied X-Request-ID header.  Values longer than this are rejected
// and a server-generated ID is used instead.
const maxRequestIDLen = 128

// RequestID injects a unique identifier into every request.  If the client
// already sends an X-Request-ID header its value is reused – provided it
// passes the basic safety checks (max 128 chars, alphanumeric / hyphens /
// underscores only).  Otherwise a new 16-byte random hex string is generated.
//
// The ID is stored in the Gin context under the key "X-Request-ID" and is
// also written to the response header so clients can correlate log entries.
//
//	api.Engine().Use(middleware.RequestID())
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDKey)
		if id == "" || !isValidRequestID(id) {
			id = generateID()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDKey, id)
		c.Next()
	}
}

// isValidRequestID returns true when id is a non-empty string whose length is
// within maxRequestIDLen and whose characters are all alphanumeric, hyphens,
// or underscores.  This prevents log-injection and HTTP-header-injection attacks.
func isValidRequestID(id string) bool {
	if len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

// GetRequestID retrieves the request ID stored by the RequestID middleware.
// Returns an empty string if the middleware was not registered.
func GetRequestID(c *gin.Context) string {
	id, _ := c.Get(requestIDKey)
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
