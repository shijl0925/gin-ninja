package middleware

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS policy settings.
type CORSConfig struct {
	// AllowOrigins is the list of allowed origin patterns.
	// Use ["*"] to allow all origins (not recommended for production with credentials).
	AllowOrigins []string
	// AllowMethods is the list of allowed HTTP methods.
	// Defaults to common REST methods if empty.
	AllowMethods []string
	// AllowHeaders is the list of headers the client may include.
	AllowHeaders []string
	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool
	// MaxAgeSecs is the max age (seconds) for preflight response caching.
	MaxAgeSecs int
}

// CORS returns a gin middleware that applies the supplied CORS policy.
// If cfg is nil, a permissive default policy (allow all origins) suitable for
// development is used.  Passing nil in production (gin.ReleaseMode) emits a
// warning to the standard logger; supply an explicit CORSConfig instead.
//
//	api.Engine().Use(middleware.CORS(nil))
//	api.Engine().Use(middleware.CORS(&middleware.CORSConfig{
//	    AllowOrigins: []string{"https://example.com"},
//	    AllowCredentials: true,
//	}))
func CORS(cfg *CORSConfig) gin.HandlerFunc {
	c := cors.DefaultConfig()

	if cfg == nil {
		if gin.Mode() == gin.ReleaseMode {
			log.Println("[gin-ninja] WARNING: middleware.CORS(nil) allows all origins – " +
				"provide an explicit CORSConfig in production")
		}
		c.AllowAllOrigins = true
		c.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		c.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
		return cors.New(c)
	}

	if len(cfg.AllowOrigins) == 0 {
		c.AllowAllOrigins = true
	} else {
		c.AllowOrigins = cfg.AllowOrigins
	}

	if len(cfg.AllowMethods) > 0 {
		c.AllowMethods = cfg.AllowMethods
	} else {
		c.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}

	if len(cfg.AllowHeaders) > 0 {
		c.AllowHeaders = cfg.AllowHeaders
	} else {
		c.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
	}

	c.AllowCredentials = cfg.AllowCredentials

	if cfg.MaxAgeSecs > 0 {
		c.MaxAge = time.Duration(cfg.MaxAgeSecs) * time.Second
	}

	return cors.New(c)
}
