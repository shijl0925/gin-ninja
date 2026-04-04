# gin-ninja

A **django-ninja**-inspired web framework built on top of [Gin](https://github.com/gin-gonic/gin) with automatic OpenAPI 3.0 documentation, type-safe request/response handling, production-ready middleware, and first-class [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) ORM integration.

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
- **Pagination** – reusable `PageInput` and `Page[T]` types for consistent list responses.
- **ORM integration** – thin helpers around [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) for repository/service patterns.
- **Built-in middleware** – CORS, JWT auth, structured request logging (Zap), request ID, and panic recovery.
- **Settings** – Viper-based YAML/env configuration management.
- **Logger** – Zap-based structured logger with console/JSON output.
- **Standard response envelope** – `{"code": 0, "message": "success", "data": ...}`.
- **Bootstrap helpers** – one-call database and logger initialization.

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
│   ├── jwt.go        ← JWT auth (golang-jwt/jwt)
│   ├── logger.go     ← structured request logger (Zap)
│   ├── recovery.go   ← panic recovery
│   └── requestid.go  ← X-Request-ID injection
│
├── settings/         ← Viper-based configuration
│   └── settings.go   ← Config, Load, MustLoad
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

Environment variables override file settings using double-underscore separators:
```bash
export SERVER__PORT=9090
export JWT__SECRET=my-secret
```

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

Embed `pagination.PageInput` in a list input struct, then add `filter:"column,op"` to query fields that should become database filters:

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `form:"search"   filter:"name,like"    description:"Filter by name (partial match)"`
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

    if err := filter.Apply(query, in); err != nil {
        return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
    }

    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), query.ToOptions()...)
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
- invalid filter declarations return a 400 error when you surface `filter.Apply(...)` errors

### Safe sorting

`pagination.PageInput.Sort` accepts a comma-separated sort string. Prefix a field with `-` for descending or `+` for ascending:

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

For safety, validate requested sort fields against an allowlist before applying them:

```go
var userSortSchema = pagination.NewSortSchema(
    "id",
    "name",
    "email",
    "age",
    "is_admin",
    "created_at",
)

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    query, _ := gormx.NewQuery[User]()

    if err := pagination.ApplySort(query, in.PageInput, userSortSchema); err != nil {
        return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
    }

    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), query.ToOptions()...)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}
```

If you need a public alias that maps to a different database column, add it explicitly:

```go
schema := pagination.NewSortSchema("name").
    Allow("created", "created_at")
```

Any sort field outside the allowlist is rejected with an error instead of being passed through to the query layer.

### Example

The full example app uses both patterns on `GET /api/v1/users`:

- `search` → `filter:"name,like"`
- `is_admin` → `filter:"is_admin,eq"`
- `sort` → validated against `userSortSchema`

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

## Full Example

See [examples/full](./examples/full/) for a complete application with:
- Settings from `config.yaml`
- Bootstrap (DB + logger initialisation)
- JWT-protected user CRUD endpoints
- Auth login endpoint
- Structured Zap logging
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
