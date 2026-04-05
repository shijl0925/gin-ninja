package ninja

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Error represents an API error response.
type Error struct {
	// Status is the HTTP status code.
	Status int `json:"-"`
	// Code is an optional machine-readable error code.
	Code string `json:"code,omitempty"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
	// Detail provides additional error context.
	Detail interface{} `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Status, e.Code, e.Message)
}

// ValidationError represents one or more validation failures on the request.
type ValidationError struct {
	// Errors contains the individual field validation errors.
	Errors []FieldError `json:"errors"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed with %d error(s)", len(e.Errors))
}

// FieldError represents a single field-level validation failure.
type FieldError struct {
	// Field is the name of the field that failed validation.
	Field string `json:"field"`
	// Message describes why validation failed.
	Message string `json:"message"`
}

// Common pre-built API errors.
var (
	ErrBadRequest   = &Error{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "bad request"}
	ErrUnauthorized = &Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "unauthorized"}
	ErrForbidden    = &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "forbidden"}
	ErrNotFound     = &Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: "not found"}
	ErrConflict     = &Error{Status: http.StatusConflict, Code: "CONFLICT", Message: "conflict"}
	ErrInternal     = &Error{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "internal server error"}
)

// NewError creates a new API error with the given status code and message.
func NewError(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// NewErrorWithCode creates a new API error with a status code, machine-readable code, and message.
func NewErrorWithCode(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// errorResponse is the JSON envelope returned for errors.
type errorResponse struct {
	Error interface{} `json:"error"`
}

// ErrorMapper converts arbitrary errors into framework errors.
// Returning nil means the mapper did not handle the error.
type ErrorMapper func(error) error

var (
	errorMappersMu sync.RWMutex
	errorMappers   = defaultErrorMappers()
)

func defaultErrorMappers() []ErrorMapper {
	return []ErrorMapper{
		func(err error) error {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return &Error{
					Status:  http.StatusRequestTimeout,
					Code:    "REQUEST_TIMEOUT",
					Message: "request timed out",
				}
			default:
				return nil
			}
		},
	}
}

func cloneBuiltinError(err *Error) *Error {
	if err == nil {
		return nil
	}
	cloned := *err
	return &cloned
}

func errorMappersSnapshot() []ErrorMapper {
	errorMappersMu.RLock()
	defer errorMappersMu.RUnlock()
	return append([]ErrorMapper(nil), errorMappers...)
}

// RegisterErrorMapper appends a custom process-wide error mapper.
//
// Deprecated: prefer api.RegisterErrorMapper for per-instance behavior.
func RegisterErrorMapper(mapper ErrorMapper) {
	if mapper == nil {
		return
	}
	errorMappersMu.Lock()
	defer errorMappersMu.Unlock()
	errorMappers = append(errorMappers, mapper)
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	return mapErrorWithMappers(err, errorMappersSnapshot())
}

func mapErrorWithMappers(err error, mappers []ErrorMapper) error {
	for _, mapper := range mappers {
		if mapper == nil {
			continue
		}
		if mapped := mapper(err); mapped != nil {
			err = mapped
			break
		}
	}
	return err
}

// WriteError writes an appropriate JSON error response.
func WriteError(c *gin.Context, err error) {
	if api, ok := currentAPI(c); ok {
		err = api.mapError(err)
	} else {
		err = mapError(err)
	}

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

// BadRequestError returns a fresh copy of the standard bad-request error.
func BadRequestError() *Error { return cloneBuiltinError(ErrBadRequest) }

// UnauthorizedError returns a fresh copy of the standard unauthorized error.
func UnauthorizedError() *Error { return cloneBuiltinError(ErrUnauthorized) }

// ForbiddenError returns a fresh copy of the standard forbidden error.
func ForbiddenError() *Error { return cloneBuiltinError(ErrForbidden) }

// NotFoundError returns a fresh copy of the standard not-found error.
func NotFoundError() *Error { return cloneBuiltinError(ErrNotFound) }

// ConflictError returns a fresh copy of the standard conflict error.
func ConflictError() *Error { return cloneBuiltinError(ErrConflict) }

// InternalError returns a fresh copy of the standard internal-server error.
func InternalError() *Error { return cloneBuiltinError(ErrInternal) }
