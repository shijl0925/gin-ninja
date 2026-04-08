package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pkg/i18n"
)

// I18n returns a Gin middleware that negotiates the client locale from the
// Accept-Language request header and stores it in the Gin context under the
// key returned by LocaleKey().
//
// Locale selection is limited to the supported locales ("en", "zh") and
// falls back to "en" for unsupported or missing headers.
//
// Register this middleware before any handler that needs locale-aware
// behaviour (e.g. translated validation errors):
//
//	api.UseGin(middleware.I18n())
//
// Retrieve the active locale in a handler or downstream middleware:
//
//	locale := middleware.GetLocale(c)        // from *gin.Context
//	locale := ctx.Locale()                   // from *ninja.Context
func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.NegotiateLocale(c.GetHeader("Accept-Language"))
		c.Set(contextkeys.Locale, locale)
		c.Next()
	}
}

// GetLocale returns the negotiated locale stored by the I18n middleware.
// Returns the default locale ("en") if the middleware has not been registered
// or the context does not contain a locale value.
func GetLocale(c *gin.Context) string {
	v, exists := c.Get(contextkeys.Locale)
	if !exists {
		return i18n.Default
	}
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return i18n.Default
}

// LocaleKey returns the gin context key used to store the negotiated locale.
func LocaleKey() string { return contextkeys.Locale }
