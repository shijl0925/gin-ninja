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
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("[%d] %s: %s", e.Status, e.Code, e.Message)
	case e.Code != "":
		return fmt.Sprintf("[%d] %s", e.Status, e.Code)
	case e.Message != "":
		return fmt.Sprintf("[%d] %s", e.Status, e.Message)
	default:
		return fmt.Sprintf("[%d]", e.Status)
	}
}

// Is reports whether the target error represents the same API error kind.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	other, ok := target.(*Error)
	if !ok || other == nil {
		return false
	}
	switch {
	case e.Code != "" && other.Code != "":
		return e.Status == other.Status && e.Code == other.Code
	case e.Message != "" && other.Message != "":
		return e.Status == other.Status && e.Message == other.Message
	default:
		return e.Status != 0 && e.Status == other.Status
	}
}

// BusinessError represents a domain-level business logic failure.
// Unlike *Error (which carries an HTTP status code), BusinessError always
// produces an HTTP 200 response body using the standard business envelope:
//
//	{"code": <non-zero int>, "message": "...", "data": null}
//
// This mirrors the pkg/response.R convention used for business-level codes.
//
// Example:
//
//	return nil, ninja.NewBusinessError(10001, "user account is disabled")
type BusinessError struct {
	// Code is the application-level integer error code (must be != 0).
	Code int `json:"code"`
	// Message is a human-readable description of the failure.
	Message string `json:"message"`
	// Detail carries optional structured diagnostic data.
	Detail interface{} `json:"detail,omitempty"`
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[business:%d] %s", e.Code, e.Message)
}

// Is reports whether the target represents the same business error.
func (e *BusinessError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	other, ok := target.(*BusinessError)
	if !ok || other == nil {
		return false
	}
	return e.Code != 0 && e.Code == other.Code
}

// NewBusinessError creates a BusinessError with the given code and message.
func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

// NewBusinessErrorWithDetail creates a BusinessError with a detail payload.
func NewBusinessErrorWithDetail(code int, message string, detail interface{}) *BusinessError {
	return &BusinessError{Code: code, Message: message, Detail: detail}
}

// businessErrorResponse is the JSON envelope for business errors.
type businessErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

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

// Common builtin API error templates.
var (
	errBadRequest   = &Error{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "bad request"}
	errUnauthorized = &Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "unauthorized"}
	errForbidden    = &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "forbidden"}
	errNotFound     = &Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: "not found"}
	errConflict     = &Error{Status: http.StatusConflict, Code: "CONFLICT", Message: "conflict"}
	errInternal     = &Error{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "internal server error"}
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
	case *BusinessError:
		c.AbortWithStatusJSON(http.StatusOK, businessErrorResponse{
			Code:    e.Code,
			Message: e.Message,
			Data:    e.Detail,
		})
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
func BadRequestError() *Error { return cloneBuiltinError(errBadRequest) }

// UnauthorizedError returns a fresh copy of the standard unauthorized error.
func UnauthorizedError() *Error { return cloneBuiltinError(errUnauthorized) }

// ForbiddenError returns a fresh copy of the standard forbidden error.
func ForbiddenError() *Error { return cloneBuiltinError(errForbidden) }

// NotFoundError returns a fresh copy of the standard not-found error.
func NotFoundError() *Error { return cloneBuiltinError(errNotFound) }

// ConflictError returns a fresh copy of the standard conflict error.
func ConflictError() *Error { return cloneBuiltinError(errConflict) }

// InternalError returns a fresh copy of the standard internal-server error.
func InternalError() *Error { return cloneBuiltinError(errInternal) }

func IsBadRequest(err error) bool { return errors.Is(err, errBadRequest) }

func IsUnauthorized(err error) bool { return errors.Is(err, errUnauthorized) }

func IsForbidden(err error) bool { return errors.Is(err, errForbidden) }

func IsNotFound(err error) bool { return errors.Is(err, errNotFound) }

func IsConflict(err error) bool { return errors.Is(err, errConflict) }

func IsInternal(err error) bool { return errors.Is(err, errInternal) }
