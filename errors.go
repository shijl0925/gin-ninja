package ninja

import (
	"fmt"
	"net/http"
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
