# English Documentation

[Docs Home](../README.md) | [中文](../zh/README.md)

Use this page as the English starting point. The guides are ordered as a learning path first: build a tiny API, learn request and response shapes, split code into modules, then add production concerns.

## Core Pieces

- `NinjaAPI`: the application entry point for the Gin engine, global middleware, OpenAPI, docs, and lifecycle hooks.
- `Router`: a group of operations with a shared prefix, tags, security, and middleware.
- Operation: one HTTP method, path, and typed Go handler, such as `ninja.Get(...)`.
- Input/output structs: plain Go structs that describe request sources, validation, and JSON responses.

Minimal shape:

```go
api := ninja.New(ninja.Config{Title: "Demo API"})

router := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
ninja.Get(router, "/", sayHello, ninja.Summary("Say hello"))
api.AddRouter(router)

api.Run(":8080")
```

## Recommended Reading Path

1. [Overview](./overview.md) — understand what gin-ninja adds on top of Gin.
2. [Getting Started](./getting-started.md) — build your first typed API.
3. [NinjaAPI and Instance Configuration](./api-and-configuration.md) — understand the API instance, global middleware, OpenAPI, server, and `ninja.Config`.
4. [Request Body and Form Data](./request-body-and-forms.md) — learn JSON bodies, URL encoded forms, and multipart forms.
5. [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md) — learn list endpoint pagination, filtering, search, and ordering.
6. [Responses, Errors, and Schemas](./data-and-responses.md) — learn DTOs, paginated responses, error responses, and response schemas.
7. [CRUD Tutorial: Users](./crud-tutorial.md) — build complete users CRUD with native GORM and gormx repositories.
8. [ModelSchema Advanced Usage](./model-schema.md) — optional advanced model-derived schemas, field pruning, input binding, and preloads.
9. [Routers and Operations](./routers-and-operations.md) — look up router configuration, operation registration helpers, and options.
10. [Project and CRUD Scaffolding](./scaffolding.md) — create projects, apps, migrations, and CRUD starter code.
11. [Configuration, Bootstrap, and Lifecycle](./configuration.md) — load settings, initialize logs/databases, and handle graceful shutdown.
12. [Middleware and Security](./middleware-security.md) — add CORS, auth, sessions, CSRF, security headers, logging, i18n, and upload limits.
13. [File Transfer: Uploads and Downloads](./file-transfer.md) — handle file uploads, downloads, and streaming.
14. [Testing APIs with TestClient](./testing.md) — test routers and APIs without opening a real port.
15. [Advanced Features](./advanced-features.md) — add caching, ETag, timeouts, rate limits, transactions, and API versions.
16. [Optional Admin and Full Example](./admin-and-examples.md) — use the admin package and runnable examples.

## Task Index

| Goal | Guide |
| --- | --- |
| Run the first API | [Getting Started](./getting-started.md) |
| Look up `NinjaAPI` and `ninja.Config` options | [NinjaAPI and Instance Configuration](./api-and-configuration.md) |
| Handle JSON bodies, forms, and multipart uploads | [Request Body and Form Data](./request-body-and-forms.md) |
| Build list pagination, filtering, and ordering | [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md) |
| Define JSON output, error responses, and response schemas | [Responses, Errors, and Schemas](./data-and-responses.md) |
| Build users CRUD | [CRUD Tutorial: Users](./crud-tutorial.md) |
| Optionally derive schemas from models | [ModelSchema Advanced Usage](./model-schema.md) |
| Split code into users/orders modules | [Getting Started](./getting-started.md) / [Scaffolding](./scaffolding.md) |
| Look up Router, Get/Post/Put/Patch/Delete, and operation options | [Routers and Operations](./routers-and-operations.md) |
| Create a full project skeleton | [Project and CRUD Scaffolding](./scaffolding.md) |
| Configure database, logging, and env vars | [Configuration, Bootstrap, and Lifecycle](./configuration.md) |
| Add JWT, API keys, CORS, or CSRF | [Middleware and Security](./middleware-security.md) |
| Upload or download files | [File Transfer: Uploads and Downloads](./file-transfer.md) |
| Change Swagger/OpenAPI output | [NinjaAPI and Instance Configuration](./api-and-configuration.md#docs-and-openapi-routes) / [Routers and Operations](./routers-and-operations.md) |
| Write API tests | [Testing APIs with TestClient](./testing.md) |
| Add cache, rate limits, or versioning | [Advanced Features](./advanced-features.md) |
