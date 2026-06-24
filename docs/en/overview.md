# Overview

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/README.md)

## What is gin-ninja?

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
- **Automatic parameter binding** – path params (`path:`), query params (`query:`), headers (`header:`), cookies (`cookie:`), and JSON/form bodies (`json:` / `form:`) are all bound via struct tags.
- **Default parameter values** – `default:"..."` works for query/header/cookie fields and is reflected in OpenAPI.
- **Validation** – powered by [go-playground/validator](https://github.com/go-playground/validator) using the standard `binding:` tag.
- **File transfer abstractions** – first-class multipart upload binding and binary download responses.
- **Auto-generated OpenAPI 3.0 docs** – served as `/openapi.json`.
- **Swagger UI** – available at `/docs` out of the box.
- **Router groups** – nest routers with shared prefixes, OpenAPI tags, and per-router middleware.
- **API Controller** – group all routes for a resource into a struct with dependency injection via `Controller` interface and `api.AddController`.
- **Gin middleware support** – `UseGin()` on both the API and individual routers.
- **OpenAPI controls** – hide internal endpoints from docs and declare extra documented responses per operation.
- **Operation controls** – per-endpoint timeout, in-memory rate limiting, and standard paginated response declarations.
- **ModelSchema-style responses** – use `ResponseModel` / `ResponseSchema` to bind, validate, and prune output fields at runtime while generating matching OpenAPI schemas.
- **Route-level caching** – built-in `Cache(...)`, `ETag()`, `CacheControl(...)`, cache tags, and pluggable memory/Redis stores for read-heavy endpoints.
- **API version isolation** – version-aware routers, per-version OpenAPI/Swagger output, and deprecation headers.
- **Streaming endpoints** – first-class SSE and WebSocket route registration helpers.
- **Pagination** – reusable `PageInput` and `Page[T]` types for consistent list responses.
- **ORM integration** – thin helpers around [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) for repository/service patterns.
- **Built-in middleware** – CORS, JWT auth, structured request logging (Zap), request ID, panic recovery, i18n locale negotiation, **HMAC-signed cookie sessions**, **CSRF protection**, **security response headers**, and **upload size/content-type limits**.
- **Lifecycle hooks** – startup and shutdown hooks with graceful server shutdown.
- **Settings** – Viper-based YAML/env configuration management with **multi-environment override** support.
- **Logger** – Zap-based structured logger with console/JSON output, file sinks, and size-based log rotation.
- **Standard response envelope** – `{"code": 200, "message": "success", "data": ...}`.
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

Next: [Getting Started](./getting-started.md)
