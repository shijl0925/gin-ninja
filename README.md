# gin-ninja

[![DeepSource](https://app.deepsource.com/gh/shijl0925/gin-ninja.svg/?label=active+issues&show_trend=true&token=Z7EU9QDXvlUfgC30hbZQc3dz)](https://app.deepsource.com/gh/shijl0925/gin-ninja/)

A **django-ninja** inspired web framework built on top of [Gin](https://github.com/gin-gonic/gin) with automatic OpenAPI 3.0 documentation, type-safe request/response handling, production-ready middleware, and first-class [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) ORM integration.

## Features

- **Type-safe handlers** – use plain Go structs for request input and response output.
- **Automatic parameter binding** – path params (`path:`), query params (`form:`), headers (`header:`), cookies (`cookie:`), and JSON bodies (`json:`) are all bound via struct tags.
- **Default parameter values** – `default:"..."` works for query/header/cookie fields and is reflected in OpenAPI.
- **Validation** – powered by [go-playground/validator](https://github.com/go-playground/validator) using the standard `binding:` tag.
- **File transfer abstractions** – first-class multipart upload binding and binary download responses.
- **Auto-generated OpenAPI 3.0 docs** – served as `/openapi.json`.
- **Swagger UI** – available at `/docs` out of the box.
- **Router groups** – nest routers with shared prefixes, OpenAPI tags, and per-router middleware.
- **Gin middleware support** – `UseGin()` on both the API and individual routers.
- **OpenAPI controls** – hide internal endpoints from docs and declare extra documented responses per operation.
- **Operation controls** – per-endpoint timeout, in-memory rate limiting, and standard paginated response declarations.
- **ModelSchema-style responses** – wrap models with `fields` / `exclude` controls for filtered JSON output and OpenAPI schemas.
- **Route-level caching** – built-in `Cache(...)`, `ETag()`, `CacheControl(...)`, cache tags, and pluggable memory/Redis stores for read-heavy endpoints.
- **API version isolation** – version-aware routers, per-version OpenAPI/Swagger output, and deprecation headers.
- **Streaming endpoints** – first-class SSE and WebSocket route registration helpers.
- **Pagination** – reusable `PageInput` and `Page[T]` types for consistent list responses.
- **ORM integration** – thin helpers around [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) for repository/service patterns.
- **Built-in middleware** – CORS, JWT auth, structured request logging (Zap), request ID, panic recovery, i18n locale negotiation, **HMAC-signed cookie sessions**, **CSRF protection**, **security response headers**, and **upload size/content-type limits**.
- **Lifecycle hooks** – startup and shutdown hooks with graceful server shutdown.
- **Settings** – Viper-based YAML/env configuration management with **multi-environment override** support.
- **Logger** – Zap-based structured logger with console/JSON output.
- **Standard response envelope** – `{"code": 0, "message": "success", "data": ...}`.
- **Business errors** – `BusinessError` type with integer codes integrated into the error pipeline.
- **Bootstrap helpers** – one-call database and logger initialization.
- **i18n / L10n** – locale negotiation via `Accept-Language`, translated validation errors and general messages in English and Chinese.
- **API version deprecation** – RFC-compliant `Deprecation` and `Sunset` date headers, migration link.

---

## Package Structure

```
gin-ninja/
├── ninja.go          ← NinjaAPI (core API instance)
├── router.go         ← Router (route groups)
├── operation.go      ← typed handler wrappers
├── binding.go        ← parameter binding (path/query/header/body)
├── context.go        ← Context (extends *gin.Context)
├── errors.go         ← typed error types
├── openapi.go        ← OpenAPI 3.0 spec generation + Swagger UI
├── schema.go         ← JSON Schema generation
│
├── middleware/       ← production-ready HTTP middleware
│   ├── cors.go       ← CORS (gin-contrib/cors)
│   ├── csrf.go       ← CSRF double-submit cookie protection
│   ├── i18n.go       ← locale negotiation (Accept-Language)
│   ├── jwt.go        ← JWT auth (golang-jwt/jwt)
│   ├── logger.go     ← structured request logger (Zap)
│   ├── recovery.go   ← panic recovery
│   ├── requestid.go  ← X-Request-ID injection
│   ├── secure.go     ← security response headers
│   ├── session.go    ← HMAC-signed cookie sessions
│   └── upload.go     ← upload size limit + content-type whitelist
│
├── pkg/
│   ├── i18n/         ← locale negotiation + validation-error translation
│   │   └── i18n.go
│   ├── logger/       ← Zap logger bootstrap
│   └── response/     ← standard JSON response envelope
│
├── settings/         ← Viper-based configuration
│   └── settings.go   ← Config, Load, MustLoad, LoadWithOverrides, LoadForEnv
│
├── pkg/
│   ├── logger/       ← Zap logger setup
│   │   └── logger.go
│   └── response/     ← standard response envelope
│       └── response.go
│
├── bootstrap/        ← application bootstrap helpers
│   └── bootstrap.go  ← InitLogger, InitDB, MustInitDB
│
├── orm/              ← gormx integration
│   └── orm.go        ← Init, Middleware, GetDB, WithContext
│
└── pagination/       ← pagination types
    └── pagination.go ← PageInput, Page[T]
```

---

## Installation

```bash
go get github.com/shijl0925/gin-ninja
```

---

## Quick Start

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/middleware"
)

type HelloInput struct {
    Name string `form:"name" binding:"required"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    api := ninja.New(ninja.Config{
        Title:             "Hello API",
        Version:           "1.0.0",
        DisableGinDefault: true, // use custom middleware instead
    })

    api.UseGin(
        gin.Logger(),                // keep native [GIN] access logs
        gin.Recovery(),              // keep native panic recovery
        middleware.RequestID(),
        middleware.CORS(nil),
    )

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

Visit `http://localhost:8080/docs` for the Swagger UI.

---

## ModelSchema-style Responses

```go
type User struct {
    ID       uint   `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type UserOut struct {
    ninja.ModelSchema[User] `fields:"id,name,email" exclude:"password"`
}

func getUser(ctx *ninja.Context, in *struct{}) (*UserOut, error) {
    return ninja.BindModelSchema[UserOut](User{
        ID:       1,
        Name:     "alice",
        Email:    "alice@example.com",
        Password: "secret",
    })
}
```

`fields:"..."` keeps only the listed serializable fields, while `exclude:"..."` removes sensitive fields from both the JSON response and generated OpenAPI schema.

If you only need ad-hoc filtering without defining a new response type, use `ninja.NewModelSchema(model, ninja.Fields(...), ninja.Exclude(...))`.

---

## Configuration (settings)

```go
import "github.com/shijl0925/gin-ninja/settings"

cfg := settings.MustLoad("config.yaml")
// or
cfg, err := settings.Load("config.yaml")
```

Sample `config.yaml`:

```yaml
app:
  name: "My API"
  version: "1.0.0"
  env: "production"
  debug: false

server:
  host: "0.0.0.0"
  port: 8080

database:
  driver: "sqlite"
  dsn: "app.db"

jwt:
  secret: "change-me-in-production"
  expire_hours: 24

log:
  level: "info"
  format: "json"
  output: "stdout"
```

MySQL / PostgreSQL can use the same `database` block:

```yaml
database:
  # MySQL
  driver: "mysql"
  dsn: "root:p%40ss%3Aword@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"

  # Or use structured fields so special characters in passwords are escaped safely:
  # mysql:
  #   host: "127.0.0.1"
  #   port: 3306
  #   user: "root"
  #   password: "p@ss:word+plus"
  #   name: "gin_ninja"
  #   charset: "utf8mb4"
  #   parse_time: true
  #   loc: "Local"

  # PostgreSQL
  # driver: "postgres"
  # dsn: "host=127.0.0.1 user=postgres password=postgres dbname=gin_ninja port=5432 sslmode=disable TimeZone=Asia/Shanghai"
  # postgres:
  #   host: "127.0.0.1"
  #   port: 5432
  #   user: "postgres"
  #   password: "p@ss word"
  #   name: "gin_ninja"
  #   sslmode: "disable"
  #   time_zone: "Asia/Shanghai"
```

If you still provide a raw MySQL DSN and the password contains reserved characters such as `@`, `:`, `/`, `?`, `#`, or `+`, URL-encode the password segment first. Structured `database.mysql` / `database.postgres` fields avoid that manual escaping step.

Environment variables override file settings using double-underscore separators:
```bash
export SERVER__PORT=9090
export JWT__SECRET=my-secret
```

### Multi-environment config merging

For projects with environment-specific settings, use `LoadWithOverrides` or `LoadForEnv`.

**`LoadWithOverrides`** – loads a base file then merges one or more override files.  Later files
win.  Missing override files are silently skipped, so it is safe to commit the override path even
when the file only exists in certain environments.

```go
// Merges config.local.yaml on top of config.yaml, if it exists.
cfg := settings.MustLoadWithOverrides("config.yaml", "config.local.yaml")
```

**`LoadForEnv`** – automatically discovers and merges the environment-specific override file based
on `app.env` (or the `APP__ENV` environment variable).

```
config.yaml          ← base (always loaded)
config.production.yaml ← merged when app.env=production
config.staging.yaml  ← merged when app.env=staging
config.development.yaml ← merged when app.env=development (default)
```

```go
// Reads app.env from config.yaml, then merges config.<env>.yaml.
cfg := settings.MustLoadForEnv("config.yaml")
```

Only keys present in the override file are changed; all other keys keep their base or default values.

---

## Bootstrap

```go
import (
    "github.com/shijl0925/gin-ninja/bootstrap"
    "github.com/shijl0925/gin-ninja/orm"
    "github.com/shijl0925/gin-ninja/pkg/logger"
)

cfg := settings.MustLoad("config.yaml")

// Initialise Zap logger and set as global.
log := bootstrap.InitLogger(&cfg.Log)
defer logger.Sync()

// Initialise database.
db := bootstrap.MustInitDB(&cfg.Database)
orm.Init(db)
```

`bootstrap.MustInitDB` now supports `sqlite`, `mysql`, and `postgres` directly.

`examples/full/config.yaml` already includes ready-to-copy MySQL and PostgreSQL DSN examples.

### Boundary-case checklist for parser changes

For any code that parses external strings (DSN, headers, query/form values, filter/sort DSL, version params), verify:

- protocol strings are treated as structured input, not generic text
- special characters are covered: `@ : / ? # % + = , ;` and spaces
- empty, malformed, repeated, and mixed-case inputs are tested
- documentation examples have matching tests
- pure parsing helpers have fuzz/property coverage to guard against panics and silent reinterpretation

---

## Middleware

### Engine-level (applies to all routes)

```go
api.UseGin(
    middleware.RequestID(),          // injects X-Request-ID
    middleware.Recovery(log),        // panic recovery with Zap logging
    middleware.Logger(log),          // structured request logging
    middleware.CORS(nil),            // permissive CORS (dev)
    orm.Middleware(db),              // per-request DB in context
)
```

### Router-level (applies only to that group)

```go
protected := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
protected.UseGin(middleware.JWTAuth())  // JWT auth for /admin/* only
```

### JWT Authentication

```go
// Generate a token (e.g. after login):
token, err := middleware.GenerateToken(user.ID, user.Name)

// Protect routes:
r.UseGin(middleware.JWTAuth())

// Read claims in a handler:
claims := middleware.GetClaims(ctx.Context)
fmt.Println(claims.UserID, claims.Username)

```

### I18n – Locale Negotiation and Translated Messages

Register `middleware.I18n()` to automatically negotiate the client locale from the `Accept-Language`
request header.  Supported locales are `"en"` (English) and `"zh"` (Chinese), with `"en"` as the
fallback.

```go
api.UseGin(middleware.I18n())
```

Once registered, **validation-error messages are automatically translated** into the negotiated
locale without any additional code:

```
POST /users  Accept-Language: zh-CN

{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "请求参数校验失败",
    "errors": [
      { "field": "email", "message": "必须是有效的电子邮件地址" }
    ]
  }
}
```

Read the active locale inside a handler:

```go
func myHandler(ctx *ninja.Context, in *MyInput) (*MyOutput, error) {
    locale := ctx.Locale()           // "en" or "zh"
    msg    := ctx.T("not_found")     // "not found" or "资源不存在"
    _ = locale
    _ = msg
    return nil, nil
}
```

Or directly from a raw `*gin.Context` (e.g. inside a custom gin middleware):

```go
locale := middleware.GetLocale(c)
```

The `pkg/i18n` package exposes helpers for translating validation tags and general messages:

```go
import "github.com/shijl0925/gin-ninja/pkg/i18n"

locale := i18n.NegotiateLocale(r.Header.Get("Accept-Language"))
msg    := i18n.TranslateValidation("required", "", locale) // "field is required" / "字段不能为空"
msg2   := i18n.T(locale, "not_found")                     // "not found" / "资源不存在"
```

Available general message keys: `bad_request`, `unauthorized`, `forbidden`, `not_found`,
`conflict`, `internal`, `timeout`, `validation`, `rate_limited`.

### Session / Cookie Authentication

`middleware.SessionMiddleware` provides HMAC-SHA256-signed, cookie-based sessions without external
dependencies.  The session data (a `map[string]string`) is serialised as JSON, signed, and stored
in a single cookie.  Tampered cookies are automatically discarded.

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret:   "change-me-in-production",
    MaxAge:   86400,          // 24 h
    Secure:   true,           // HTTPS only
    HTTPOnly: true,
}))

// In a handler:
session := middleware.GetSession(c)
session.Set("user_id", "42")          // mutations are saved automatically
v, ok := session.Get("user_id")
session.Delete("user_id")

// Generate a fresh session ID (for server-side session stores):
id := middleware.NewSessionID()
```

### CSRF Protection

`middleware.CSRF` implements the **double-submit cookie** pattern.  A random token is set as a
cookie on the first safe request and must be echoed back in the `X-CSRF-Token` header (or
`csrf_token` form field) for all state-changing methods (POST, PUT, PATCH, DELETE).

```go
api.UseGin(middleware.CSRF(nil))   // defaults

// Custom config:
api.UseGin(middleware.CSRF(&middleware.CSRFConfig{
    CookieSecure: true,
    CookieSameSite: http.SameSiteStrictMode,
}))

// Embed the token in forms / single-page apps:
token := middleware.CSRFToken(c)
```

Requests with missing or mismatched tokens are rejected with HTTP 403.

### Security Response Headers

`middleware.SecureHeaders` sets industry-standard security headers in a single call:

```go
// Sensible defaults:
api.UseGin(middleware.SecureHeaders(nil))

// Strict production config (HTTPS):
api.UseGin(middleware.SecureHeadersStrict())

// Custom config:
api.UseGin(middleware.SecureHeaders(&middleware.SecurityConfig{
    ContentTypeNoSniff:    true,
    FrameOption:           "SAMEORIGIN",
    XSSProtection:         true,
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    HSTSMaxAge:            31536000,       // 1 year
    HSTSIncludeSubDomains: true,
    ContentSecurityPolicy: "default-src 'self'",
    PermissionsPolicy:     "geolocation=()",
}))
```

HSTS is only emitted when the request arrives over HTTPS (or the `X-Forwarded-Proto: https`
proxy header is present).

### Upload Size Limit and Content-Type Whitelist

`middleware.UploadLimit` rejects oversized bodies (HTTP 413) and requests with disallowed
content types (HTTP 415) for POST/PUT/PATCH endpoints:

```go
api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
    MaxSize:          5 << 20,   // 5 MiB
    AllowedMIMETypes: []string{
        "application/json",
        "image/",   // prefix: matches image/jpeg, image/png, etc.
    },
}))
```

Pass `nil` to use defaults (10 MiB limit, no content-type checking).

### Security Best Practices

For production deployments, combine the built-in middleware with a few operational safeguards:

- **Use strong secrets**: keep `jwt.secret` and `SessionConfig.Secret` long, random, and environment-specific; never commit placeholder secrets such as `change-me-in-production`.
- **Force HTTPS end-to-end**: enable `Secure` cookies for sessions/CSRF, terminate TLS at the edge, and forward the original scheme so HSTS can be emitted correctly behind proxies.
- **Prefer strict browser protections**: start with `middleware.SecureHeadersStrict()` or explicitly set CSP, Referrer-Policy, `X-Frame-Options`, and HSTS for public deployments.
- **Keep cookies scoped tightly**: use `HTTPOnly`, an appropriate `SameSite` mode, and the narrowest practical `Domain`/`Path` to reduce cross-site exposure.
- **Protect all state-changing routes**: pair cookie-based auth with `middleware.CSRF(...)`, and make sure browser clients echo the CSRF token on every POST/PUT/PATCH/DELETE request.
- **Minimize upload attack surface**: set `UploadLimit` with both a size cap and an explicit MIME allowlist instead of accepting arbitrary request bodies.
- **Harden API docs exposure**: if `/docs` or `/openapi.json` should not be public in production, gate them behind auth, network policy, or disable those routes in your deployment wrapper.
- **Rotate and expire credentials**: keep JWT lifetimes short, rotate signing secrets during incident response, and issue new session IDs after login or privilege changes.

---

## Business Errors

`BusinessError` is a domain-level error that always produces an HTTP 200 response body with a
non-zero integer business code — following the `{"code": <int>, "message": "...", "data": null}`
envelope used by the `pkg/response` package:

```go
// Return a business error from any handler:
return nil, ninja.NewBusinessError(10001, "account is disabled")
return nil, ninja.NewBusinessErrorWithDetail(10002, "quota exceeded", map[string]int{"limit": 100})

// Compare:
if errors.Is(err, ninja.NewBusinessError(10001, "")) { ... }
```

Response:
```json
{"code": 10001, "message": "account is disabled", "data": null}
```

This is distinct from `*ninja.Error` (which sets a non-200 HTTP status code).  Use
`BusinessError` for domain / application-layer failures and `*ninja.Error` for protocol-level
failures (authentication, not found, etc.).

---

## API Version Deprecation Policy

`VersionConfig` now supports richer deprecation metadata:

```go
api := ninja.New(ninja.Config{
    Versions: map[string]ninja.VersionConfig{
        "v1": {
            Deprecated:      true,
            // Optional: emit an HTTP-date in the Deprecation header (RFC 8594):
            DeprecatedSince: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
            // Optional: emit a Sunset header:
            SunsetTime:      time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
            // Or use a pre-formatted string:
            // Sunset: "Tue, 01 Jul 2025 00:00:00 GMT",
            // Optional: emit a Link header pointing to migration docs:
            MigrationURL: "https://example.com/migrate-to-v2",
        },
    },
})
```

Response headers on any deprecated version endpoint:

```
Deprecation: Mon, 01 Jan 2024 00:00:00 GMT
Sunset:      Tue, 01 Jul 2025 00:00:00 GMT
Link:        <https://example.com/migrate-to-v2>; rel="deprecation"
```

When `DeprecatedSince` is zero the `Deprecation` header falls back to the literal `"true"`.

---

## Lifecycle Hooks

```go
api := ninja.New(ninja.Config{
    GracefulShutdownTimeout: 15 * time.Second,
    ReadTimeout:             15 * time.Second,
    WriteTimeout:            30 * time.Second,
    IdleTimeout:             60 * time.Second,
})

api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources()
})

log.Fatal(api.Run(":8080"))
```

`Run()` performs graceful shutdown on `SIGINT` / `SIGTERM` and executes shutdown hooks once.
`Serve(listener)` is available for custom embedding and manual shutdown orchestration.

---

## Standard Response Envelope

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

// Success: {"code": 0, "message": "success", "data": {...}}
response.Success(c, users)

// Error:   {"code": -1, "message": "not found"}
response.NotFound(c, "user not found")

// Custom:  {"code": 0, "message": "created", "data": {...}}
response.JSON(c, response.OKWithMessage("created", user))
```

---

## Parameter Binding

| Tag          | Source                         | Methods            |
|--------------|--------------------------------|--------------------|
| `path:"x"`   | URL path parameter             | all                |
| `form:"x"`   | URL query string / form field  | all                |
| `header:"x"` | Request header                 | all                |
| `cookie:"x"` | Request cookie                 | all                |
| `json:"x"`   | JSON request body              | POST / PUT / PATCH |
| `file:"x"`   | Multipart uploaded file(s)     | POST / PUT / PATCH |

`binding:"..."` uses [go-playground/validator](https://github.com/go-playground/validator).

`default:"..."` applies to `form`, `header`, and `cookie` fields when the client omits the value.

---

## Declarative Filtering & Safe Sorting

### Declarative filtering

Embed `pagination.PageInput` in a list input struct, then add `filter:"column,op"` to query fields that should become database filters. To match one input field against multiple columns, separate the columns with `|`:

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `form:"search"   filter:"name|email,like" description:"Filter by name or email (partial match)"`
    IsAdmin *bool  `form:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
}
```

Supported operators:

- `eq`
- `ne`
- `gt`
- `ge`
- `lt`
- `le`
- `like`
- `in`

Apply the declared filters in the handler:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    query, _ := gormx.NewQuery[User]()

    filterOpts, err := filter.BuildOptions(in)
    if err != nil {
        return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
    }

    opts := append(filterOpts, query.ToOptions()...)
    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}
```

Behavior notes:

- only fields tagged with `filter:"..."` participate in filtering
- zero values are ignored, so omitted query params do not add conditions
- `like` is suitable for contains-style fuzzy matching
- `filter:"name|email,like"` means `(name LIKE ? OR email LIKE ?)`; multi-field declarative filters use OR semantics
- invalid filter declarations return a 400 error when you surface `filter.BuildOptions(...)` or `filter.Apply(...)` errors

### Safe sorting

Use a `sort` query parameter with an `order:"..."` allowlist. Prefix a field with `-` for descending or `+` for ascending:

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

For paginated handlers, keep using `pagination.PageInput` for page/size and declare `Sort` separately:

```go
import "github.com/shijl0925/gin-ninja/order"

type ListUsersInput struct {
    pagination.PageInput
    Sort   string `form:"sort" order:"id|name|email|age|is_admin|created_at"`
    Search string `form:"search" filter:"name|email,like"`
}

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    query, _ := gormx.NewQuery[User]()

    if err := order.ApplyOrder(query, in); err != nil {
        return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
    }

    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), query.ToOptions()...)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}
```

If you need a public alias that maps to a different database column, use `alias:column` or `alias=column`:

```go
type ListUsersInput struct {
    Sort string `form:"sort" order:"name|created:created_at"`
}
```

Any sort field outside the allowlist is rejected with an error instead of being passed through to the query layer.

### Example

The full example app uses declarative sorting on paginated users:

- `GET /api/v1/users` → paginated filtering + sorting
- `sort` → validated by `order:"..."` allowlists before reaching the query layer

Try requests like:

- `/api/v1/users?search=ali`
- `/api/v1/users?is_admin=true&sort=-age`

---

## Multipart File Upload & Download

### Single-file upload

Use `file:"..."` with `*ninja.UploadedFile`:

```go
type UploadSingleInput struct {
    Title string              `form:"title" binding:"required"`
    File  *ninja.UploadedFile `file:"file"  binding:"required"`
}

type UploadDemoOutput struct {
    Title     string   `json:"title,omitempty"`
    Category  string   `json:"category,omitempty"`
    Filename  string   `json:"filename,omitempty"`
    Size      int64    `json:"size,omitempty"`
    FileCount int      `json:"file_count"`
    Names     []string `json:"names,omitempty"`
}

func uploadSingle(ctx *ninja.Context, in *UploadSingleInput) (*UploadDemoOutput, error) {
    return &UploadDemoOutput{
        Title:     in.Title,
        Filename:  in.File.Filename,
        Size:      in.File.Size,
        FileCount: 1,
    }, nil
}

ninja.Post(router, "/upload-single", uploadSingle,
    ninja.Summary("Single file upload"),
    ninja.Description("Demonstrates multipart form-data binding with one file and extra form fields."),
)
```

`UploadedFile` wraps `multipart.FileHeader` and exposes:

- `in.File.Filename`
- `in.File.Size`
- `in.File.Open()`
- `in.File.Bytes()`

### Multi-file upload

Use `[]*ninja.UploadedFile` for repeated multipart fields:

```go
type UploadManyInput struct {
    Category string                `form:"category" binding:"required"`
    Files    []*ninja.UploadedFile `file:"files"    binding:"required"`
}

func uploadMany(ctx *ninja.Context, in *UploadManyInput) (*UploadDemoOutput, error) {
    names := make([]string, 0, len(in.Files))
    for _, file := range in.Files {
        names = append(names, file.Filename)
    }
    return &UploadDemoOutput{
        Category:  in.Category,
        FileCount: len(in.Files),
        Names:     names,
    }, nil
}
```

### Mixed form + file binding

`form:"..."` and `file:"..."` can be mixed in the same input struct. When the request uses `multipart/form-data`, gin-ninja binds regular form fields and uploaded files together and generates the matching OpenAPI request body automatically.

### File download responses

Return `*ninja.Download` when the handler should write a binary response instead of JSON:

```go
func download(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    return ninja.NewDownload(
        "report.txt",
        "text/plain; charset=utf-8",
        []byte("hello from gin-ninja\n"),
    ), nil
}

func downloadReader(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    body := strings.NewReader("streamed content\n")
    return ninja.NewDownloadReader(
        "stream.txt",
        "text/plain; charset=utf-8",
        int64(body.Len()),
        body,
    ), nil
}
```

Available helpers:

- `ninja.NewDownload(filename, contentType, data)` – byte-slice backed download
- `ninja.NewDownloadReader(filename, contentType, size, reader)` – reader-backed download
- `Download.Inline = true` – switch `Content-Disposition` from `attachment` to `inline`
- `Download.Headers` – add custom response headers

OpenAPI will describe upload inputs as `multipart/form-data`, and `*ninja.Download` responses as binary `application/octet-stream`.

### Example routes

The full example app includes ready-to-run routes:

- `POST /api/v1/examples/upload-single`
- `POST /api/v1/examples/upload-many`
- `GET /api/v1/examples/download`
- `GET /api/v1/examples/download-reader`

---

## OpenAPI Operation Controls

```go
users := ninja.NewRouter(
    "/users",
    ninja.WithTags("Users"),
    ninja.WithTagDescription("Users", "User management endpoints"),
)

type SessionInput struct {
    Session string `cookie:"session" binding:"required" default:"guest"`
}

type SessionOutput struct {
    Session string `json:"session"`
}

ninja.Get(router, "/session", getSession,
    ninja.Response(401, "Unauthorized", nil),
    ninja.Response(404, "Session not found", &SessionOutput{}),
)

ninja.Get(router, "/internal/health", healthz,
    ninja.ExcludeFromDocs(),
)

ninja.Get(users, "/", listUsers,
    ninja.Timeout(2*time.Second),
    ninja.RateLimit(20, 40),
    ninja.PaginatedResponse[UserOut](200, "Paginated users"),
)
```

Use `Response(...)` / `PaginatedResponse[...]` to document non-default OpenAPI responses, `ExcludeFromDocs()` for internal endpoints, `Timeout(...)` for context-based per-operation deadlines, and `RateLimit(...)` for per-operation throttling.

---

## Route-Level Cache / ETag / Cache-Control

For read-only endpoints, you can enable built-in response caching and conditional requests:

```go
type ArticleInput struct {
    Slug string `path:"slug" binding:"required"`
}

type ArticleOutput struct {
    Slug    string `json:"slug"`
    Title   string `json:"title"`
    Content string `json:"content"`
}

func getArticle(ctx *ninja.Context, in *ArticleInput) (*ArticleOutput, error) {
    return &ArticleOutput{
        Slug:    in.Slug,
        Title:   "gin-ninja cache demo",
        Content: "This response can be cached",
    }, nil
}

articles := ninja.NewRouter("/articles", ninja.WithTags("Articles"))

ninja.Get(articles, "/:slug", getArticle,
    ninja.Summary("Get article"),
    ninja.Cache(5*time.Minute),
)
```

Behavior:

- `Cache(ttl)` enables route caching with the default in-memory backend
- successful GET/HEAD responses automatically include `ETag`
- when `CacheControl(...)` is not set explicitly, `Cache(ttl)` emits `Cache-Control: public, max-age=<ttl>`
- requests with `If-None-Match` return `304 Not Modified` when the cached entity tag matches
- the same API can target Redis by passing `CacheWithStore(...)`

Useful options:

```go
store := ninja.NewMemoryCacheStore()

ninja.Get(articles, "/:slug", getArticle,
    ninja.Cache(5*time.Minute,
        ninja.CacheWithStore(store),
        ninja.CacheWithKey(func(ctx *ninja.Context) string {
            return "article:" + ctx.Param("slug")
        }),
        ninja.CacheWithTags(func(ctx *ninja.Context) []string {
            return []string{"articles", "article:" + ctx.Param("slug")}
        }),
    ),
    ninja.CacheControl("public, max-age=300, stale-while-revalidate=60"),
    ninja.ETag(),
)
```

Redis-backed store:

```go
store, err := ninja.NewRedisCacheStore(ninja.RedisCacheConfig{
    Addr:   "127.0.0.1:6379",
    Prefix: "myapp:",
})
if err != nil {
    panic(err)
}

invalidator := ninja.NewCacheInvalidator(store)
invalidator.InvalidateTags("article:welcome")
```

Notes:

- cache support is intended for safe read endpoints
- SSE / WebSocket routes are not cached
- `NewCacheInvalidator(store)` provides a unified delete / tag-invalidation / lock entry point
- OpenAPI automatically documents `ETag` and `Cache-Control` response headers

---

## API Version Management

gin-ninja now supports version-aware routing in addition to a global prefix.

```go
api := ninja.New(ninja.Config{
    Title:   "Example API",
    Version: "main",
    Prefix:  "/api",
    Versions: map[string]ninja.VersionConfig{
        "v1": {
            Prefix:       "/v1",
            Description:  "Legacy API",
            Deprecated:   true,
            Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
            MigrationURL: "https://example.com/migrate-to-v2",
        },
        "v2": {
            Prefix:      "/v2",
            Description: "Current stable API",
        },
    },
})

v1Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v1"))
v2Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v2"))

ninja.Get(v1Users, "/", listUsersV1, ninja.Summary("List users (v1)"))
ninja.Get(v2Users, "/", listUsersV2, ninja.Summary("List users (v2)"))

api.AddRouter(v1Users)
api.AddRouter(v2Users)
```

This registers:

- `GET /api/v1/users`
- `GET /api/v2/users`
- `GET /openapi/v1.json`
- `GET /openapi/v2.json`
- `GET /docs/v1`
- `GET /docs/v2`

Deprecation behavior:

- when a version is marked `Deprecated: true`, responses include `Deprecation: true`
- `Sunset` is emitted when configured
- `Link: <...>; rel="deprecation"` is emitted when `MigrationURL` is configured
- versioned OpenAPI output marks operations in deprecated versions as `deprecated: true`

Recommended pattern:

- keep `Config.Prefix` for a shared top-level namespace such as `/api`
- use `WithVersion("v1")`, `WithVersion("v2")` on routers that belong to a specific API generation
- use separate handlers/schema types when versions diverge semantically

---

## SSE (Server-Sent Events)

Use `ninja.SSE(...)` for one-way server push / streaming text output:

```go
type EventsInput struct {
    Topic string `form:"topic" default:"system"`
}

events := ninja.NewRouter("/events", ninja.WithTags("Events"))

ninja.SSE(events, "/stream", func(ctx *ninja.Context, in *EventsInput, stream *ninja.SSEStream) error {
    if err := stream.Send(ninja.SSEEvent{
        Event: "ready",
        Data: map[string]string{
            "topic": in.Topic,
            "status": "connected",
        },
    }); err != nil {
        return err
    }

    return stream.Send(ninja.SSEEvent{
        Event: "message",
        Data:  "hello from gin-ninja",
    })
})
```

Default response headers:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

You can send:

- plain strings
- byte slices
- structs / maps (encoded as JSON)
- `ID`, `Event`, and `Retry` metadata via `ninja.SSEEvent`

Example client:

```js
const source = new EventSource("/events/stream?topic=system");
source.addEventListener("message", (event) => {
  console.log(event.data);
});
```

---

## WebSocket

Use `ninja.WebSocket(...)` for bidirectional realtime communication:

```go
type ChatInput struct {
    Room string `form:"room" default:"lobby"`
}

ws := ninja.NewRouter("/ws", ninja.WithTags("Realtime"))

ninja.WebSocket(ws, "/chat", func(ctx *ninja.Context, in *ChatInput, conn *ninja.WebSocketConn) error {
    text, err := conn.ReceiveText()
    if err != nil {
        return err
    }
    return conn.SendText(in.Room + ":" + text)
})
```

Convenience helpers:

- `conn.SendText(...)`
- `conn.ReceiveText()`
- `conn.SendJSON(...)`
- `conn.ReceiveJSON(...)`

Example client:

```js
const ws = new WebSocket("ws://localhost:8080/ws/chat?room=lobby");
ws.onopen = () => ws.send("ping");
ws.onmessage = (event) => console.log(event.data);
```

OpenAPI documents the route as a `101 Switching Protocols` response so the upgrade is visible in generated docs.

---

## Full Example

See [examples/full](./examples/full/) for a complete application with:
- Settings from `config.yaml`
- Bootstrap (DB + logger initialisation)
- JWT-protected user CRUD endpoints
- Auth register/login endpoints
- Structured Zap logging
- Route-level cache / ETag / Cache-Control demos
- Versioned API routing and per-version docs demos
- SSE / WebSocket demos
- Multipart single-file and multi-file upload demos
- Binary download and reader-backed download demos

```bash
cd examples/full
go run .
# Open http://localhost:8080/docs
```

---

## License

[MIT](./LICENSE)
