# Errors and context helpers

## Error families

- `*ninja.Error` is the protocol-level HTTP error type.
- Use builtin constructors for standard HTTP semantics:
  - `ninja.BadRequestError()`
  - `ninja.UnauthorizedError()`
  - `ninja.ForbiddenError()`
  - `ninja.NotFoundError()`
  - `ninja.ConflictError()`
  - `ninja.InternalError()`
- Use `ninja.NewError(status, message)` or `ninja.NewErrorWithCode(status, code, message)` for custom HTTP errors.
- Use the predicate helpers when branching on returned errors:
  - `ninja.IsBadRequest(err)`
  - `ninja.IsUnauthorized(err)`
  - `ninja.IsForbidden(err)`
  - `ninja.IsNotFound(err)`
  - `ninja.IsConflict(err)`
  - `ninja.IsInternal(err)`

## Business errors

- `ninja.BusinessError` is for domain or application-level failures that should still return HTTP 200.
- Prefer it for business rule failures such as disabled accounts, quota limits, or state-machine violations.
- Create them with:
  - `ninja.NewBusinessError(code, message)`
  - `ninja.NewBusinessErrorWithDetail(code, message, detail)`
- Response envelope:
  - `{"code": <non-zero int>, "message": "...", "data": ...}`
- Use `*ninja.Error` instead when the failure should change the HTTP status code.

## Validation errors

- `ninja.ValidationError` represents request validation failures.
- It carries field-level details via `[]ninja.FieldError`.
- In normal handler flows this is usually produced by binding and validation on typed input structs rather than constructed manually.
- Typical triggers:
  - missing `binding:"required"` fields
  - invalid validator rules such as `email`, `min`, or similar tags
  - malformed bound request data
- It is written as HTTP 422 with a `VALIDATION_ERROR` payload.

## Error mapping

- Prefer `api.RegisterErrorMapper(...)` to translate domain/infrastructure errors into framework errors for one API instance.
- Good fits:
  - convert storage-layer not-found errors into `ninja.NotFoundError()`
  - map timeout or dependency failures into stable HTTP responses
  - keep handler bodies thin by centralizing cross-cutting error translation
- Returning `nil` means the mapper did not handle the error.
- The package-level `ninja.RegisterErrorMapper(...)` exists for process-wide legacy behavior, but the per-instance API method is the preferred default.

## Context helpers

`*ninja.Context` wraps `*gin.Context` and also satisfies `context.Context`.

- request metadata:
  - `ctx.RequestID()` -> value from the RequestID middleware
  - `ctx.GetUserID()` -> authenticated user ID from JWT claims when auth middleware stored it
  - `ctx.Locale()` -> negotiated locale, defaulting to `"en"`
  - `ctx.T(key, args...)` -> translate through `pkg/i18n` using the request locale
- JSON shortcuts:
  - `ctx.JSON200(obj)`
  - `ctx.JSON201(obj)`
  - `ctx.JSON204()`
- auth shortcuts:
  - `ctx.Forbidden(message)`
  - `ctx.Unauthorized(message)`
- context bridging:
  - `ctx.StdContext()` returns the request's standard-library context
  - `ctx.Value(...)`, `ctx.Done()`, `ctx.Err()`, and `ctx.Deadline()` proxy to the request context when needed

## Good defaults

1. Return framework errors instead of writing ad-hoc JSON error bodies in handlers.
2. Use `BusinessError` only when the contract intentionally keeps HTTP 200 for business failures.
3. Let typed request structs and binding tags produce validation behavior automatically.
4. Prefer `ctx.T(...)` and `ctx.Locale()` over custom locale plumbing.
5. Prefer `api.RegisterErrorMapper(...)` when the same translation rule appears in multiple handlers.
