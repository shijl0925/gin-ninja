package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pkg/response"
	"github.com/shijl0925/gin-ninja/settings"
)

const claimsKey = contextkeys.JWTClaims

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

func generateToken(userID uint, username, secret string, ttl time.Duration, issuer string) (string, error) {
	if secret == "" {
		return "", errors.New("jwt: secret must not be empty")
	}
	if issuer == "" {
		issuer = "gin-ninja"
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID:   userID,
		Username: username,
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
	parts := strings.Fields(auth)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
