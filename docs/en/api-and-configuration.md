# NinjaAPI and Instance Configuration

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/api-and-configuration.md)

`NinjaAPI` is the entry point of a gin-ninja application. It wraps a `*gin.Engine` and manages global middleware, router mounting, controller mounting, OpenAPI generation, docs UI, lifecycle hooks, and the HTTP server.

Minimal shape:

```go
api := ninja.New(ninja.Config{
    Title:   "Demo API",
    Version: "1.0.0",
})

api.AddRouter(usersRouter)
log.Fatal(api.Run(":8080"))
```

`ninja.Config` configures the framework instance created by `ninja.New`. `settings.Config` is application configuration loaded from YAML, environment variables, and overrides. In real projects, you usually load `settings.Config` first, then copy the fields that affect framework behavior into `ninja.Config`.

## What NinjaAPI Manages

| Capability | Entry point |
| --- | --- |
| Create an API instance | `ninja.New(ninja.Config{...})` |
| Add global Gin middleware | `api.UseGin(...)` |
| Mount a business router | `api.AddRouter(router)` |
| Mount a controller | `api.AddController(prefix, controller, opts...)` |
| Access the underlying Gin engine | `api.Engine()` |
| Use as an `http.Handler` | `api.Handler()` |
| Start and handle OS signals | `api.Run(":8080")` |
| Serve with a custom listener | `api.Serve(listener)` |
| Gracefully stop the server | `api.Shutdown(ctx)` |
| Register startup/shutdown hooks | `api.OnStartup(...)`, `api.OnShutdown(...)` |
| Register error mapping | `api.RegisterErrorMapper(...)` |

A fuller startup skeleton:

```go
cfg := settings.MustLoad("config.yaml")
db := bootstrap.MustInitDB(&cfg.Database)

api := ninja.New(ninja.Config{
    Title:               cfg.App.Name,
    Version:             cfg.App.Version,
    Description:         "Internal user service",
    Settings:            cfg,
    TransactionHandlers: orm.TransactionHandlers(),
    DisableGinDefault:   true,
    ReadTimeout:         15 * time.Second,
    WriteTimeout:        30 * time.Second,
})

api.UseGin(
    gin.Logger(),
    gin.Recovery(),
    middleware.RequestID(),
    middleware.CORSFromConfig(cfg.CORS),
    orm.Middleware(db),
)

api.AddRouter(NewUserRouter(db))
log.Fatal(api.Run(fmt.Sprintf(":%d", cfg.Server.Port)))
```

## Creation Defaults

`ninja.New` fills empty fields with safe defaults:

| Field | Default |
| --- | --- |
| `Title` | `"Gin Ninja API"` |
| `Version` | `"1.0.0"` |
| `DocsURL` | `"/docs"` |
| `OpenAPIURL` | `"/openapi.json"` |
| `HomepageURL` | `"/"` |
| `Docs` | `ninja.Swagger()` |
| `ReadTimeout` | `15 * time.Second` |
| `WriteTimeout` | `30 * time.Second` |
| `IdleTimeout` | `60 * time.Second` |
| `GracefulShutdownTimeout` | `10 * time.Second` |

When `DisableOpenAPI: true` is set, the docs UI is also disabled because Swagger UI/ReDoc needs the OpenAPI JSON route.

## OpenAPI Info Options

These fields populate OpenAPI `info` or `servers`:

| Field | Effect |
| --- | --- |
| `Title` | API title shown at the top of Swagger UI/ReDoc |
| `Version` | API version string |
| `Description` | Long API description |
| `TermsOfService` | Terms of service URL |
| `Contact` | OpenAPI contact information |
| `LicenseInfo` | OpenAPI license information |
| `Servers` | OpenAPI server list |

Example:

```go
api := ninja.New(ninja.Config{
    Title:          "Billing API",
    Version:        "1.2.0",
    Description:    "APIs for invoices, payments, and subscriptions.",
    TermsOfService: "https://example.com/terms",
    Contact: &ninja.Contact{
        Name:  "API Support",
        Email: "support@example.com",
    },
    LicenseInfo: &ninja.LicenseInfo{
        Name: "MIT",
        URL:  "https://opensource.org/licenses/MIT",
    },
    Servers: []ninja.Server{
        {URL: "https://api.example.com", Description: "Production"},
        {URL: "https://staging-api.example.com", Description: "Staging"},
    },
})
```

## Docs And OpenAPI Routes

| Field | Effect |
| --- | --- |
| `DocsURL` | Docs UI path, default `/docs` |
| `Docs` | Docs UI renderer, default `ninja.Swagger()` |
| `DisableDocs` | Disables the docs UI route |
| `HideDocsShortcut` | Hides only the Docs button on the homepage |
| `OpenAPIURL` | OpenAPI JSON path, default `/openapi.json` |
| `DisableOpenAPI` | Disables OpenAPI JSON and the docs UI |

Move Swagger UI to `/swagger`:

```go
api := ninja.New(ninja.Config{
    DocsURL:    "/swagger",
    OpenAPIURL: "/schema/openapi.json",
})
```

Use ReDoc:

```go
api := ninja.New(ninja.Config{
    Docs: ninja.Redoc(),
})
```

Disable docs and OpenAPI in production:

```go
api := ninja.New(ninja.Config{
    DisableOpenAPI: true,
})
```

OpenAPI documents are generated mainly from:

- `ninja.Config`: API title, version, description, contact, license, servers, security schemes, and docs routes.
- input struct tags: path/query/header/cookie/body/file parameters.
- handler return types: successful response schemas.
- router options: tags, shared security requirements, and grouping metadata.
- operation options: summary, description, operationId, extra responses, security requirements, and visibility.

Keep OpenAPI basics in `ninja.Config`; keep endpoint-specific documentation near route registration. Operation-level options are covered in [Routers and Operations](./routers-and-operations.md).

## Route Prefix And Versions

| Field | Effect |
| --- | --- |
| `Prefix` | Global path prefix prepended to every business route |
| `Versions` | Named API version namespaces, descriptions, and deprecation metadata |

Use a global prefix when every business route should live under `/api`:

```go
api := ninja.New(ninja.Config{
    Prefix: "/api",
})
```

Version config works with router `WithVersion`:

```go
api := ninja.New(ninja.Config{
    Prefix: "/api",
    Versions: map[string]ninja.VersionConfig{
        "v1": {Prefix: "/v1", Description: "Stable API"},
        "v2": {Prefix: "/v2", Description: "Next API"},
    },
})

v1 := ninja.NewRouter("/users", ninja.WithVersion("v1"), ninja.WithTags("Users"))
v2 := ninja.NewRouter("/users", ninja.WithVersion("v2"), ninja.WithTags("Users"))

api.AddRouter(v1)
api.AddRouter(v2)
```

The final paths are like `/api/v1/users` and `/api/v2/users`. Versions also get version-scoped OpenAPI documents.

## Homepage And Admin Shortcut

| Field | Effect |
| --- | --- |
| `HomepageURL` | Welcome homepage path, default `/` |
| `DisableHomepage` | Disables the welcome homepage |
| `AdminURL` | Homepage Admin button target; the framework does not mount admin automatically |

```go
api := ninja.New(ninja.Config{
    HomepageURL: "/",
    AdminURL:   "/admin/",
})
```

`AdminURL` is only a homepage shortcut. Mount admin APIs or admin UI explicitly when you need them.

## Security Schemes

`SecuritySchemes` declares reusable OpenAPI security schemes. Router and operation options such as `WithSecurity`, `WithBearerAuth`, `Security`, and `BearerAuth` reference these schemes.

```go
api := ninja.New(ninja.Config{
    SecuritySchemes: map[string]ninja.SecurityScheme{
        "bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
        "apiKeyAuth": ninja.APIKeyHeaderSecurityScheme("X-API-Key"),
    },
})

users := ninja.NewRouter("/users",
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

Use `WithSecurity("bearerAuth")` when you only want OpenAPI metadata. Use `WithBearerAuth(middleware)` or `UseGin` when requests should actually be authenticated.

## Transactions

`TransactionHandlers` powers `ninja.WithTransaction()`. GORM projects usually wire it like this:

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})

api.UseGin(orm.Middleware(db))

ninja.Post(users, "/", createUser,
    ninja.WithTransaction(),
)
```

Without `TransactionHandlers`, an operation using `WithTransaction()` cannot start a request-scoped transaction.

## Error Mapping

`api.RegisterErrorMapper(...)` converts ordinary errors from third-party libraries or service layers into gin-ninja HTTP errors. Returning `nil` means the mapper did not handle that error, so the framework can keep trying later mappers or fall back to the default error handling.

GORM projects can register the built-in mappings:

```go
api := ninja.New(ninja.Config{Title: "Demo API"})
orm.RegisterDefaultErrorMappers(api)
```

This maps `gorm.ErrRecordNotFound` to `404 Not Found` and `gorm.ErrDuplicatedKey` to `409 Conflict`.

Custom business error:

```go
var ErrPlanExpired = errors.New("plan expired")

api.RegisterErrorMapper(func(err error) error {
    if errors.Is(err, ErrPlanExpired) {
        return ninja.NewError(402, "plan expired")
    }
    return nil
})
```

Error mappers are useful for common repository/service errors. For a branch that is obvious inside one handler, returning `ninja.NewError(...)` directly is usually clearer.

## Gin And Middleware

By default, `ninja.New` uses `gin.Default()`, which includes Gin's Logger and Recovery middleware. If you want full control over middleware, set `DisableGinDefault: true` and mount your own stack:

```go
api := ninja.New(ninja.Config{
    DisableGinDefault: true,
})

api.UseGin(
    gin.Logger(),
    gin.Recovery(),
    middleware.RequestID(),
)
```

Call `api.UseGin(...)` before `api.AddRouter(...)` or `api.AddController(...)`. Global middleware affects routers registered after the call. For module-scoped middleware, prefer `router.UseGin(...)`.

Use the raw Gin engine when you need native Gin features:

```go
api.Engine().GET("/healthz", func(c *gin.Context) {
    c.String(200, "ok")
})
```

Use `Handler()` when embedding gin-ninja in another server or a test harness:

```go
server := httptest.NewServer(api.Handler())
defer server.Close()
```

## HTTP Server And Lifecycle

| Field | Effect |
| --- | --- |
| `ReadTimeout` | Timeout for reading the full request |
| `WriteTimeout` | Timeout for writing a response |
| `IdleTimeout` | Timeout for idle keep-alive connections |
| `GracefulShutdownTimeout` | Maximum shutdown wait after `Run` receives `SIGINT` or `SIGTERM` |

Lifecycle hooks:

```go
api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources(ctx)
})
```

`Run(addr)` creates the listener, runs startup hooks, starts accepting requests, and calls `Shutdown(ctx)` after receiving `SIGINT` or `SIGTERM`. If you manage the listener yourself, use:

```go
listener, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatal(err)
}
log.Fatal(api.Serve(listener))
```

## Option Lookup

| Goal | Main options |
| --- | --- |
| Change docs title, version, description | `Title`, `Version`, `Description` |
| Change Swagger/ReDoc paths | `DocsURL`, `OpenAPIURL`, `Docs` |
| Disable production docs | `DisableOpenAPI` or `DisableDocs` |
| Put all routes under `/api` | `Prefix` |
| Configure `/v1` and `/v2` | `Versions` + router `WithVersion` |
| Show an Admin homepage button | `AdminURL` |
| Show Swagger Authorize schemes | `SecuritySchemes` |
| Use `WithTransaction()` | `TransactionHandlers` + `orm.Middleware(db)` |
| Map repository errors to HTTP errors | `api.RegisterErrorMapper(...)` / `orm.RegisterDefaultErrorMappers(api)` |
| Customize the Gin middleware stack | `DisableGinDefault` + `api.UseGin(...)` |
| Tune server timeouts | `ReadTimeout`, `WriteTimeout`, `IdleTimeout` |
| Control graceful shutdown duration | `GracefulShutdownTimeout` |

Previous: [Getting Started](./getting-started.md) | Next: [Request Body and Form Data](./request-body-and-forms.md)
