package ninja

import "strconv"

// ResponseCode is the business-level result code used by response envelopes.
type ResponseCode int

// Standard business-level response codes.
const (
	// CONTINUE means request received, please continue.
	CONTINUE ResponseCode = 100
	// SWITCHING_PROTOCOLS means switching to new protocol; obey Upgrade header.
	SWITCHING_PROTOCOLS ResponseCode = 101
	// PROCESSING means processing.
	PROCESSING ResponseCode = 102

	// OK means request fulfilled, document follows.
	OK ResponseCode = 200
	// CREATED means document created, URL follows.
	CREATED ResponseCode = 201
	// ACCEPTED means request accepted, processing continues off-line.
	ACCEPTED ResponseCode = 202
	// NON_AUTHORITATIVE_INFORMATION means request fulfilled from cache.
	NON_AUTHORITATIVE_INFORMATION ResponseCode = 203
	// NO_CONTENT means request fulfilled, nothing follows.
	NO_CONTENT ResponseCode = 204
	// RESET_CONTENT means clear input form for further input.
	RESET_CONTENT ResponseCode = 205
	// PARTIAL_CONTENT means partial content follows.
	PARTIAL_CONTENT ResponseCode = 206
	// MULTI_STATUS means multi-status.
	MULTI_STATUS ResponseCode = 207
	// ALREADY_REPORTED means already reported.
	ALREADY_REPORTED ResponseCode = 208
	// IM_USED means IM used.
	IM_USED ResponseCode = 226

	// MULTIPLE_CHOICES means object has several resources -- see URI list.
	MULTIPLE_CHOICES ResponseCode = 300
	// MOVED_PERMANENTLY means object moved permanently -- see URI list.
	MOVED_PERMANENTLY ResponseCode = 301
	// FOUND means object moved temporarily -- see URI list.
	FOUND ResponseCode = 302
	// SEE_OTHER means object moved -- see Method and URL list.
	SEE_OTHER ResponseCode = 303
	// NOT_MODIFIED means document has not changed since given time.
	NOT_MODIFIED ResponseCode = 304
	// USE_PROXY means you must use proxy specified in Location to access this resource.
	USE_PROXY ResponseCode = 305
	// TEMPORARY_REDIRECT means object moved temporarily -- see URI list.
	TEMPORARY_REDIRECT ResponseCode = 307
	// PERMANENT_REDIRECT means object moved temporarily -- see URI list.
	PERMANENT_REDIRECT ResponseCode = 308

	// BAD_REQUEST means bad request syntax or unsupported method.
	BAD_REQUEST ResponseCode = 400
	// UNAUTHORIZED means no permission -- see authorization schemes.
	UNAUTHORIZED ResponseCode = 401
	// PAYMENT_REQUIRED means no payment -- see charging schemes.
	PAYMENT_REQUIRED ResponseCode = 402
	// FORBIDDEN means request forbidden -- authorization will not help.
	FORBIDDEN ResponseCode = 403
	// NOT_FOUND means nothing matches the given URI.
	NOT_FOUND ResponseCode = 404
	// METHOD_NOT_ALLOWED means specified method is invalid for this resource.
	METHOD_NOT_ALLOWED ResponseCode = 405
	// NOT_ACCEPTABLE means URI not available in preferred format.
	NOT_ACCEPTABLE ResponseCode = 406
	// PROXY_AUTHENTICATION_REQUIRED means you must authenticate with this proxy before proceeding.
	PROXY_AUTHENTICATION_REQUIRED ResponseCode = 407
	// REQUEST_TIMEOUT means request timed out; try again later.
	REQUEST_TIMEOUT ResponseCode = 408
	// CONFLICT means request conflict.
	CONFLICT ResponseCode = 409
	// GONE means URI no longer exists and has been permanently removed.
	GONE ResponseCode = 410
	// LENGTH_REQUIRED means client must specify Content-Length.
	LENGTH_REQUIRED ResponseCode = 411
	// PRECONDITION_FAILED means precondition in headers is false.
	PRECONDITION_FAILED ResponseCode = 412
	// REQUEST_ENTITY_TOO_LARGE means entity is too large.
	REQUEST_ENTITY_TOO_LARGE ResponseCode = 413
	// REQUEST_URI_TOO_LONG means URI is too long.
	REQUEST_URI_TOO_LONG ResponseCode = 414
	// UNSUPPORTED_MEDIA_TYPE means entity body in unsupported format.
	UNSUPPORTED_MEDIA_TYPE ResponseCode = 415
	// REQUESTED_RANGE_NOT_SATISFIABLE means cannot satisfy request range.
	REQUESTED_RANGE_NOT_SATISFIABLE ResponseCode = 416
	// EXPECTATION_FAILED means expect condition could not be satisfied.
	EXPECTATION_FAILED ResponseCode = 417
	// UNPROCESSABLE_ENTITY means unprocessable entity.
	UNPROCESSABLE_ENTITY ResponseCode = 422
	// LOCKED means locked.
	LOCKED ResponseCode = 423
	// FAILED_DEPENDENCY means failed dependency.
	FAILED_DEPENDENCY ResponseCode = 424
	// UPGRADE_REQUIRED means upgrade required.
	UPGRADE_REQUIRED ResponseCode = 426
	// PRECONDITION_REQUIRED means the origin server requires the request to be conditional.
	PRECONDITION_REQUIRED ResponseCode = 428
	// TOO_MANY_REQUESTS means the user has sent too many requests in a given amount of time ("rate limiting").
	TOO_MANY_REQUESTS ResponseCode = 429
	// REQUEST_HEADER_FIELDS_TOO_LARGE means the server is unwilling to process the request because its header fields are too large.
	REQUEST_HEADER_FIELDS_TOO_LARGE ResponseCode = 431

	// INTERNAL_SERVER_ERROR means server got itself in trouble.
	INTERNAL_SERVER_ERROR ResponseCode = 500
	// NOT_IMPLEMENTED means server does not support this operation.
	NOT_IMPLEMENTED ResponseCode = 501
	// BAD_GATEWAY means invalid responses from another server/proxy.
	BAD_GATEWAY ResponseCode = 502
	// SERVICE_UNAVAILABLE means the server cannot process the request due to a high load.
	SERVICE_UNAVAILABLE ResponseCode = 503
	// GATEWAY_TIMEOUT means the gateway server did not receive a timely response.
	GATEWAY_TIMEOUT ResponseCode = 504
	// HTTP_VERSION_NOT_SUPPORTED means cannot fulfill request.
	HTTP_VERSION_NOT_SUPPORTED ResponseCode = 505
	// VARIANT_ALSO_NEGOTIATES means variant also negotiates.
	VARIANT_ALSO_NEGOTIATES ResponseCode = 506
	// INSUFFICIENT_STORAGE means insufficient storage.
	INSUFFICIENT_STORAGE ResponseCode = 507
	// LOOP_DETECTED means loop detected.
	LOOP_DETECTED ResponseCode = 508
	// NOT_EXTENDED means not extended.
	NOT_EXTENDED ResponseCode = 510
	// NETWORK_AUTHENTICATION_REQUIRED means the client needs to authenticate to gain network access.
	NETWORK_AUTHENTICATION_REQUIRED ResponseCode = 511

	// CodeOK is kept for compatibility; use OK for new code.
	CodeOK = OK
	// CodeError is kept for compatibility; use INTERNAL_SERVER_ERROR for new code.
	CodeError = INTERNAL_SERVER_ERROR
	// CodeUnauthorized is kept for compatibility; use UNAUTHORIZED for new code.
	CodeUnauthorized = UNAUTHORIZED
	// CodeForbidden is kept for compatibility; use FORBIDDEN for new code.
	CodeForbidden = FORBIDDEN
	// CodeNotFound is kept for compatibility; use NOT_FOUND for new code.
	CodeNotFound = NOT_FOUND
	// CodeValidation is kept for compatibility; use UNPROCESSABLE_ENTITY for new code.
	CodeValidation = UNPROCESSABLE_ENTITY
)

// String returns the decimal string representation of the response code.
func (c ResponseCode) String() string {
	return strconv.Itoa(c.Int())
}

// Int returns the integer representation of the response code.
func (c ResponseCode) Int() int {
	return int(c)
}

func responseCodeFromStatus(status int) ResponseCode {
	return ResponseCode(status)
}

// Text returns the standard reason phrase for the response code.
func (c ResponseCode) Text() string {
	switch c {
	case CONTINUE:
		return "Continue"
	case SWITCHING_PROTOCOLS:
		return "Switching Protocols"
	case PROCESSING:
		return "Processing"
	case OK:
		return "OK"
	case CREATED:
		return "Created"
	case ACCEPTED:
		return "Accepted"
	case NON_AUTHORITATIVE_INFORMATION:
		return "Non-Authoritative Information"
	case NO_CONTENT:
		return "No Content"
	case RESET_CONTENT:
		return "Reset Content"
	case PARTIAL_CONTENT:
		return "Partial Content"
	case MULTI_STATUS:
		return "Multi-Status"
	case ALREADY_REPORTED:
		return "Already Reported"
	case IM_USED:
		return "IM Used"
	case MULTIPLE_CHOICES:
		return "Multiple Choices"
	case MOVED_PERMANENTLY:
		return "Moved Permanently"
	case FOUND:
		return "Found"
	case SEE_OTHER:
		return "See Other"
	case NOT_MODIFIED:
		return "Not Modified"
	case USE_PROXY:
		return "Use Proxy"
	case TEMPORARY_REDIRECT:
		return "Temporary Redirect"
	case PERMANENT_REDIRECT:
		return "Permanent Redirect"
	case BAD_REQUEST:
		return "Bad Request"
	case UNAUTHORIZED:
		return "Unauthorized"
	case PAYMENT_REQUIRED:
		return "Payment Required"
	case FORBIDDEN:
		return "Forbidden"
	case NOT_FOUND:
		return "Not Found"
	case METHOD_NOT_ALLOWED:
		return "Method Not Allowed"
	case NOT_ACCEPTABLE:
		return "Not Acceptable"
	case PROXY_AUTHENTICATION_REQUIRED:
		return "Proxy Authentication Required"
	case REQUEST_TIMEOUT:
		return "Request Timeout"
	case CONFLICT:
		return "Conflict"
	case GONE:
		return "Gone"
	case LENGTH_REQUIRED:
		return "Length Required"
	case PRECONDITION_FAILED:
		return "Precondition Failed"
	case REQUEST_ENTITY_TOO_LARGE:
		return "Request Entity Too Large"
	case REQUEST_URI_TOO_LONG:
		return "Request-URI Too Long"
	case UNSUPPORTED_MEDIA_TYPE:
		return "Unsupported Media Type"
	case REQUESTED_RANGE_NOT_SATISFIABLE:
		return "Requested Range Not Satisfiable"
	case EXPECTATION_FAILED:
		return "Expectation Failed"
	case UNPROCESSABLE_ENTITY:
		return "Unprocessable Entity"
	case LOCKED:
		return "Locked"
	case FAILED_DEPENDENCY:
		return "Failed Dependency"
	case UPGRADE_REQUIRED:
		return "Upgrade Required"
	case PRECONDITION_REQUIRED:
		return "Precondition Required"
	case TOO_MANY_REQUESTS:
		return "Too Many Requests"
	case REQUEST_HEADER_FIELDS_TOO_LARGE:
		return "Request Header Fields Too Large"
	case INTERNAL_SERVER_ERROR:
		return "Internal Server Error"
	case NOT_IMPLEMENTED:
		return "Not Implemented"
	case BAD_GATEWAY:
		return "Bad Gateway"
	case SERVICE_UNAVAILABLE:
		return "Service Unavailable"
	case GATEWAY_TIMEOUT:
		return "Gateway Timeout"
	case HTTP_VERSION_NOT_SUPPORTED:
		return "HTTP Version Not Supported"
	case VARIANT_ALSO_NEGOTIATES:
		return "Variant Also Negotiates"
	case INSUFFICIENT_STORAGE:
		return "Insufficient Storage"
	case LOOP_DETECTED:
		return "Loop Detected"
	case NOT_EXTENDED:
		return "Not Extended"
	case NETWORK_AUTHENTICATION_REQUIRED:
		return "Network Authentication Required"
	default:
		return ""
	}
}

// Description returns the standard description for the response code.
func (c ResponseCode) Description() string {
	switch c {
	case CONTINUE:
		return "Request received, please continue"
	case SWITCHING_PROTOCOLS:
		return "Switching to new protocol; obey Upgrade header"
	case PROCESSING:
		return "Processing"
	case OK:
		return "Request fulfilled, document follows"
	case CREATED:
		return "Document created, URL follows"
	case ACCEPTED:
		return "Request accepted, processing continues off-line"
	case NON_AUTHORITATIVE_INFORMATION:
		return "Request fulfilled from cache"
	case NO_CONTENT:
		return "Request fulfilled, nothing follows"
	case RESET_CONTENT:
		return "Clear input form for further input"
	case PARTIAL_CONTENT:
		return "Partial content follows"
	case MULTI_STATUS:
		return "Multi-Status"
	case ALREADY_REPORTED:
		return "Already Reported"
	case IM_USED:
		return "IM Used"
	case MULTIPLE_CHOICES:
		return "Object has several resources -- see URI list"
	case MOVED_PERMANENTLY:
		return "Object moved permanently -- see URI list"
	case FOUND:
		return "Object moved temporarily -- see URI list"
	case SEE_OTHER:
		return "Object moved -- see Method and URL list"
	case NOT_MODIFIED:
		return "Document has not changed since given time"
	case USE_PROXY:
		return "You must use proxy specified in Location to access this resource"
	case TEMPORARY_REDIRECT:
		return "Object moved temporarily -- see URI list"
	case PERMANENT_REDIRECT:
		return "Object moved temporarily -- see URI list"
	case BAD_REQUEST:
		return "Bad request syntax or unsupported method"
	case UNAUTHORIZED:
		return "No permission -- see authorization schemes"
	case PAYMENT_REQUIRED:
		return "No payment -- see charging schemes"
	case FORBIDDEN:
		return "Request forbidden -- authorization will not help"
	case NOT_FOUND:
		return "Nothing matches the given URI"
	case METHOD_NOT_ALLOWED:
		return "Specified method is invalid for this resource"
	case NOT_ACCEPTABLE:
		return "URI not available in preferred format"
	case PROXY_AUTHENTICATION_REQUIRED:
		return "You must authenticate with this proxy before proceeding"
	case REQUEST_TIMEOUT:
		return "Request timed out; try again later"
	case CONFLICT:
		return "Request conflict"
	case GONE:
		return "URI no longer exists and has been permanently removed"
	case LENGTH_REQUIRED:
		return "Client must specify Content-Length"
	case PRECONDITION_FAILED:
		return "Precondition in headers is false"
	case REQUEST_ENTITY_TOO_LARGE:
		return "Entity is too large"
	case REQUEST_URI_TOO_LONG:
		return "URI is too long"
	case UNSUPPORTED_MEDIA_TYPE:
		return "Entity body in unsupported format"
	case REQUESTED_RANGE_NOT_SATISFIABLE:
		return "Cannot satisfy request range"
	case EXPECTATION_FAILED:
		return "Expect condition could not be satisfied"
	case UNPROCESSABLE_ENTITY:
		return "Unprocessable Entity"
	case LOCKED:
		return "Locked"
	case FAILED_DEPENDENCY:
		return "Failed Dependency"
	case UPGRADE_REQUIRED:
		return "Upgrade Required"
	case PRECONDITION_REQUIRED:
		return "The origin server requires the request to be conditional"
	case TOO_MANY_REQUESTS:
		return "The user has sent too many requests in a given amount of time (\"rate limiting\")"
	case REQUEST_HEADER_FIELDS_TOO_LARGE:
		return "The server is unwilling to process the request because its header fields are too large"
	case INTERNAL_SERVER_ERROR:
		return "Server got itself in trouble"
	case NOT_IMPLEMENTED:
		return "Server does not support this operation"
	case BAD_GATEWAY:
		return "Invalid responses from another server/proxy"
	case SERVICE_UNAVAILABLE:
		return "The server cannot process the request due to a high load"
	case GATEWAY_TIMEOUT:
		return "The gateway server did not receive a timely response"
	case HTTP_VERSION_NOT_SUPPORTED:
		return "Cannot fulfill request"
	case VARIANT_ALSO_NEGOTIATES:
		return "Variant Also Negotiates"
	case INSUFFICIENT_STORAGE:
		return "Insufficient Storage"
	case LOOP_DETECTED:
		return "Loop Detected"
	case NOT_EXTENDED:
		return "Not Extended"
	case NETWORK_AUTHENTICATION_REQUIRED:
		return "The client needs to authenticate to gain network access"
	default:
		return ""
	}
}

// R is the standard response envelope.
//
//	{"code": 200, "message": "success", "data": null}
type R struct {
	// Code is the business-level result code (200 = success).
	Code ResponseCode `json:"code"`
	// Message is a human-readable result description.
	Message string `json:"message"`
	// Data contains the response payload (can be any JSON-serialisable value).
	Data interface{} `json:"data"`
}
