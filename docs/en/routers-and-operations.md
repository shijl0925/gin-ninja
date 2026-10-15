# Routers and Operations

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/routers-and-operations.md)

A `Router` groups related endpoints under one URL prefix and lets them share tags, security requirements, middleware, and version metadata. An operation is one endpoint: HTTP method, path, input struct, typed handler, response type, and operation options.

Most modules start with this shape:

```go
r := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(r, "/", listUsers, ninja.Summary("List users"), ninja.Paginated[UserOut]())
ninja.Post(r, "/", createUser, ninja.Summary("Create user"), ninja.WithTransaction())

api.AddRouter(r)
```

Use router-level options for settings that apply to the whole module, such as tags, auth, and versioning. Use operation-level options for one endpoint, such as summary, response model, pagination, timeout, rate limit, and transactions.

## Create A Router

Minimal form:

```go
r := ninja.NewRouter("/users")
api.AddRouter(r)
```

With OpenAPI grouping:

```go
r := ninja.NewRouter(
    "/users",
    ninja.WithTags("Users"),
    ninja.WithTagDescription("Users", "User management endpoints"),
)
```

`api.AddRouter(r)` mounts all operations already registered on the router into the Gin engine and the OpenAPI document. After a router is mounted, adding operations, middleware, or sub-routers panics. That keeps the route table stable after registration.

If you use controllers, `api.AddController(prefix, controller, opts...)` creates a router internally, calls the controller's `Register` method, and mounts it on the API. The same router options, such as `WithTags`, `WithBearerAuth`, and `WithVersion`, apply there too:

```go
api.AddController("/users", &UserController{db: db},
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

## Router Options

| Option | Effect | Common use |
| --- | --- | --- |
| `ninja.WithTags(tags...)` | Sets default OpenAPI tags for operations on this router | Group Swagger UI by `Users`, `Orders`, and similar resources |
| `ninja.WithTagDescription(tag, description)` | Adds a description for one OpenAPI tag | Explain a resource group |
| `ninja.WithTagDescriptions(map[string]string)` | Adds descriptions for several tags | A router uses multiple tags |
| `ninja.WithSecurity(name, scopes...)` | Adds OpenAPI security metadata only | Docs should show auth, while middleware is mounted elsewhere |
| `ninja.WithSecurityMiddleware(name, mw, scopes...)` | Adds OpenAPI security metadata and router Gin middleware | Custom auth schemes |
| `ninja.WithBearerAuth(mw...)` | Uses the default `bearerAuth` scheme and mounts auth middleware | JWT bearer auth |
| `ninja.WithBasicAuth(mw...)` | Uses the default `basicAuth` scheme and mounts auth middleware | Basic auth |
| `ninja.WithAPIKeyAuth(name, mw...)` | Uses a named API key scheme and mounts auth middleware | `X-API-Key` or gateway keys |
| `ninja.WithOAuth2Auth(scopes...)` | Adds default `oauth2` security metadata only | OAuth2 docs metadata |
| `ninja.WithOAuth2AuthMiddleware(mw, scopes...)` | Adds default `oauth2` security metadata and middleware | OAuth2 scope checks |
| `ninja.WithVersion(version)` | Marks this router as part of a named API version | `/v1/users`, `/v2/users` |

Helpers such as `WithBearerAuth`, `WithBasicAuth`, and `WithAPIKeyAuth` require real Gin middleware. Use `WithSecurity(...)` when you only want OpenAPI security metadata.

The security schemes themselves should be declared in `ninja.Config.SecuritySchemes` so Swagger UI knows how to display `bearerAuth`, `basicAuth`, `oauth2`, or custom API key schemes.

## Router Middleware

A router supports two middleware styles.

`Use` receives gin-ninja's typed context:

```go
r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))

r.Use(func(ctx *ninja.Context) error {
    if ctx.GetHeader("X-Admin") != "true" {
        return ninja.NewError(403, "admin only")
    }
    return nil
})
```

`UseGin` receives raw `gin.HandlerFunc` middleware, which is useful when reusing existing Gin middleware:

```go
r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
r.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))
```

At runtime, router Gin middleware runs first, then router typed middleware, then the operation handler. Operation-level auth middleware applies only to that operation.

## Nested Routers

`AddRouter` mounts a child router under a parent router:

```go
apiRouter := ninja.NewRouter("/api")
v1 := ninja.NewRouter("/v1")
users := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(users, "/:id", getUser, ninja.Summary("Get user"))

v1.AddRouter(users)
apiRouter.AddRouter(v1)
api.AddRouter(apiRouter)
```

The final path is `/api/v1/users/:id`. Child routers inherit the path nesting. Keep security, tags, and version defaults close to the business module when possible, so readers do not have to chase settings through a deep tree.

## Register Operations

gin-ninja registers operations with Go generic functions, not Python-style decorators.

| Function | HTTP method | Handler signature | Default success status |
| --- | --- | --- | --- |
| `ninja.Get[TIn, TOut](r, path, handler, opts...)` | `GET` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Post[TIn, TOut](r, path, handler, opts...)` | `POST` | `func(*ninja.Context, *TIn) (*TOut, error)` | `201 Created` |
| `ninja.Put[TIn, TOut](r, path, handler, opts...)` | `PUT` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Patch[TIn, TOut](r, path, handler, opts...)` | `PATCH` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Delete[TIn](r, path, handler, opts...)` | `DELETE` | `func(*ninja.Context, *TIn) error` | `204 No Content` |

`GET` operations also register the matching `HEAD` route. `DELETE` has no response body by default, so its handler only returns `error`.

```go
type GetUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    user, err := repo.Get(ctx.Request.Context(), in.UserID)
    if err != nil {
        return nil, err
    }
    return toUserOut(user), nil
}

ninja.Get(r, "/:id", getUser,
    ninja.Summary("Get user"),
    ninja.Response(404, "User not found", &ErrorOut{}),
)
```

## Documentation Options

| Option | Effect |
| --- | --- |
| `ninja.Summary(text)` | Sets the OpenAPI summary |
| `ninja.Description(text)` | Sets the OpenAPI description |
| `ninja.OperationID(id)` | Sets a stable OpenAPI `operationId` |
| `ninja.Tags(tags...)` | Overrides tags for this operation |
| `ninja.TagDescription(tag, description)` | Adds a description for a tag used by this operation |
| `ninja.Deprecated()` | Marks the operation as deprecated |
| `ninja.ExcludeFromDocs()` | Hides the operation from OpenAPI |

Router `WithTags(...)` provides defaults. Operation `Tags(...)` overrides them:

```go
r := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(r, "/internal/health", healthz,
    ninja.Tags("Internal"),
    ninja.ExcludeFromDocs(),
)
```

## Response Options

| Option | Effect |
| --- | --- |
| `ninja.SuccessStatus(code)` | Changes the successful response status code |
| `ninja.Response(status, description, model)` | Adds an extra OpenAPI response |
| `ninja.ResponseModel[T]()` | Uses a model schema for the success response and applies it at runtime |
| `ninja.ResponseSchema(descriptor)` | Uses a reusable model schema descriptor |
| `ninja.Paginated[T]()` | Declares a standard paginated success response |
| `ninja.PaginatedSchema(descriptor)` | Declares a paginated response with a schema descriptor |
| `ninja.CursorPaginated[T]()` | Declares a cursor-paginated success response |
| `ninja.CursorPaginatedSchema(descriptor)` | Declares a cursor-paginated response with a schema descriptor |
| `ninja.PaginatedResponse[T](status, description)` | Adds an extra paginated OpenAPI response |
| `ninja.CursorPaginatedResponse[T](status, description)` | Adds an extra cursor-paginated OpenAPI response |

List endpoints usually return `*pagination.Page[T]` and use `Paginated[T]()`:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    return service.ListUsers(ctx.Request.Context(), in)
}

ninja.Get(r, "/", listUsers,
    ninja.Summary("List users"),
    ninja.Paginated[UserOut](),
)
```

When a handler returns a GORM model but the public response must hide fields, use `ResponseModel` or `ResponseSchema`:

```go
ninja.Get(r, "/:id", getUser,
    ninja.ResponseModel[UserOut](),
)
```

## Security Options

| Option | Effect |
| --- | --- |
| `ninja.Security(name, scopes...)` | Adds OpenAPI security metadata for this operation only |
| `ninja.SecurityMiddleware(name, mw, scopes...)` | Adds security metadata and operation Gin middleware |
| `ninja.BearerAuth(mw...)` | Uses default `bearerAuth` security and middleware |
| `ninja.BasicAuth(mw...)` | Uses default `basicAuth` security and middleware |
| `ninja.APIKeyAuth(name, mw...)` | Uses a named API key security scheme and middleware |
| `ninja.OAuth2Auth(scopes...)` | Adds default `oauth2` security metadata only |
| `ninja.OAuth2AuthMiddleware(mw, scopes...)` | Adds default `oauth2` security metadata and middleware |

When most endpoints in a module require auth, put it on the router:

```go
r := ninja.NewRouter("/users",
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

When only one endpoint needs extra auth, put it on the operation:

```go
adminOnly := middleware.JWTAuthWithConfig(cfg.JWT)

ninja.Delete(r, "/:id", deleteUser,
    ninja.Summary("Delete user"),
    ninja.BearerAuth(adminOnly),
)
```

## Runtime Options

| Option | Effect |
| --- | --- |
| `ninja.Timeout(d)` | Sets a cooperative timeout for this operation |
| `ninja.RateLimit(requestsPerSecond, burst...)` | Sets an in-memory token bucket rate limit per client IP for this operation |
| `ninja.WithTransaction()` | Wraps this operation in a request-scoped database transaction |

`Timeout` cancels the request context and allows an early timeout response. Long-running work should still observe `ctx.Request.Context().Done()`.

```go
ninja.Get(r, "/slow", slowReport,
    ninja.Timeout(3*time.Second),
)
```

`WithTransaction()` requires transaction handlers on the API and `orm.Middleware(db)`:

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})
api.UseGin(orm.Middleware(db))

ninja.Post(r, "/", createUser,
    ninja.WithTransaction(),
)
```

## Cache Options

| Option | Effect |
| --- | --- |
| `ninja.Cache(ttl, opts...)` | Enables route-level response caching and ETag by default |
| `ninja.CacheWithMaxBodyBytes(max)` | Limits how much response body data is buffered for cache storage and ETag generation |
| `ninja.CacheControl(value)` | Sets the `Cache-Control` header on successful responses |
| `ninja.ETag()` | Generates ETag headers for successful responses |

Cache stable read endpoints, not writes with side effects:

```go
ninja.Get(r, "/:id", getUser,
    ninja.Cache(30*time.Second),
    ninja.CacheControl("public, max-age=30"),
)
```

## Placement Rules

| Setting | Recommended place |
| --- | --- |
| Module URL prefix | `ninja.NewRouter("/users")` |
| Swagger grouping | Router `WithTags(...)` |
| Module-level auth | Router `WithBearerAuth(...)` or `WithSecurity(...)` |
| Per-endpoint docs | Operation `Summary(...)` and `Description(...)` |
| Per-endpoint response shape | Operation `ResponseModel(...)` and `Paginated(...)` |
| Per-endpoint transaction | Operation `WithTransaction()` |
| Per-endpoint rate limit or timeout | Operation `RateLimit(...)` and `Timeout(...)` |
| API version | Router `WithVersion(...)` |

A practical rule: when reading route registration code, you should quickly see the endpoint path, input, response, documentation name, auth requirement, and important runtime behavior. Put shared rules on the router and endpoint-specific details on the operation.

Previous: [ModelSchema Advanced Usage](./model-schema.md) | Next: [Project and CRUD Scaffolding](./scaffolding.md)
