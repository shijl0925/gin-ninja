package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "X-Request-ID"

// RequestID injects a unique identifier into every request.  If the client
// already sends an X-Request-ID header its value is reused; otherwise a new
// 16-byte random hex string is generated.
//
// The ID is stored in the Gin context under the key "X-Request-ID" and is
// also written to the response header so clients can correlate log entries.
//
//	api.Engine().Use(middleware.RequestID())
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDKey)
		if id == "" {
			id = generateID()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDKey, id)
		c.Next()
	}
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
