package ninja

import (
	"context"
	"errors"
	"fmt"
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

// BearerAuth marks this operation as requiring the default bearerAuth scheme.
func BearerAuth() OperationOption {
	return Security("bearerAuth")
}

// Deprecated marks the operation as deprecated in the docs.
func Deprecated() OperationOption {
	return func(op *operation) { op.deprecated = true }
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

// Timeout applies a context-based per-operation timeout.
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
		output, err := handler(ctx, input)
		if err != nil {
			writeError(c, err)
			return
		}

		// Write the response.
		if output == nil {
			c.Status(http.StatusNoContent)
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

		if err := handler(ctx, input); err != nil {
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
		handler = wrapTimeout(op.timeout, handler)
	}
	if op.rateLimit != nil {
		handler = wrapRateLimit(op.rateLimit, handler)
	}
	op.ginHandler = handler
}

// writeError writes an appropriate JSON error response.
func writeError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *Error:
		status := e.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		c.AbortWithStatusJSON(status, errorResponse{Error: e})
	case *ValidationError:
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "request validation failed",
				"errors":  e.Errors,
			},
		})
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithStatusJSON(http.StatusRequestTimeout, errorResponse{
				Error: &Error{
					Status:  http.StatusRequestTimeout,
					Code:    "REQUEST_TIMEOUT",
					Message: "request timed out",
				},
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{
			Error: &Error{
				Status:  http.StatusInternalServerError,
				Code:    "INTERNAL_ERROR",
				Message: fmt.Sprintf("%v", err),
			},
		})
	}
}
