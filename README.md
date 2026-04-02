# gin-ninja

A **django-ninja**-inspired web framework built on top of [Gin](https://github.com/gin-gonic/gin) with automatic OpenAPI 3.0 documentation, type-safe request/response handling, production-ready middleware, and first-class [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) ORM integration.

## Features

- **Type-safe handlers** – use plain Go structs for request input and response output.
- **Automatic parameter binding** – path params (`path:`), query params (`form:`), request headers (`header:`), and JSON bodies (`json:`) are all bound via struct tags.
- **Validation** – powered by [go-playground/validator](https://github.com/go-playground/validator) using the standard `binding:` tag.
- **Auto-generated OpenAPI 3.0 docs** – served as `/openapi.json`.
- **Swagger UI** – available at `/docs` out of the box.
- **Router groups** – nest routers with shared prefixes, OpenAPI tags, and per-router middleware.
- **Gin middleware support** – `UseGin()` on both the API and individual routers.
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

| Tag          | Source              | Methods            |
|--------------|---------------------|--------------------|
| `path:"x"`   | URL path parameter  | all                |
| `form:"x"`   | URL query string    | all                |
| `header:"x"` | Request header      | all                |
| `json:"x"`   | JSON request body   | POST / PUT / PATCH |

`binding:"..."` uses [go-playground/validator](https://github.com/go-playground/validator).

---

## Full Example

See [examples/full](./examples/full/) for a complete application with:
- Settings from `config.yaml`
- Bootstrap (DB + logger initialisation)
- JWT-protected user CRUD endpoints
- Auth login endpoint
- Structured Zap logging

```bash
cd examples/full
go run .
# Open http://localhost:8080/docs
```

---

## License

[MIT](./LICENSE)

