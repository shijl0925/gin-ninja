package ninja

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/internal/contextkeys"
	"github.com/shijl0925/gin-ninja/pkg/i18n"
)

const (
	requestIDContextKey = "X-Request-ID"
)

// Context wraps *gin.Context and is passed to every handler function.
// It gives handlers access to the underlying gin context while remaining
// compatible with the standard library context.Context interface.
type Context struct {
	*gin.Context
}

var _ context.Context = (*Context)(nil)

// newContext wraps a gin.Context.
func newContext(c *gin.Context) *Context {
	return &Context{Context: c}
}

// Deadline implements context.Context.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Request.Context().Deadline()
}

// Done implements context.Context.
func (c *Context) Done() <-chan struct{} {
	return c.Request.Context().Done()
}

// Err implements context.Context.
func (c *Context) Err() error {
	return c.Request.Context().Err()
}

// Value implements context.Context.  Keys set via gin Set/Get are checked
// first; if not found, the request context is consulted.
func (c *Context) Value(key any) any {
	if k, ok := key.(string); ok {
		if v, exists := c.Get(k); exists {
			return v
		}
	}
	return c.Request.Context().Value(key)
}

// StdContext returns the standard library context from the request.
func (c *Context) StdContext() context.Context {
	return c.Request.Context()
}

// RequestID returns the X-Request-ID value injected by the RequestID middleware.
// Returns an empty string if the middleware was not registered.
func (c *Context) RequestID() string {
	id, _ := c.Get(requestIDContextKey)
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// GetUserID returns the authenticated user's ID from the JWT claims.
// Returns 0 if the JWTAuth middleware was not registered or the token was invalid.
func (c *Context) GetUserID() uint {
	v, exists := c.Get(contextkeys.JWTClaims)
	if !exists {
		return 0
	}
	// Claims is stored by the middleware package as *middleware.Claims which has
	// a UserID field.  Use a minimal interface to avoid a circular import.
	type claimsWithUserID interface {
		GetUserID() uint
	}
	if cl, ok := v.(claimsWithUserID); ok {
		return cl.GetUserID()
	}
	return 0
}

// Locale returns the negotiated locale stored by the I18n middleware.
// Returns "en" if the I18n middleware is not registered or the context does
// not contain a locale value.
//
//	locale := ctx.Locale()  // "en" or "zh"
func (c *Context) Locale() string {
	v, exists := c.Get(contextkeys.Locale)
	if !exists {
		return i18n.Default
	}
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return i18n.Default
}

// T returns the translated message for key in the request locale.  Variadic
// args are applied with fmt.Sprintf when the message contains format verbs.
//
//	msg := ctx.T("not_found")           // "not found" or "资源不存在"
func (c *Context) T(key string, args ...interface{}) string {
	return i18n.T(c.Locale(), key, args...)
}

// JSON200 is a convenience method to respond with 200 OK and a JSON body.
func (c *Context) JSON200(obj interface{}) {
	c.JSON(http.StatusOK, obj)
}

// JSON201 is a convenience method to respond with 201 Created and a JSON body.
func (c *Context) JSON201(obj interface{}) {
	c.JSON(http.StatusCreated, obj)
}

// JSON204 is a convenience method to respond with 204 No Content.
func (c *Context) JSON204() {
	c.Status(http.StatusNoContent)
}

// Forbidden aborts the request with 403 Forbidden.
func (c *Context) Forbidden(message string) {
	WriteError(c.Context, &Error{
		Status:  http.StatusForbidden,
		Code:    "FORBIDDEN",
		Message: message,
	})
}

// Unauthorized aborts the request with 401 Unauthorized.
func (c *Context) Unauthorized(message string) {
	WriteError(c.Context, &Error{
		Status:  http.StatusUnauthorized,
		Code:    "UNAUTHORIZED",
		Message: message,
	})
}

// BeginTx starts a request-scoped transaction if one is not already active.
//
// Deprecated: import github.com/shijl0925/gin-ninja/orm and call orm.BeginTx instead.
func (c *Context) BeginTx() error {
	begin, _, _, _ := transactionHandlers()
	if begin == nil {
		return errTransactionUnavailable()
	}
	return begin(c.Context)
}

// CommitTx commits the active request-scoped transaction.
//
// Deprecated: import github.com/shijl0925/gin-ninja/orm and call orm.CommitTx instead.
func (c *Context) CommitTx() error {
	_, commit, _, _ := transactionHandlers()
	if commit == nil {
		return errTransactionUnavailable()
	}
	return commit(c.Context)
}

// RollbackTx rolls back the active request-scoped transaction.
//
// Deprecated: import github.com/shijl0925/gin-ninja/orm and call orm.RollbackTx instead.
func (c *Context) RollbackTx() error {
	_, _, rollback, _ := transactionHandlers()
	if rollback == nil {
		return errTransactionUnavailable()
	}
	return rollback(c.Context)
}
