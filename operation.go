package ninja

import (
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
)

// OperationOption is a functional option for configuring an Operation.
type OperationOption func(*operation)

// Summary sets the human-readable summary shown in the OpenAPI docs.
func Summary(s string) OperationOption {
	return func(op *operation) { op.summary = s }
}

// Description sets the long description shown in the OpenAPI docs.
func Description(d string) OperationOption {
	return func(op *operation) { op.description = d }
}

// OperationID sets an explicit operationId in the OpenAPI spec.
func OperationID(id string) OperationOption {
	return func(op *operation) { op.operationID = id }
}

// Tags overrides the tags for this specific operation.
func Tags(tags ...string) OperationOption {
	return func(op *operation) { op.tags = tags }
}

// TagDescription records a top-level OpenAPI tag description for this operation.
func TagDescription(tag, description string) OperationOption {
	return func(op *operation) {
		if op.tagDescriptions == nil {
			op.tagDescriptions = map[string]string{}
		}
		op.tagDescriptions[tag] = description
	}
}

// Security adds an OpenAPI security requirement to this operation.
func Security(name string, scopes ...string) OperationOption {
	return func(op *operation) {
		op.security = append(op.security, SecurityRequirement{name: append([]string{}, scopes...)})
	}
}

// SecurityMiddleware adds an OpenAPI security requirement and attaches the
// matching Gin middleware to this operation.
func SecurityMiddleware(name string, mw gin.HandlerFunc, scopes ...string) OperationOption {
	return func(op *operation) {
		Security(name, scopes...)(op)
		appendOperationAuthMiddleware(op, mw)
	}
}

// BearerAuth marks this operation as requiring the default bearerAuth scheme.
// If middleware is provided, it is also attached to the operation.
func BearerAuth(mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		Security("bearerAuth")(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// BasicAuth marks this operation as requiring the default basicAuth scheme.
// If middleware is provided, it is also attached to the operation.
func BasicAuth(mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		Security("basicAuth")(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// APIKeyAuth marks this operation as requiring the named API key scheme.
// If middleware is provided, it is also attached to the operation.
func APIKeyAuth(name string, mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		Security(name)(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// OAuth2Auth marks this operation as requiring the default oauth2 scheme.
func OAuth2Auth(scopes ...string) OperationOption {
	return Security("oauth2", scopes...)
}

// OAuth2AuthMiddleware applies the default oauth2 OpenAPI security requirement
// and attaches the matching Gin middleware to this operation.
func OAuth2AuthMiddleware(mw gin.HandlerFunc, scopes ...string) OperationOption {
	return SecurityMiddleware("oauth2", mw, scopes...)
}

func appendOperationAuthMiddleware(op *operation, mw ...gin.HandlerFunc) {
	for _, handler := range mw {
		if handler == nil {
			panic("gin-ninja: auth middleware must not be nil")
		}
		op.ginMiddleware = append(op.ginMiddleware, handler)
	}
}

// Deprecated marks the operation as deprecated in the docs.
func Deprecated() OperationOption {
	return func(op *operation) { op.deprecated = true }
}

// Cache enables route-level response caching for safe read endpoints.
func Cache(ttl time.Duration, opts ...CacheOption) OperationOption {
	return func(op *operation) {
		op.cache = newRouteCacheConfig(ttl)
		for _, opt := range opts {
			opt(op.cache)
		}
		if op.cacheControl == "" && ttl > 0 {
			op.cacheControl = defaultCacheControl(ttl)
		}
		op.etagEnabled = true
	}
}

// CacheControl sets the Cache-Control response header for successful responses.
func CacheControl(value string) OperationOption {
	return func(op *operation) { op.cacheControl = value }
}

// ETag enables automatic ETag generation for successful responses.
func ETag() OperationOption {
	return func(op *operation) { op.etagEnabled = true }
}

// ExcludeFromDocs omits the operation from the generated OpenAPI spec.
func ExcludeFromDocs() OperationOption {
	return func(op *operation) { op.excludeFromDocs = true }
}

// SuccessStatus sets the HTTP status code used for successful responses.
// The default is 200 OK (201 Created is common for POST).
func SuccessStatus(code int) OperationOption {
	return func(op *operation) { op.successStatus = code }
}

// Response documents an additional OpenAPI response for the operation.
// Pass model as nil for responses without a JSON response body.
func Response(status int, description string, model any) OperationOption {
	return func(op *operation) {
		var modelType reflect.Type
		if model != nil {
			modelType = reflect.TypeOf(model)
		}
		op.responses = append(op.responses, documentedResponse{
			status:       status,
			description:  description,
			responseType: modelType,
		})
	}
}

// Paginated declares a standard paginated success response schema.
func Paginated[T any]() OperationOption {
	return func(op *operation) {
		var item T
		op.paginatedItemType = reflect.TypeOf(item)
	}
}

// PaginatedResponse documents an additional paginated OpenAPI response.
func PaginatedResponse[T any](status int, description string) OperationOption {
	return func(op *operation) {
		var item T
		op.responses = append(op.responses, documentedResponse{
			status:            status,
			description:       description,
			paginatedItemType: reflect.TypeOf(item),
		})
	}
}

// Timeout applies a context-based per-operation cooperative timeout.
// It cancels the request context and allows an early timeout response, but
// long-running handlers must observe ctx.Done() or ctx.Request.Context().Done()
// to stop promptly.
func Timeout(d time.Duration) OperationOption {
	return func(op *operation) { op.timeout = d }
}

// RateLimit applies a per-operation in-memory token-bucket rate limit.
func RateLimit(requestsPerSecond int, burst ...int) OperationOption {
	return func(op *operation) {
		if requestsPerSecond <= 0 {
			op.rateLimit = nil
			return
		}
		b := requestsPerSecond
		if len(burst) > 0 && burst[0] > 0 {
			b = burst[0]
		}
		op.rateLimit = newRateLimiter(float64(requestsPerSecond), float64(b))
	}
}

type documentedResponse struct {
	status            int
	description       string
	responseType      reflect.Type
	paginatedItemType reflect.Type
}

// operation holds all metadata about a single API endpoint and the
// gin-compatible handler function that wraps the user-supplied typed handler.
type operation struct {
	method            string
	path              string
	ginHandler        gin.HandlerFunc
	inputType         reflect.Type
	outputType        reflect.Type
	summary           string
	description       string
	operationID       string
	ginMiddleware     []gin.HandlerFunc
	tags              []string
	tagDescriptions   map[string]string
	security          []SecurityRequirement
	deprecated        bool
	successStatus     int
	responses         []documentedResponse
	paginatedItemType reflect.Type
	timeout           time.Duration
	rateLimit         *rateLimiter
	excludeFromDocs   bool
	withTransaction   bool
	cache             *routeCacheConfig
	cacheControl      string
	etagEnabled       bool
	version           string
	versionInfo       *VersionConfig
	stream            *streamConfig
}

// WithTransaction wraps the operation in a request-scoped database transaction.
func WithTransaction() OperationOption {
	return func(op *operation) { op.withTransaction = true }
}

func cloneSecurityRequirements(reqs []SecurityRequirement) []SecurityRequirement {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]SecurityRequirement, 0, len(reqs))
	for _, req := range reqs {
		cloned := make(SecurityRequirement, len(req))
		for name, scopes := range req {
			cloned[name] = append([]string{}, scopes...)
		}
		out = append(out, cloned)
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneVersionInfo(info *VersionConfig) *VersionConfig {
	if info == nil {
		return nil
	}
	cloned := *info
	return &cloned
}

// newOperation builds an operation and wraps the typed handler with
// parameter binding, error handling, and response serialization.
func newOperation[TIn any, TOut any](
	method, path string,
	handler func(ctx *Context, input *TIn) (*TOut, error),
	defaultTags []string,
) *operation {
	var zeroIn TIn
	var zeroOut TOut
	inputType := reflect.TypeOf(zeroIn)
	outputType := reflect.TypeOf(zeroOut)

	op := &operation{
		method:        method,
		path:          path,
		inputType:     inputType,
		outputType:    outputType,
		tags:          append([]string(nil), defaultTags...),
		successStatus: http.StatusOK,
	}

	op.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)

		// Allocate and populate the typed input.
		input := new(TIn)
		if err := bindInput(c, method, input); err != nil {
			writeError(c, err)
			return
		}

		// Invoke the user handler.
		var (
			output *TOut
			err    error
		)
		invoke := func() error {
			output, err = handler(ctx, input)
			return err
		}
		if op.withTransaction {
			api, _ := currentAPI(c)
			_, _, _, withTransaction := apiTransactionHandlers(api)
			if withTransaction == nil {
				err = errTransactionUnavailable()
			} else {
				err = withTransaction(c, func() error {
					return invokeWithContextGuard(c, invoke)
				})
			}
		} else {
			err = invoke()
		}
		if err != nil {
			writeError(c, err)
			return
		}

		// Write the response.
		if output == nil {
			c.Status(http.StatusNoContent)
			return
		}
		if writer, ok := any(output).(responseWriter); ok {
			writer.writeTo(c, op.successStatus)
			return
		}
		if c.Request.Method == http.MethodHead {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Status(op.successStatus)
			return
		}
		c.JSON(op.successStatus, output)
	}

	return op
}

// newVoidOperation builds an operation whose handler returns no typed body.
// Useful for DELETE endpoints that return 204 No Content.
func newVoidOperation[TIn any](
	method, path string,
	handler func(ctx *Context, input *TIn) error,
	defaultTags []string,
) *operation {
	var zeroIn TIn
	inputType := reflect.TypeOf(zeroIn)

	op := &operation{
		method:        method,
		path:          path,
		inputType:     inputType,
		outputType:    nil,
		tags:          append([]string(nil), defaultTags...),
		successStatus: http.StatusNoContent,
	}

	op.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)

		input := new(TIn)
		if err := bindInput(c, method, input); err != nil {
			writeError(c, err)
			return
		}

		invoke := func() error {
			return handler(ctx, input)
		}
		var err error
		if op.withTransaction {
			api, _ := currentAPI(c)
			_, _, _, withTransaction := apiTransactionHandlers(api)
			if withTransaction == nil {
				err = errTransactionUnavailable()
			} else {
				err = withTransaction(c, func() error {
					return invokeWithContextGuard(c, invoke)
				})
			}
		} else {
			err = invoke()
		}
		if err != nil {
			writeError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}

	return op
}

func (op *operation) finalize() {
	if op.ginHandler == nil {
		return
	}

	handler := op.ginHandler
	if op.timeout > 0 {
		if op.usesDirectResponseWriter() {
			handler = wrapCooperativeTimeout(op.timeout, handler)
		} else {
			handler = wrapTimeout(op.timeout, handler)
		}
	}
	if op.cache != nil || op.cacheControl != "" || op.etagEnabled {
		handler = wrapCache(op, handler)
	}
	if op.rateLimit != nil {
		handler = wrapRateLimit(op.rateLimit, handler)
	}
	op.ginHandler = handler
}

func (op *operation) usesDirectResponseWriter() bool {
	if op.stream != nil {
		return true
	}
	if op.outputType == nil {
		return false
	}
	responseWriterType := reflect.TypeOf((*responseWriter)(nil)).Elem()
	if reflect.PointerTo(op.outputType).Implements(responseWriterType) {
		return true
	}
	return op.outputType.Implements(responseWriterType)
}

func writeError(c *gin.Context, err error) {
	WriteError(c, err)
}

func invokeWithContextGuard(c *gin.Context, invoke func() error) error {
	if err := invoke(); err != nil {
		return err
	}
	if err := c.Request.Context().Err(); err != nil {
		return err
	}
	return nil
}
