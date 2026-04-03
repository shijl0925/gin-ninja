package ninja

import (
	"fmt"
	"net/http"
	"reflect"

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

// Security adds an OpenAPI security requirement to this operation.
func Security(name string, scopes ...string) OperationOption {
	return func(op *operation) {
		op.security = append(op.security, SecurityRequirement{name: append([]string(nil), scopes...)})
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

// SuccessStatus sets the HTTP status code used for successful responses.
// The default is 200 OK (201 Created is common for POST).
func SuccessStatus(code int) OperationOption {
	return func(op *operation) { op.successStatus = code }
}

// operation holds all metadata about a single API endpoint and the
// gin-compatible handler function that wraps the user-supplied typed handler.
type operation struct {
	method        string
	path          string
	ginHandler    gin.HandlerFunc
	inputType     reflect.Type
	outputType    reflect.Type
	summary       string
	description   string
	operationID   string
	tags          []string
	security      []SecurityRequirement
	deprecated    bool
	successStatus int
}

func cloneSecurityRequirements(reqs []SecurityRequirement) []SecurityRequirement {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]SecurityRequirement, 0, len(reqs))
	for _, req := range reqs {
		cloned := make(SecurityRequirement, len(req))
		for name, scopes := range req {
			cloned[name] = append([]string(nil), scopes...)
		}
		out = append(out, cloned)
	}
	return out
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{
			Error: &Error{
				Status:  http.StatusInternalServerError,
				Code:    "INTERNAL_ERROR",
				Message: fmt.Sprintf("%v", err),
			},
		})
	}
}
