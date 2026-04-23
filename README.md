# gin-ninja

[![DeepSource](https://app.deepsource.com/gh/shijl0925/gin-ninja.svg/?label=active+issues&show_trend=true&token=Z7EU9QDXvlUfgC30hbZQc3dz)](https://app.deepsource.com/gh/shijl0925/gin-ninja/)
[![DeepSource](https://app.deepsource.com/gh/shijl0925/gin-ninja.svg/?label=code+coverage&show_trend=true&token=Z7EU9QDXvlUfgC30hbZQc3dz)](https://app.deepsource.com/gh/shijl0925/gin-ninja/)

[English](./README.md) | [中文](./README-zh.md)

A **django-ninja** inspired web framework built on top of [Gin](https://github.com/gin-gonic/gin) with automatic OpenAPI 3.0 documentation, type-safe request/response handling, production-ready middleware, and first-class [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) ORM integration.

## Overview

gin-ninja is designed for Go teams that want Gin's routing performance with a more structured API layer:

- define handlers with plain Go structs instead of manual binding boilerplate
- generate OpenAPI and Swagger UI automatically from the same route definitions
- keep cross-cutting concerns in reusable middleware and operation options
- scale from small CRUD services to versioned, documented, production-facing APIs

Typical use cases:

- REST APIs with strict request/response contracts
- internal platforms that need fast iteration plus always-up-to-date docs
- services that want built-in auth, security headers, request logging, and config loading
- applications that need versioned APIs, cacheable read endpoints, or realtime SSE / WebSocket routes

## Architecture at a Glance

At runtime, gin-ninja adds a typed API layer on top of Gin:

1. Gin accepts the incoming HTTP request.
2. Engine-level and router-level middleware run first.
3. gin-ninja binds path/query/header/cookie/body/file inputs into typed structs.
4. The typed handler executes with `*ninja.Context`.
5. The framework writes JSON, download, SSE, or WebSocket responses.
6. Route metadata is reused to generate OpenAPI documents and Swagger UI.

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
- **Logger** – Zap-based structured logger with console/JSON output, file sinks, and size-based log rotation.
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
├── cache.go          ← route cache, ETag, cache invalidation helpers
├── openapi.go        ← OpenAPI 3.0 spec generation + Swagger UI
├── schema.go         ← JSON Schema generation
├── stream.go         ← SSE and WebSocket support
├── transfer.go       ← upload/download abstractions
├── versioning.go     ← version-aware docs and deprecation headers
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
├── bootstrap/        ← application bootstrap helpers
│   └── bootstrap.go  ← InitLogger, InitDB, MustInitDB
│
├── filter/           ← declarative query filter builders
├── order/            ← safe sorting helpers
├── orm/              ← gormx integration
│   └── orm.go        ← Init, Middleware, GetDB, WithContext
│
├── pagination/       ← pagination types
│   └── pagination.go ← PageInput, Page[T]
│
└── examples/         ← runnable basic, users, features, admin, and full applications
```

Core module responsibilities:

| Module | Responsibility |
| --- | --- |
| `NinjaAPI` | Owns the Gin engine, global middleware, lifecycle hooks, and OpenAPI/Swagger endpoints |
| `Router` | Groups endpoints by prefix, tags, version, and router-scoped middleware |
| `operation.go` | Wraps typed handlers, binds input, enforces options, and writes typed responses |
| `binding.go` | Maps request data from path/query/header/cookie/json/multipart inputs into structs |
| `middleware/` | Provides production-ready auth, logging, i18n, security, session, and upload middleware |
| `cache.go` / `versioning.go` / `stream.go` | Adds caching, API versioning/deprecation, SSE, and WebSocket capabilities |

---

## Installation

```bash
go get github.com/shijl0925/gin-ninja
```

## Copilot Skill

This repository now includes a workspace Skill at `.github/skills/gin-ninja/`.

- invoke it explicitly with `/gin-ninja`
- or let the agent auto-load it for gin-ninja-specific API, middleware, scaffold, and OpenAPI tasks

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

After startup you can visit:

- `http://localhost:8080/` for the default welcome homepage
- `http://localhost:8080/docs` for the Swagger UI
- `http://localhost:8080/openapi.json` for the raw OpenAPI document

If you want the homepage to include a shortcut to your admin backend, set `AdminURL` in `ninja.Config`.
If you want to keep Swagger UI enabled but hide the homepage shortcut in production, set `HideDocsShortcut: true`.

---

## Lightweight path recommendation

For a small or medium CRUD/internal API, start with the shortest path and add layers only when needed:

- Follow [examples/basic](./examples/basic/): `New + Router + Handler + orm.Middleware`
- Start scaffolding with the default `minimal` template: `gin-ninja-cli startproject mysite -module github.com/acme/mysite`
- Scaffolded repos keep the built-in repo interface layer, while `minimal` still stays the recommended starting point for small CRUD services
- Move to `-template standard|auth|admin` only when you need the extra infrastructure, auth, or admin surface

A good minimal app package usually contains only:

- `app/models.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

---

## Project / App Scaffold Commands

gin-ninja also includes Django-style bootstrap commands for quickly creating a runnable project and new app packages.

The CLI now follows a progressive-help model:

- `gin-ninja-cli --help` shows command groups and recommended entry points
- `gin-ninja-cli help startproject` or `gin-ninja-cli startproject -h` shows full command details
- `gin-ninja-cli init` starts an interactive wizard for new users

The runtime framework stays in the root module, while `cmd/gin-ninja-cli` is maintained as a separate tool module so app builds do not inherit CLI/codegen package boundaries.

Install the CLI into your Go binary directory (`$GOBIN`, or `$GOPATH/bin` when `GOBIN` is unset):

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest

# or install from the cloned repository with Make
make install-cli

# or build only (binary placed at ./bin/gin-ninja-cli)
make build-cli
./bin/gin-ninja-cli --help
```

```bash
gin-ninja-cli --help
gin-ninja-cli help startproject

# small/medium CRUD services: default minimal is usually enough
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
gin-ninja-cli makemigrations
gin-ninja-cli migrate
go run .

# add another app / model package later
gin-ninja-cli startapp blog
gin-ninja-cli makemigrations -app-dir blog -name add-blog-app
gin-ninja-cli migrate

# richer templates / optional features (opt in only when you need them)
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template admin \
  -database postgres \
  -app-dir internal/app \
  -with-tests
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp accounts -template standard -with-gormx -database mysql

# interactive wizard
gin-ninja-cli init

# load a reusable scaffold preset
gin-ninja-cli startproject -config ./scaffold.yaml
gin-ninja-cli startapp -config ./scaffold.yaml
```

`startproject` creates a new directory with:

- `go.mod`
- `main.go`
- `config.yaml`
- `app/models.go`
- `app/migrations.go`
- `app/repos.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

When you opt into `-template standard`, `-template auth`, `-template admin`, or feature flags such as `-with-tests`, the scaffold also adds richer starter files, including:

- `.air.toml`
- `cmd/server/main.go`
- `internal/server/server.go`
- `bootstrap/db.go`
- `bootstrap/logger.go`
- `bootstrap/cache.go`
- `settings/config.local.yaml.example`
- `settings/config.prod.yaml.example`
- `.env.example`
- `Makefile`
- `Dockerfile`
- `docker-compose.yml`
- `README.md`
- `migrations/.gitkeep`
- `scripts/.gitkeep`

`startapp` creates a new app package directory with the same core CRUD files, and richer templates can additionally generate:

- `migrations.go`
- `scaffold_test.go`
- `auth.go`
- `admin.go`
- `permissions.go`

In practice:

- `minimal` keeps the shortest CRUD path
- `standard` mainly adds project-level infrastructure; when `auth/admin` are not enabled it no longer forces `services.go` / `errors.go`
- `auth` / `admin` templates add the fuller service, error, and permission scaffolding

Useful scaffold flags:

- `-template minimal|standard|auth|admin`
- `-with-tests`
- `-with-auth`
- `-with-admin`
- `-database <sqlite|mysql|postgres|none>` (`startproject` defaults to `sqlite`; `startapp` defaults to `none`; selecting a driver wires the matching registration import)
- `-with-gormx` (default `false`; set it to generate gormx-based repos/services instead of native GORM code)
- `-config <path>` (load scaffold values from a YAML/JSON preset; CLI flags override preset values)
- `-app-dir <path>` (`startproject` only)
- `-force`

Example scaffold preset:

```yaml
name: mysite
module: github.com/acme/mysite
output: ./mysite
app_dir: internal/app
database: postgres
template: admin
with_tests: true
with_gormx: false
```

Standard-style project scaffolds also ship with an official [air](https://github.com/air-verse/air) preset for hot reload during development:

```bash
cd mysite
make install-air
make dev
```

The generated code is intended as a starting point and compiles as a minimal CRUD-style template; you can then customize models, validation, middleware, routing, and business logic for your own project.

### Database migrations

The CLI also provides Django-style migration commands driven by an app package that exports:

```go
func MigrationModels() []any
```

Generated scaffolds include this function automatically.

```bash
gin-ninja-cli makemigrations [-config ./config.yaml] [-app-dir app] [-name add_users]
gin-ninja-cli migrate [target|zero]
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

- `makemigrations` captures the SQL emitted by GORM `AutoMigrate` in dry-run mode and writes a timestamped SQL migration under `migrations/`
- `migrate` applies pending migrations, migrates to a target migration, or rolls everything back with `zero`
- `showmigrations` lists all migration files and whether they have been applied
- `sqlmigrate` prints the generated SQL for a migration (`-direction up|down|all`)

---

## CRUD Scaffold Generator

gin-ninja now includes a small scaffolding CLI for generating model-based CRUD boilerplate.

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./examples/full/app/models.go \
  -output ./examples/full/app/user_crud_gen.go
```

The generator:

- reads a Go model struct from the provided file
- creates request/response schemas and CRUD handlers in the same package
- generates a `Register<Model>CRUDRoutes(router)` helper for route registration
- uses `PATCH /:id` for generated partial-update handlers instead of advertising partial updates as `PUT`
- can generate list filter / sort / keyword-search inputs from model `crud:"..."` tags
- can detect same-file belongs-to / has-many / many-to-many relations and generate preload, relation input, and relation output scaffolding

Generated code is intended as a starting point. Review the scaffold and adjust validation, persistence rules, permissions, and router composition for your application.

### CRUD generator tags

Use the `crud:"..."` tag on model fields to opt into generated query inputs:

```go
type Project struct {
    ID      uint   `json:"id"`
    Name    string `json:"name" crud:"filter,sort,search"`
    Status  string `json:"status" crud:"filter:like,sort,search"`
    OwnerID uint   `json:"owner_id" crud:"filter,sort"`
    Owner   User   `gorm:"foreignKey:OwnerID" json:"-"`
    Tasks   []Task `gorm:"foreignKey:ProjectID" json:"-"`
    Tags    []Tag  `gorm:"many2many:project_tags;" json:"-"`
}
```

Supported generator directives:

- `crud:"filter"` → adds a generated list field with `filter:"column,eq"`
- `crud:"filter:like"` → adds a generated list field with `filter:"column,like"`
- `crud:"sort"` → includes the field in generated `Sort string \`order:"..."\``
- `crud:"search"` → includes the field in generated keyword search

The generated list handler wires these into `filter.BuildOptions(...)` and `order.ApplyOrder(...)` automatically.

### Generated relation support

When the generator can resolve related models from the same model file, it now scaffolds relation-aware CRUD output and loading:

- `belongs to` → generates nested relation output plus scalar relation input when needed
- `has many` / `many2many` → generates nested relation output plus `...IDs` input fields
- generated list/detail loads automatically include `Preload(...)`
- generated relation helpers keep association syncing logic out of the handler body

For example, a generated scaffold can now emit:

- nested response fields such as `Owner *ProjectOwnerOut`, `Tasks []ProjectTasksOut`
- relation inputs such as `TagsIDs []uint`
- association helpers such as `syncProjectTagsRelations(...)`

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
  output: "stdout"      # set a file path such as "logs/app.log" to enable file logging
  max_size_mb: 100      # rotate after the file reaches 100 MB
  max_age_days: 7       # keep rotated files for 7 days
  max_backups: 3        # keep up to 3 rotated files
  compress: false       # gzip old rotated files when true
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

### Secrets and environment-variable placeholders

Storing plaintext passwords in `config.yaml` is a security risk, especially in
containerised or cloud deployments where the file may be committed to source control.
gin-ninja supports Spring-style `${VAR}` / `${VAR:default}` placeholders in any string
config value.  After the YAML file is parsed, every token is replaced by the value of
the named environment variable.  If the variable is unset or empty the text after the
first `:` is used as the default; omitting the default causes the field to become an
empty string.

```yaml
database:
  driver: "postgres"
  # Entire DSN from environment; fall back to a local dev connection if unset.
  dsn: "${DATASOURCE_URL:host=localhost user=postgres dbname=myapp sslmode=disable}"

  # Or use structured fields – each credential can come from its own variable.
  postgres:
    host:     "${DB_HOST:localhost}"
    user:     "${DB_USER:postgres}"
    password: "${DB_PASSWORD}"          # no default → empty string when unset

redis:
  password: "${REDIS_PASSWORD}"

jwt:
  secret: "${JWT_SECRET:change-me-in-production}"
```

Multiple placeholders in a single value are supported:

```yaml
database:
  dsn: "${DB_USER:root}:${DB_PASSWORD}@tcp(${DB_HOST:127.0.0.1}:3306)/${DB_NAME:app}"
```

**Precedence (lowest → highest)**

| Source | Example |
|---|---|
| `config.yaml` default value | `password: "fallback"` |
| `${VAR:default}` default | `password: "${DB_PASSWORD:fallback}"` |
| Env var named in placeholder | `DB_PASSWORD=real-pass` |
| Double-underscore env override | `DATABASE__POSTGRES__PASSWORD=top` |

Double-underscore overrides (Viper `AutomaticEnv`) are applied last and therefore
take precedence over placeholders.  Use them when you need to override a key that
does not already contain a placeholder.

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
    _ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
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

`bootstrap.MustInitDB` resolves drivers through registration packages. Import the matching package for the driver you configure, for example:

- `github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres`

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

## Admin Package

The `admin` sub-package provides a metadata-driven back-office API layer plus a built-in single-page admin UI shell that talks to that API.  All three pieces — Site, API routes, and UI pages — are wired up independently so you can use any subset.

### 1. Create a Site

```go
import admin "github.com/shijl0925/gin-ninja/admin"

site := admin.NewSite(
    // optional: enforce auth on every action
    admin.WithPermissionChecker(func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        if ctx.GetUserID() == 0 {
            return ninja.UnauthorizedError()
        }
        return nil
    }),
)
```

`NewSite` accepts zero or more `Option` values.  The only built-in option is `WithPermissionChecker`, which runs before every list / detail / create / update / delete action.

### 2. Register Models with `MustRegisterModel`

Each GORM model gets one `ModelResource` descriptor that controls which fields appear in which views and what operations are allowed.

```go
site.MustRegisterModel(&admin.ModelResource{
    // Model is the GORM model struct (value, not pointer).
    Model: User{},

    // Preloads lists GORM association names to Preload on every query.
    Preloads: []string{"Roles"},

    // Field lists control which fields appear in each view.
    ListFields:   []string{"id", "name", "email", "is_admin", "createdAt"},
    DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt"},
    CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    FilterFields: []string{"is_admin", "age", "createdAt"},
    SortFields:   []string{"id", "name", "email", "age", "createdAt"},
    SearchFields: []string{"name", "email"},

    // Optional per-field display/component overrides.
    FieldOptions: map[string]admin.FieldOptions{
        "is_admin": {Label: "Admin?", Component: "switch"},
    },

    // Optional permission hook called for every action on this resource.
    Permissions: func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        return nil
    },

    // Optional row-level query scope (e.g. multi-tenant filtering).
    RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, res *admin.Resource, db *gorm.DB) *gorm.DB {
        return db.Where("owner_id = ?", ctx.GetUserID())
    }),

    // Optional lifecycle hooks.
    BeforeCreate: func(ctx *ninja.Context, data map[string]any) error { return nil },
    AfterCreate:  func(ctx *ninja.Context, record any) error { return nil },
})
```

`MustRegisterModel` panics on configuration errors (e.g. duplicate resource name).  Use `RegisterModel` instead if you want to handle the error yourself.

Relation fields pointing to another registered model are resolved automatically: the framework infers `value_field`, `label_field`, and `search_fields` from the target resource.

### 3. Mount the Admin API Routes

`site.Mount` registers REST endpoints for every resource under the given `*ninja.Router`.  The router is a standard gin-ninja router, so you can attach JWT middleware or any other gin middleware to it.

```go
adminRouter := ninja.NewRouter(
    "/admin",
    ninja.WithTags("Admin"),
    ninja.WithBearerAuth(),
    ninja.WithVersion("v1"),
)
adminRouter.UseGin(middleware.JWTAuth()) // protect all admin API routes

site.Mount(adminRouter)
api.AddRouter(adminRouter)
```

This registers the following endpoints under `/api/v1/admin` (given `Prefix: "/api"` on `NinjaAPI`):

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/resources` | List all registered resources |
| `GET` | `/resources/{path}/meta` | Resource field metadata |
| `GET` | `/resources/{path}` | Paginated record list (search / filter / sort) |
| `GET` | `/resources/{path}/{id}` | Single record detail |
| `POST` | `/resources/{path}` | Create record |
| `PUT` | `/resources/{path}/{id}` | Update record |
| `DELETE` | `/resources/{path}/{id}` | Delete record |
| `POST` | `/resources/{path}/bulk-delete` | Bulk delete records |
| `GET` | `/resources/{path}/fields/{field}/options` | Relation selector options |

### 4. Mount the Built-in Admin UI Shell

`admin.MountUI` registers the standalone login page, admin workspace, and legacy prototype page as plain HTML routes on any `gin.IRoutes` (including `api.Engine()` for top-level paths outside the API prefix).

```go
// Use all defaults: /admin/login, /admin, /admin-prototype
admin.MountUI(api.Engine(), admin.DefaultUIConfig())

// Or customise paths and title:
admin.MountUI(api.Engine(), admin.UIConfig{
    Title:         "My App Admin",
    APIBasePath:   "/api/v1/admin",
    AuthLoginPath: "/api/v1/auth/login",
    AdminPath:     "/admin",
    LoginPath:     "/admin/login",
    PrototypePath: "/admin-prototype",
})
```

`UIConfig` fields and their defaults:

| Field | Default | Description |
|-------|---------|-------------|
| `Title` | `"Gin Ninja Admin"` | Browser tab title |
| `APIBasePath` | `"/api/v1/admin"` | Admin API root path (for resource navigation) |
| `AuthLoginPath` | `"/api/v1/auth/login"` | Login endpoint called by the sign-in form |
| `AdminPath` | `"/admin"` | Admin workspace page path |
| `LoginPath` | `"/admin/login"` | Standalone login page path |
| `PrototypePath` | `"/admin-prototype"` | Legacy sandbox entry path |
| `TokenExtractExpr` | `"payload.token"` | JS expression to extract the token from the login response |
| `UserNameExtractExpr` | `"payload.name"` | JS expression to extract the display name |
| `UserIDExtractExpr` | `"payload.user_id \|\| payload.userID"` | JS expression to extract the user ID |

#### Customising the token extraction expression

By default the UI reads `payload.token` from the login response.  If your auth endpoint returns the token under a different key (e.g. `{"data": {"accessToken": "..."}}`) set `TokenExtractExpr`:

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:    "/api/v1/user/login",
    // For {"data": {"accessToken": "..."}}
    TokenExtractExpr: "payload.data && payload.data.accessToken",
})
```

The expression is a raw JavaScript expression that receives the parsed `payload` object and should return the token string (or a falsy value on failure).  Similarly, `UserNameExtractExpr` and `UserIDExtractExpr` customise where the display name and user ID are read from:

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:       "/api/v1/user/login",
    TokenExtractExpr:    "payload.data && payload.data.accessToken",
    UserNameExtractExpr: "payload.data && payload.data.userName",
    UserIDExtractExpr:   "payload.data && payload.data.id",
})
```

> **Security note:** the expressions are injected verbatim as JavaScript function bodies.  They must come from trusted, developer-controlled configuration — never from user-supplied input.

---

## Full Example

Split examples are available by feature:

- [examples/users](./examples/users/) — auth register/login plus JWT-protected users CRUD and the cached v2 users API
- [examples/features](./examples/features/) — request metadata, cache / ETag, rate limit, timeout, versioned routing, SSE, WebSocket, upload, and download demos
- [examples/admin](./examples/admin/) — JWT-protected admin resource APIs plus the standalone admin pages
- [examples/full](./examples/full/) — the combined application with every feature above in one app
- [examples/compact](./examples/compact/) — a compact counterpart to `examples/full` that shows the same feature set with fewer local files

The combined [examples/full](./examples/full/) application includes:
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

### Admin console prototype in `examples/full`

The full example also includes a metadata-driven admin experience built on top of the JWT-protected admin resource APIs.

It includes:

- a standalone login page at `/admin/login`
- a standalone admin workspace at `/admin`
- the legacy sandbox entry at `/admin-prototype`
- resource navigation backed by `/api/v1/admin/resources`
- record listing with search, metadata-driven filters, sort, page size, and pagination
- detail, create, update, delete, and bulk delete flows
- relation-backed field selectors with option search previews
- a more compact “Admin Workspace” header for a denser back-office layout

Suggested manual flow:

1. Start the full example:
   ```bash
   cd examples/full
   go run .
   ```
2. Open `http://localhost:8080/admin/login`
3. Sign in with the demo credentials shown on the page
4. After redirecting to `/admin`, pick a resource from the left sidebar
5. Use the workspace to:
   - search and filter the current resource
   - change sort order and page size
   - page through result sets
   - inspect record details
   - create, edit, delete, or bulk delete records
   - preview relation options while filling relation-backed fields

Useful routes:

- `/admin/login` — standalone login shell
- `/admin` — standalone admin workspace
- `/admin-prototype` — legacy prototype entry
- `/api/v1/admin/resources` — admin metadata and CRUD API root

```bash
cd examples/full
go run .
# Open http://localhost:8080/docs
```

---

## License

[MIT](./LICENSE)
