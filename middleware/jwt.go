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

// JWTAuthWithSecret returns a gin middleware that validates Bearer tokens using
// an explicit HMAC secret.
func JWTAuthWithSecret(secret string) gin.HandlerFunc {
	if secret == "" {
		panic("jwt: Secret must not be empty")
	}
	return func(c *gin.Context) {
		validateJWTToken(c, secret)
	}
}

// JWTAuthWithConfig returns a gin middleware configured from an explicit
// settings.JWTConfig value.
func JWTAuthWithConfig(cfg settings.JWTConfig) gin.HandlerFunc {
	return JWTAuthWithSecret(cfg.Secret)
}

// validateJWTToken extracts, parses, and validates the Bearer token in c.
// On success it stores the parsed claims and calls c.Next(); on failure it
// aborts with 401.
func validateJWTToken(c *gin.Context, secret string) {
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

// GenerateTokenWithSecret creates a signed JWT token with an explicit secret
// and TTL.
func GenerateTokenWithSecret(userID uint, username, secret string, ttl time.Duration) (string, error) {
	return generateToken(userID, username, secret, ttl, "gin-ninja")
}

// GenerateTokenWithConfig creates a signed JWT token from an explicit JWT
// configuration rather than reading from global settings.
func GenerateTokenWithConfig(userID uint, username string, cfg settings.JWTConfig) (string, error) {
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = "gin-ninja"
	}
	return generateToken(userID, username, cfg.Secret, cfg.ExpireDuration(), issuer)
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
