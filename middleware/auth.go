package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/shijl0925/gin-ninja/pkg/response"
)

const authPrincipalKey = "gin_ninja_auth_principal"

// APIKeyAuthenticator validates an API key and returns the authenticated principal.
type APIKeyAuthenticator func(c *gin.Context, key string) (any, bool)

// BasicAuthenticator validates HTTP Basic credentials and returns the authenticated principal.
type BasicAuthenticator func(c *gin.Context, username, password string) (any, bool)

// BearerTokenAuthenticator validates a bearer token and returns the authenticated principal.
type BearerTokenAuthenticator func(c *gin.Context, token string) (any, bool)

// AuthPrincipalKey returns the context key used to store non-JWT auth principals.
func AuthPrincipalKey() string { return authPrincipalKey }

// GetAuthPrincipal retrieves the principal stored by API key, Basic, or OAuth2 bearer middleware.
func GetAuthPrincipal(c *gin.Context) any {
	v, exists := c.Get(authPrincipalKey)
	if !exists {
		return nil
	}
	return v
}

// APIKeyHeader returns middleware that authenticates an API key from a header.
func APIKeyHeader(name string, authenticate APIKeyAuthenticator) gin.HandlerFunc {
	return apiKeyAuth(func(c *gin.Context) string {
		return c.GetHeader(name)
	}, authenticate)
}

// APIKeyCookie returns middleware that authenticates an API key from a cookie.
func APIKeyCookie(name string, authenticate APIKeyAuthenticator) gin.HandlerFunc {
	return apiKeyAuth(func(c *gin.Context) string {
		key, err := c.Cookie(name)
		if err != nil {
			return ""
		}
		return key
	}, authenticate)
}

// APIKeyQuery returns middleware that authenticates an API key from a query parameter.
func APIKeyQuery(name string, authenticate APIKeyAuthenticator) gin.HandlerFunc {
	return apiKeyAuth(func(c *gin.Context) string {
		return c.Query(name)
	}, authenticate)
}

// HTTPBasicAuth returns middleware that authenticates HTTP Basic credentials.
func HTTPBasicAuth(realm string, authenticate BasicAuthenticator) gin.HandlerFunc {
	if authenticate == nil {
		panic("basic auth: authenticator must not be nil")
	}
	if realm == "" {
		realm = "Restricted"
	}
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			unauthorizedWithChallenge(c, `Basic realm="`+strings.ReplaceAll(realm, `"`, `\"`)+`"`, "missing basic credentials")
			return
		}
		principal, valid := authenticate(c, username, password)
		if !valid {
			unauthorizedWithChallenge(c, `Basic realm="`+strings.ReplaceAll(realm, `"`, `\"`)+`"`, "invalid basic credentials")
			return
		}
		c.Set(authPrincipalKey, principal)
		c.Next()
	}
}

// HTTPBearerAuth returns middleware that authenticates a generic bearer token.
func HTTPBearerAuth(authenticate BearerTokenAuthenticator) gin.HandlerFunc {
	if authenticate == nil {
		panic("bearer auth: authenticator must not be nil")
	}
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			unauthorizedWithChallenge(c, "Bearer", "missing or malformed token")
			return
		}
		principal, valid := authenticate(c, token)
		if !valid {
			unauthorizedWithChallenge(c, "Bearer", "invalid token")
			return
		}
		c.Set(authPrincipalKey, principal)
		c.Next()
	}
}

// OAuth2BearerAuth returns middleware for OAuth2 bearer token protected endpoints.
func OAuth2BearerAuth(authenticate BearerTokenAuthenticator) gin.HandlerFunc {
	return HTTPBearerAuth(authenticate)
}

func apiKeyAuth(extract func(*gin.Context) string, authenticate APIKeyAuthenticator) gin.HandlerFunc {
	if authenticate == nil {
		panic("api key auth: authenticator must not be nil")
	}
	return func(c *gin.Context) {
		key := extract(c)
		if key == "" {
			response.Unauthorized(c, "missing api key")
			return
		}
		principal, valid := authenticate(c, key)
		if !valid {
			response.Unauthorized(c, "invalid api key")
			return
		}
		c.Set(authPrincipalKey, principal)
		c.Next()
	}
}

func unauthorizedWithChallenge(c *gin.Context, challenge, message string) {
	c.Header("WWW-Authenticate", challenge)
	response.Unauthorized(c, message)
}
