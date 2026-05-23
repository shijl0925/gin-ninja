package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/shijl0925/gin-ninja/pkg/response"
)

const authPrincipalKey = "gin_ninja_auth_principal"
const authScopesKey = "gin_ninja_auth_scopes"

// APIKeyAuthenticator validates an API key and returns the authenticated principal.
type APIKeyAuthenticator func(c *gin.Context, key string) (any, bool)

// BasicAuthenticator validates HTTP Basic credentials and returns the authenticated principal.
type BasicAuthenticator func(c *gin.Context, username, password string) (any, bool)

// BearerTokenAuthenticator validates a bearer token and returns the authenticated principal.
type BearerTokenAuthenticator func(c *gin.Context, token string) (any, bool)

// OAuth2TokenAuthenticator validates an OAuth2 bearer token and returns the
// authenticated principal and granted scopes.
type OAuth2TokenAuthenticator func(c *gin.Context, token string) (principal any, scopes []string, valid bool)

// AuthPrincipalKey returns the context key used to store auth principals.
func AuthPrincipalKey() string { return authPrincipalKey }

// AuthScopesKey returns the context key used to store granted OAuth2 scopes.
func AuthScopesKey() string { return authScopesKey }

// GetAuthPrincipal retrieves the principal stored by API key, Basic, bearer,
// OAuth2, or JWT middleware.
func GetAuthPrincipal(c *gin.Context) any {
	v, exists := c.Get(authPrincipalKey)
	if exists {
		return v
	}
	v, exists = c.Get(claimsKey)
	if exists {
		return v
	}
	return nil
}

// GetAuthScopes retrieves the granted OAuth2 scopes stored by scope-aware
// OAuth2 bearer middleware.
func GetAuthScopes(c *gin.Context) []string {
	v, exists := c.Get(authScopesKey)
	if !exists {
		return nil
	}
	scopes, _ := v.([]string)
	return append([]string{}, scopes...)
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
			unauthorizedWithChallenge(c, basicChallenge(realm), "missing basic credentials")
			return
		}
		principal, valid := authenticate(c, username, password)
		if !valid {
			unauthorizedWithChallenge(c, basicChallenge(realm), "invalid basic credentials")
			return
		}
		setAuthPrincipal(c, principal)
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
		setAuthPrincipal(c, principal)
		c.Next()
	}
}

// OAuth2BearerAuth returns middleware for OAuth2 bearer token protected endpoints.
func OAuth2BearerAuth(authenticate BearerTokenAuthenticator) gin.HandlerFunc {
	return HTTPBearerAuth(authenticate)
}

// OAuth2BearerAuthWithScopes returns middleware for OAuth2 bearer token
// protected endpoints that require all listed scopes.
func OAuth2BearerAuthWithScopes(requiredScopes []string, authenticate OAuth2TokenAuthenticator) gin.HandlerFunc {
	if authenticate == nil {
		panic("oauth2 bearer auth: authenticator must not be nil")
	}
	requiredScopes = append([]string{}, requiredScopes...)
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			unauthorizedWithChallenge(c, bearerChallenge(requiredScopes, ""), "missing or malformed token")
			return
		}
		principal, grantedScopes, valid := authenticate(c, token)
		if !valid {
			unauthorizedWithChallenge(c, bearerChallenge(requiredScopes, "invalid_token"), "invalid token")
			return
		}
		if !hasRequiredScopes(grantedScopes, requiredScopes) {
			forbiddenWithChallenge(c, bearerChallenge(requiredScopes, "insufficient_scope"), "insufficient scope")
			return
		}
		setAuthPrincipal(c, principal)
		setAuthScopes(c, grantedScopes)
		c.Next()
	}
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
		setAuthPrincipal(c, principal)
		c.Next()
	}
}

func setAuthPrincipal(c *gin.Context, principal any) {
	c.Set(authPrincipalKey, principal)
}

func setAuthScopes(c *gin.Context, scopes []string) {
	c.Set(authScopesKey, append([]string{}, scopes...))
}

func unauthorizedWithChallenge(c *gin.Context, challenge, message string) {
	c.Header("WWW-Authenticate", challenge)
	response.Unauthorized(c, message)
}

func forbiddenWithChallenge(c *gin.Context, challenge, message string) {
	c.Header("WWW-Authenticate", challenge)
	response.Forbidden(c, message)
}

func basicChallenge(realm string) string {
	return `Basic realm="` + quoteAuthParam(realm) + `"`
}

func bearerChallenge(scopes []string, errCode string) string {
	params := make([]string, 0, 2)
	if errCode != "" {
		params = append(params, `error="`+quoteAuthParam(errCode)+`"`)
	}
	if len(scopes) > 0 {
		params = append(params, `scope="`+quoteAuthParam(strings.Join(scopes, " "))+`"`)
	}
	if len(params) == 0 {
		return "Bearer"
	}
	return "Bearer " + strings.Join(params, ", ")
}

func quoteAuthParam(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

func hasRequiredScopes(grantedScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}
	granted := make(map[string]struct{}, len(grantedScopes))
	for _, scope := range grantedScopes {
		granted[scope] = struct{}{}
	}
	for _, scope := range requiredScopes {
		if _, ok := granted[scope]; !ok {
			return false
		}
	}
	return true
}
