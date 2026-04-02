// Package response provides a standardised JSON response envelope for
// gin-ninja APIs, following the common pattern used in Go admin backends:
//
//	{"code": 0, "message": "success", "data": {...}}
//
// Usage:
//
//	func listUsers(ctx *ninja.Context, in *ListInput) (*response.R, error) {
//	    users, _ := svc.List()
//	    return response.OK(users), nil
//	}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard business-level codes.
const (
	// CodeOK indicates a successful operation.
	CodeOK = 0
	// CodeError is the generic business error code.
	CodeError = -1
	// CodeUnauthorized indicates missing or invalid authentication.
	CodeUnauthorized = 401
	// CodeForbidden indicates the caller lacks sufficient permissions.
	CodeForbidden = 403
	// CodeNotFound indicates the requested resource was not found.
	CodeNotFound = 404
	// CodeValidation indicates a request validation failure.
	CodeValidation = 422
)

// R is the standard response envelope.
//
//	{"code": 0, "message": "success", "data": null}
type R struct {
	// Code is the business-level result code (0 = success).
	Code int `json:"code"`
	// Message is a human-readable result description.
	Message string `json:"message"`
	// Data contains the response payload (can be any JSON-serialisable value).
	Data interface{} `json:"data"`
}

// OK returns a successful response containing the given data.
func OK(data interface{}) *R {
	return &R{Code: CodeOK, Message: "success", Data: data}
}

// OKWithMessage returns a successful response with a custom message.
func OKWithMessage(msg string, data interface{}) *R {
	return &R{Code: CodeOK, Message: msg, Data: data}
}

// Fail returns an error response with the given code and message.
func Fail(code int, message string) *R {
	return &R{Code: code, Message: message, Data: nil}
}

// FailWithData returns an error response that also carries a data payload.
func FailWithData(code int, message string, data interface{}) *R {
	return &R{Code: code, Message: message, Data: data}
}

// Error returns a generic error response (code = -1).
func Error(message string) *R {
	return Fail(CodeError, message)
}

// ---------------------------------------------------------------------------
// Gin helpers – write directly to a gin.Context
// ---------------------------------------------------------------------------

// JSON writes the response envelope with HTTP 200 OK.
func JSON(c *gin.Context, r *R) {
	c.JSON(http.StatusOK, r)
}

// Success writes a successful (code=0) response with the given data.
func Success(c *gin.Context, data interface{}) {
	JSON(c, OK(data))
}

// Unauthorized writes a 401 response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(CodeUnauthorized, message))
}

// Forbidden writes a 403 response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	c.AbortWithStatusJSON(http.StatusForbidden, Fail(CodeForbidden, message))
}

// NotFound writes a 404 response.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "not found"
	}
	c.AbortWithStatusJSON(http.StatusNotFound, Fail(CodeNotFound, message))
}

// BadRequest writes a 400 response.
func BadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "bad request"
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, Fail(CodeError, message))
}

// ServerError writes a 500 response.
func ServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, Fail(CodeError, message))
}
