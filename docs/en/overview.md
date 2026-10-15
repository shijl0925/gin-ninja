# Overview

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/overview.md)

gin-ninja is an API framework built on top of Gin. It keeps Gin's routing performance and ecosystem, then adds typed request/response handling, automatic OpenAPI documentation, common middleware, pagination/filtering helpers, configuration loading, scaffolding, and testing tools.

## What It Is For

gin-ninja fits projects where:

- You want to keep using Gin, but do not want to repeat `ShouldBindQuery`, `ShouldBindJSON`, and response schema boilerplate in every handler.
- You want API code and Swagger/OpenAPI documentation to come from the same route definitions.
- You are building CRUD services, admin APIs, internal platforms, or backend APIs that need validation, pagination, filtering, authentication, and test helpers.
- You want small projects to stay light, while larger projects can grow into routers, controllers, app packages, and generated starter code.

For a tiny Gin-only demo, plain Gin is still a fine choice. gin-ninja becomes more useful as endpoints, parameters, and documentation start to repeat.

## Request Flow

A request roughly flows through these steps:

1. Gin receives the HTTP request.
2. Global or router-level middleware runs first, such as logging, CORS, or JWT.
3. gin-ninja binds `path`, `query`, `header`, `cookie`, `json`, `form`, and `file` values into the input struct.
4. `binding:"..."` validation errors are returned automatically.
5. The typed handler runs business logic and returns a Go value.
6. The framework writes JSON, paginated JSON, or a download response.
7. The same route and struct metadata is used to generate `/openapi.json` and `/docs`.

## Module Map

| Module | When you use it |
| --- | --- |
| `ninja.New` / `NinjaAPI` | Create the app, set title/version/prefix/docs, configure lifecycle and global middleware |
| `Router` | Group endpoints by resource, such as `/users` or `/orders` |
| `ninja.Get/Post/Put/Patch/Delete` | Register one typed endpoint |
| `Controller` | Group all endpoints for a resource in a struct and inject DB/service dependencies |
| `binding.go` | Request binding; in daily use you mostly write struct tags |
| `pagination` | Shared `page`/`size` input and paginated responses |
| `filter` / `order` | Declare safe filter and sort fields with tags |
| `middleware` | CORS, JWT, API key, session, CSRF, security headers, logging, recovery, i18n, upload limits |
| `settings` | Load YAML and environment-variable configuration |
| `bootstrap` | Initialize logging and database connections |
| `orm` | Put GORM DB in request context, with transaction helpers |
| `cache.go` | Route-level cache, ETag, and Cache-Control |
| `versioning.go` | API versions and versioned docs |
| `testing` | In-memory API tests with `ninjatest.TestClient` |
| `cmd/gin-ninja-cli` | Project, app, migration, and CRUD scaffolding |
| `admin` | Optional admin API and built-in UI shell |

## Directory Tour

```text
gin-ninja/
├── ninja.go              # NinjaAPI and Config
├── router.go             # Router, Controller, route registration helpers
├── operation.go          # Operation options, response writing, timeout/rate limit/transaction behavior
├── binding.go            # Request binding and validation
├── transfer.go           # UploadedFile and Download
├── openapi.go            # OpenAPI/Swagger/ReDoc
├── cache.go              # Route-level cache
├── versioning.go         # API versions
├── middleware/           # Common HTTP middleware
├── settings/             # Configuration loading
├── bootstrap/            # Logger and DB initialization
├── filter/ order/        # Filtering and sorting
├── pagination/           # Pagination input and output
├── testing/              # TestClient
├── admin/                # Optional Admin
├── cmd/gin-ninja-cli/    # Scaffolding and migration CLI
└── examples/             # Runnable examples
```

## Suggested First Pass

Do not wire every module on day one. A practical learning order is:

1. Build one `NinjaAPI`, one `Router`, and one `GET` endpoint.
2. Add `query`, `path`, and `json` input plus a DTO response.
3. Split a resource like users into its own router or controller.
4. When the project needs a database, add `settings`, `bootstrap`, `orm`, and migrations.
5. Before production, add CORS, auth, security headers, logging, tests, and any needed cache.

Next: [Getting Started](./getting-started.md)
