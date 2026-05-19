// Package response provides a standardised JSON response envelope for
// gin-ninja APIs, following the common pattern used in Go admin backends:
//
//	{"code": 200, "message": "success", "data": {...}}
//
// Usage:
//
//	func listUsers(ctx *ninja.Context, in *ListInput) (*response.R, error) {
//	    users, _ := svc.List()
//	    return response.OK(users), nil
//	}
package response

import (
	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja"
)

// ResponseCode is the business-level result code used by response envelopes.
type ResponseCode = ninja.ResponseCode

// Standard business-level response codes.
const (
	// CodeOK indicates a successful operation.
	CodeOK = ninja.CodeOK
	// CodeError is the generic business error code.
	CodeError = ninja.CodeError
	// CodeUnauthorized indicates missing or invalid authentication.
	CodeUnauthorized = ninja.CodeUnauthorized
	// CodeForbidden indicates the caller lacks sufficient permissions.
	CodeForbidden = ninja.CodeForbidden
	// CodeNotFound indicates the requested resource was not found.
	CodeNotFound = ninja.CodeNotFound
	// CodeValidation indicates a request validation failure.
	CodeValidation = ninja.CodeValidation
)

// R is the standard response envelope.
//
//	{"code": 200, "message": "success", "data": null}
type R = ninja.R

// OK returns a successful response containing the given data.
func OK(data interface{}) *R {
	return &R{Code: CodeOK, Message: "success", Data: data}
}

// OKWithMessage returns a successful response with a custom message.
func OKWithMessage(msg string, data interface{}) *R {
	return &R{Code: CodeOK, Message: msg, Data: data}
}

// Fail returns an error response with the given code and message.
func Fail[C ~int](code C, message string) *R {
	return &R{Code: ResponseCode(code), Message: message, Data: nil}
}

// FailWithData returns an error response that also carries a data payload.
func FailWithData[C ~int](code C, message string, data interface{}) *R {
	return &R{Code: ResponseCode(code), Message: message, Data: data}
}

// Error returns a generic error response (code = "-1").
func Error(message string) *R {
	return Fail(CodeError, message)
}

// ---------------------------------------------------------------------------
// Gin helpers – write directly to a gin.Context
// ---------------------------------------------------------------------------

// JSON writes the response envelope with HTTP 200 OK.
func JSON(c *gin.Context, r *R) {
	c.JSON(ninja.OK.Int(), r)
}

// Success writes a successful (code=200) response with the given data.
func Success(c *gin.Context, data interface{}) {
	JSON(c, OK(data))
}

// Unauthorized writes a 401 response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	c.AbortWithStatusJSON(ninja.UNAUTHORIZED.Int(), Fail(CodeUnauthorized, message))
}

// Forbidden writes a 403 response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	c.AbortWithStatusJSON(ninja.FORBIDDEN.Int(), Fail(CodeForbidden, message))
}

// NotFound writes a 404 response.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "not found"
	}
	c.AbortWithStatusJSON(ninja.NOT_FOUND.Int(), Fail(CodeNotFound, message))
}

// BadRequest writes a 400 response.
func BadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "bad request"
	}
	c.AbortWithStatusJSON(ninja.BAD_REQUEST.Int(), Fail(ninja.BAD_REQUEST, message))
}

// ServerError writes a 500 response.
func ServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	c.AbortWithStatusJSON(ninja.INTERNAL_SERVER_ERROR.Int(), Fail(CodeError, message))
}
