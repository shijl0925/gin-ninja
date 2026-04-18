# API patterns

## Core primitives

- `ninja.New(ninja.Config{...})` creates the API root and mounts the default homepage, Swagger UI, and OpenAPI endpoints.
- `api.UseGin(...)` attaches raw Gin middleware such as logging, CORS, request IDs, auth, and DB context wiring.
- `ninja.NewRouter("/prefix", ...)` groups routes by prefix, tags, security, and version.
- `ninja.Get/Post/Put/Patch/Delete(...)` register typed handlers.
- `ninja.SSE(...)` and `ninja.WebSocket(...)` cover streaming routes.

## Handler shape

Prefer handlers of the form:

- `func(ctx *ninja.Context, in *Input) (*Output, error)`
- `func(ctx *ninja.Context, in *Input) error` for `Delete`
- `func(ctx *ninja.Context, in *Input, stream *ninja.SSEStream) error` for SSE
- `func(ctx *ninja.Context, in *Input, conn *ninja.WebSocketConn) error` for WebSocket

Keep request and response structs separate from persistence models unless the API contract is intentionally identical.

## Request binding tags

- path params: ``path:"id"``
- query params: ``form:"page"``
- headers: ``header:"X-Request-ID"``
- cookies: ``cookie:"session"``
- JSON body: ``json:"name"``
- multipart files: ``file:"avatar"``
- validation: ``binding:"required,email,min=1"``
- defaults for query/header/cookie: ``default:"20"``

For list endpoints, prefer embedding `pagination.PageInput` when the response is paginated.

## Common route options

- docs metadata: `Summary`, `Description`, `OperationID`, `Tags`, `TagDescription`
- auth/docs security: `BearerAuth`, `Security`, router-level `WithBearerAuth`, `WithSecurity`
- docs control: `Response`, `PaginatedResponse`, `ExcludeFromDocs`, `Deprecated`
- behavior: `SuccessStatus`, `Timeout`, `RateLimit`, `WithTransaction`
- cache/read optimization: `Cache`, `CacheControl`, `ETag`
- versioning: router-level `WithVersion`

## Built-in feature map

- middleware: `middleware/`
- pagination: `pagination/`
- filtering: `filter/`
- ordering: `order/`
- ORM integration: `orm/`
- config loading: `settings/`
- bootstrap helpers: `bootstrap/`
- standard envelope/logging/i18n helpers: `pkg/response`, `pkg/logger`, `pkg/i18n`

## Good defaults

1. Put infrastructure middleware on `api.UseGin(...)`.
2. Put domain grouping, tags, auth, and versioning on routers.
3. Put endpoint-specific behavior on operation options.
4. Return framework errors such as `ninja.NotFoundError()` for HTTP semantics instead of ad-hoc response writing.
5. Leave raw `api.Engine()` routes for non-framework endpoints only, such as simple health checks or special cases.

## Best example starting points

- minimal CRUD shape: `examples/basic/main.go`
- app/config wiring: `examples/users/main.go`
- advanced feature surface: `examples/full/main.go` and `examples/full/app/`
