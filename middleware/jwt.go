package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/shijl0925/gin-ninja/pkg/response"
	"github.com/shijl0925/gin-ninja/settings"
)

const claimsKey = "gin_ninja_jwt_claims"

// Claims is the custom JWT claims struct used by gin-ninja.
// Embed this in your own claims type if you need extra fields.
//
//	type MyClaims struct {
//	    middleware.Claims
//	    Role string `json:"role"`
//	}
type Claims struct {
	jwt.RegisteredClaims
	// UserID is the authenticated user's ID.
	UserID uint `json:"user_id"`
	// Username is the authenticated user's name.
	Username string `json:"username"`
	// Roles contains role names granted to the user.
	Roles []string `json:"roles,omitempty"`
	// Permissions contains fine-grained permissions granted to the user.
	Permissions []string `json:"permissions,omitempty"`
	// Scopes contains OAuth-style scopes granted to the user.
	Scopes []string `json:"scopes,omitempty"`
}

// GetUserID satisfies the claimsWithUserID interface used by ninja.Context.GetUserID().
func (c *Claims) GetUserID() uint { return c.UserID }

// JWTAuth returns a gin middleware that validates Bearer tokens using the
// HMAC secret from the global settings.
//
// On success the parsed *Claims are stored in the Gin context under the key
// returned by ClaimsKey().  On failure the request is aborted with 401.
//
//	api.Engine().Use(middleware.JWTAuth())
func JWTAuth() gin.HandlerFunc {
	return JWTAuthWithSecret(settings.Global.JWT.Secret)
}

// JWTAuthWithSecret is like JWTAuth but uses an explicit secret rather than
// reading from global settings.  Useful in tests or when running multiple APIs
// with different secrets.
func JWTAuthWithSecret(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			response.Unauthorized(c, "missing or malformed token")
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			response.Unauthorized(c, "invalid or expired token")
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

// GetClaims retrieves the JWT claims stored by the JWTAuth middleware.
// Returns nil if the middleware was not registered or the token was invalid.
func GetClaims(c *gin.Context) *Claims {
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}

// ClaimsKey returns the context key used to store JWT claims.
func ClaimsKey() string { return claimsKey }

// GenerateToken creates a signed JWT token for the given user.
// The token is signed with the HMAC secret and TTL from global settings.
func GenerateToken(userID uint, username string) (string, error) {
	secret := settings.Global.JWT.Secret
	ttl := settings.Global.JWT.ExpireDuration()
	issuer := settings.Global.JWT.Issuer
	if issuer == "" {
		issuer = "gin-ninja"
	}
	return generateToken(userID, username, secret, ttl, issuer)
}

// GenerateTokenWithSecret creates a signed JWT token with an explicit secret
// and TTL.  Use this when you cannot rely on the global settings (e.g. tests).
func GenerateTokenWithSecret(userID uint, username, secret string, ttl time.Duration) (string, error) {
	return generateToken(userID, username, secret, ttl, "gin-ninja")
}

// GenerateTokenWithClaims signs an explicit claims payload using global settings.
func GenerateTokenWithClaims(claims Claims) (string, error) {
	secret := settings.Global.JWT.Secret
	ttl := settings.Global.JWT.ExpireDuration()
	issuer := settings.Global.JWT.Issuer
	if issuer == "" {
		issuer = "gin-ninja"
	}
	return generateTokenFromClaims(claims, secret, ttl, issuer)
}

// GenerateTokenWithSecretAndClaims signs an explicit claims payload with an explicit secret and TTL.
func GenerateTokenWithSecretAndClaims(claims Claims, secret string, ttl time.Duration) (string, error) {
	return generateTokenFromClaims(claims, secret, ttl, "gin-ninja")
}

func generateToken(userID uint, username, secret string, ttl time.Duration, issuer string) (string, error) {
	return generateTokenFromClaims(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
		},
		UserID:   userID,
		Username: username,
	}, secret, ttl, issuer)
}

func generateTokenFromClaims(claims Claims, secret string, ttl time.Duration, issuer string) (string, error) {
	if secret == "" {
		return "", errors.New("jwt: secret must not be empty")
	}
	if issuer == "" {
		issuer = "gin-ninja"
	}
	now := time.Now()
	if claims.Issuer == "" {
		claims.Issuer = issuer
	}
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// extractBearerToken reads the Bearer token from the Authorization header.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// RequireRoles ensures the authenticated user has every requested role.
func RequireRoles(roles ...string) gin.HandlerFunc {
	return requireClaimsMatch("roles", roles, func(claims *Claims) []string { return claims.Roles })
}

// RequirePermissions ensures the authenticated user has every requested permission.
func RequirePermissions(permissions ...string) gin.HandlerFunc {
	return requireClaimsMatch("permissions", permissions, func(claims *Claims) []string { return claims.Permissions })
}

// RequireScopes ensures the authenticated user has every requested scope.
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return requireClaimsMatch("scopes", scopes, func(claims *Claims) []string { return claims.Scopes })
}

func requireClaimsMatch(kind string, required []string, getValues func(*Claims) []string) gin.HandlerFunc {
	required = sanitizeRequiredValues(required)
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			response.Unauthorized(c, "authentication required")
			return
		}
		if !containsAll(getValues(claims), required) {
			response.Forbidden(c, "missing required "+kind)
			return
		}
		c.Next()
	}
}

func sanitizeRequiredValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsAll(have, required []string) bool {
	if len(required) == 0 {
		return true
	}
	lookup := make(map[string]struct{}, len(have))
	for _, value := range have {
		lookup[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := lookup[value]; !ok {
			return false
		}
	}
	return true
}
