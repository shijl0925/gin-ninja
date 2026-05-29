package middleware

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/shijl0925/gin-ninja/settings"
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
// Prefer CORSFromConfig with settings.CORSConfig for applications that load
// settings from config files. If cfg is nil, a permissive default policy
// (allow all origins) suitable for development is used. Passing nil in
// production (gin.ReleaseMode) panics; supply an explicit CORSConfig instead.
//
//	api.Engine().Use(middleware.CORSFromConfig(cfg.CORS))
//	api.Engine().Use(middleware.CORS(&middleware.CORSConfig{
//	    AllowOrigins: []string{"https://example.com"},
//	    AllowCredentials: true,
//	}))
func CORS(cfg *CORSConfig) gin.HandlerFunc {
	c := cors.DefaultConfig()

	if cfg == nil {
		if gin.Mode() == gin.ReleaseMode {
			panic("middleware.CORS(nil) enables allow-all origins and is not allowed in release mode; provide an explicit CORSConfig or use CORSFromConfig")
		}
		c.AllowAllOrigins = true
		c.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		c.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
		return cors.New(c)
	}

	if len(cfg.AllowOrigins) == 0 {
		defaults := settings.CORSConfig{}.WithDefaults()
		c.AllowOrigins = defaults.AllowOrigins
	} else if len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*" {
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
	if gin.Mode() == gin.ReleaseMode && c.AllowAllOrigins {
		log.Println("[gin-ninja] WARNING: middleware.CORS allows all origins in release mode")
	}

	return cors.New(c)
}

// CORSFromConfig returns a CORS middleware from settings.CORSConfig.
// Missing policy fields are filled with safe, explicit localhost defaults
// instead of falling back to middleware.CORS(nil)'s allow-all development mode.
func CORSFromConfig(cfg settings.CORSConfig) gin.HandlerFunc {
	cfg = cfg.WithDefaults()
	return CORS(&CORSConfig{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAgeSecs:       cfg.MaxAgeSecs,
	})
}
