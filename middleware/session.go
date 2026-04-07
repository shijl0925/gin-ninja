package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SessionConfig holds configuration for the cookie-based session middleware.
type SessionConfig struct {
	// CookieName is the name of the session cookie.  Defaults to "session".
	CookieName string
	// MaxAge is the cookie lifetime in seconds.  Defaults to 86400 (24 h).
	MaxAge int
	// Path is the cookie path.  Defaults to "/".
	Path string
	// Domain is the optional cookie domain.
	Domain string
	// Secure marks the cookie as Secure (HTTPS only).
	Secure bool
	// HTTPOnly marks the cookie as HttpOnly (no JavaScript access).  Defaults to true.
	// To explicitly disable HttpOnly, set HTTPOnlySet to true as well.
	HTTPOnly bool
	// HTTPOnlySet applies the HTTPOnly value even when it is false.
	// When false, HTTPOnly defaults to true.
	HTTPOnlySet bool
	// SameSite controls the SameSite attribute.  Defaults to http.SameSiteLaxMode.
	SameSite http.SameSite
	// Secret is the HMAC-SHA256 key used to sign the cookie value.
	// Must be set; panics if empty when the middleware is called.
	Secret string
}

func (cfg *SessionConfig) withDefaults() *SessionConfig {
	out := *cfg
	if out.CookieName == "" {
		out.CookieName = "session"
	}
	if out.MaxAge == 0 {
		out.MaxAge = 86400
	}
	if out.Path == "" {
		out.Path = "/"
	}
	if out.SameSite == 0 {
		out.SameSite = http.SameSiteLaxMode
	}
	if !out.HTTPOnlySet {
		out.HTTPOnly = true
	}
	return &out
}

// sessionContextKey is the gin context key under which the *Session is stored.
const sessionContextKey = "gin_ninja_session"

// Session is a mutable in-request key/value store backed by a signed cookie.
// Changes are only persisted to the client when Save is called (or when the
// middleware saves it automatically at the end of the handler chain).
type Session struct {
	cfg     *SessionConfig
	data    map[string]string
	dirty   bool
}

// Get returns the value for key and whether it was found.
func (s *Session) Get(key string) (string, bool) {
	v, ok := s.data[key]
	return v, ok
}

// Set stores key/value in the session and marks it as modified.
func (s *Session) Set(key, value string) {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	s.data[key] = value
	s.dirty = true
}

// Delete removes key from the session and marks it as modified.
func (s *Session) Delete(key string) {
	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		s.dirty = true
	}
}

// Clear removes all keys from the session and marks it as modified.
func (s *Session) Clear() {
	s.data = make(map[string]string)
	s.dirty = true
}

// Keys returns all keys currently stored in the session.
func (s *Session) Keys() []string {
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Save writes the session to the response cookie.  Call this explicitly if
// you need the cookie written before the end of the handler chain; the
// middleware calls it automatically after c.Next().
func (s *Session) Save(c *gin.Context) error {
	raw, err := encodeSession(s.data, s.cfg.Secret)
	if err != nil {
		return err
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    raw,
		Path:     s.cfg.Path,
		Domain:   s.cfg.Domain,
		MaxAge:   s.cfg.MaxAge,
		Secure:   s.cfg.Secure,
		HttpOnly: s.cfg.HTTPOnly,
		SameSite: s.cfg.SameSite,
	})
	s.dirty = false
	return nil
}

// SessionMiddleware returns a gin middleware that loads the session from the
// request cookie, makes it available via GetSession(c), and automatically
// saves any modifications back to the response cookie after the handler runs.
//
//	api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
//	    Secret:   "change-me-in-production",
//	    Secure:   true,
//	}))
//
// Retrieve the session inside a handler:
//
//	session := middleware.GetSession(c)
//	session.Set("user_id", "42")
func SessionMiddleware(cfg *SessionConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = &SessionConfig{}
	}
	cfg = cfg.withDefaults()
	if cfg.Secret == "" {
		panic("session: Secret must not be empty")
	}

	return func(c *gin.Context) {
		sess := &Session{cfg: cfg, data: make(map[string]string)}

		if raw, err := c.Cookie(cfg.CookieName); err == nil && raw != "" {
			if data, err := decodeSession(raw, cfg.Secret); err == nil {
				sess.data = data
			}
		}

		c.Set(sessionContextKey, sess)
		c.Next()

		if sess.dirty {
			_ = sess.Save(c)
		}
	}
}

// GetSession retrieves the *Session stored by the SessionMiddleware.
// Returns nil if the middleware has not been registered.
func GetSession(c *gin.Context) *Session {
	v, exists := c.Get(sessionContextKey)
	if !exists {
		return nil
	}
	s, _ := v.(*Session)
	return s
}

// ---------------------------------------------------------------------------
// Internal helpers – signing and encoding
// ---------------------------------------------------------------------------

// encodeSession serialises data as JSON, then signs and base64-encodes it.
// The resulting cookie value is:  base64(json) + "." + base64(hmac)
func encodeSession(data map[string]string, secret string) (string, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b64Payload := base64.RawURLEncoding.EncodeToString(payload)
	sig := sessionHMAC(b64Payload, secret)
	return b64Payload + "." + sig, nil
}

// decodeSession verifies and decodes a cookie value produced by encodeSession.
// Returns an error if the signature is invalid or the JSON is malformed.
func decodeSession(value, secret string) (map[string]string, error) {
	idx := strings.LastIndex(value, ".")
	if idx < 0 {
		return nil, errors.New("session: malformed cookie: missing signature separator")
	}
	b64Payload := value[:idx]
	sig := value[idx+1:]

	expectedSig := sessionHMAC(b64Payload, secret)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, errors.New("session: invalid signature")
	}

	raw, err := base64.RawURLEncoding.DecodeString(b64Payload)
	if err != nil {
		return nil, err
	}

	var data map[string]string
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func sessionHMAC(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload)) //nolint:errcheck
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// generateSessionID generates a random 32-byte URL-safe token.
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is a system-level error (e.g. /dev/urandom unavailable).
		// Using a predictable fallback would be insecure, so panic instead.
		panic("session: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// NewSessionID generates a fresh random session ID string.
// This is a convenience helper for applications that use the session to store
// a session ID that references server-side state.
func NewSessionID() string { return generateSessionID() }
