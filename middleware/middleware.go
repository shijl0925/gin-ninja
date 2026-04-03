// Package middleware provides production-ready HTTP middleware for gin-ninja
// applications.
//
// Available middleware:
//   - RequestID  – injects a unique X-Request-ID header into every request
//   - Logger     – structured request/response logging via Zap
//   - Recovery   – recovers from panics and returns a 500 error response
//   - CORS       – configurable Cross-Origin Resource Sharing
//   - JWTAuth    – validates Bearer tokens and stores claims in the context
//   - RequirePermissions – checks route-level RBAC permission codes
package middleware

// This file contains only the package declaration and doc comment.
// Individual middleware are defined in their own files.
