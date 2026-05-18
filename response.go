package ninja

import "strconv"

// ResponseCode is the business-level result code used by response envelopes.
type ResponseCode string

// Standard business-level response codes.
const (
	// CodeOK indicates a successful operation.
	CodeOK ResponseCode = "0"
	// CodeError is the generic business error code.
	CodeError ResponseCode = "-1"
	// CodeUnauthorized indicates missing or invalid authentication.
	CodeUnauthorized ResponseCode = "401"
	// CodeForbidden indicates the caller lacks sufficient permissions.
	CodeForbidden ResponseCode = "403"
	// CodeNotFound indicates the requested resource was not found.
	CodeNotFound ResponseCode = "404"
	// CodeValidation indicates a request validation failure.
	CodeValidation ResponseCode = "422"
)

// String returns the string representation of the response code.
func (c ResponseCode) String() string {
	return string(c)
}

// Int parses the response code as an integer.
func (c ResponseCode) Int() (int, error) {
	return strconv.Atoi(c.String())
}

func responseCodeFromStatus(status int) ResponseCode {
	return ResponseCode(strconv.Itoa(status))
}

// R is the standard response envelope.
//
//	{"code": "0", "message": "success", "data": null}
type R struct {
	// Code is the business-level result code ("0" = success).
	Code ResponseCode `json:"code"`
	// Message is a human-readable result description.
	Message string `json:"message"`
	// Data contains the response payload (can be any JSON-serialisable value).
	Data interface{} `json:"data"`
}
