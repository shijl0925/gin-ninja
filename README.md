# gin-ninja

[![DeepSource](https://app.deepsource.com/gh/shijl0925/gin-ninja.svg/?label=active+issues&show_trend=true&token=Z7EU9QDXvlUfgC30hbZQc3dz)](https://app.deepsource.com/gh/shijl0925/gin-ninja/)
[![DeepSource](https://app.deepsource.com/gh/shijl0925/gin-ninja.svg/?label=code+coverage&show_trend=true&token=Z7EU9QDXvlUfgC30hbZQc3dz)](https://app.deepsource.com/gh/shijl0925/gin-ninja/)

[English](./README.md) | [中文](./README-zh.md)

A **django-ninja** inspired lightweight web/API framework built on top of [Gin](https://github.com/gin-gonic/gin). The core module focuses on type-safe request/response handling, automatic Binding, OpenAPI 3.0 documentation, routers, route caching, API versioning, streaming helpers, and file transfer. Admin, ORM, settings/bootstrap, middleware, and Redis cache integrations are optional modules.

## Highlights

- Type-safe handlers with plain Go request and response structs
- Automatic binding for path, query, header, cookie, JSON, form, and file inputs
- Generated OpenAPI JSON and Swagger UI at `/openapi.json` and `/docs`
- Router groups, API controllers, operation options, middleware, and lifecycle hooks
- Built-in support for pagination, caching, versioning, SSE, WebSocket, and upload/download
- Optional modules for `admin`, `orm`, `settings`, `bootstrap`, `middleware`, `filter`, `order`, and `cache/redis`

## Installation

```bash
go get github.com/shijl0925/gin-ninja
```

Optional extensions can be added independently:

```bash
go get github.com/shijl0925/gin-ninja/admin
go get github.com/shijl0925/gin-ninja/orm
go get github.com/shijl0925/gin-ninja/bootstrap
go get github.com/shijl0925/gin-ninja/middleware
```

## Quick Start

```go
package main

import (
    "log"
    ninja "github.com/shijl0925/gin-ninja"
)

type HelloInput struct {
    Name string `query:"name" binding:"required"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    api := ninja.New(ninja.Config{Title: "Hello API", Version: "1.0.0"})

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

Use `middleware.CORSFromConfig(...)` or an explicit `middleware.CORSConfig` in production. `middleware.CORS(nil)` is development-only; it allows all origins and panics in Gin release mode.

Visit `http://localhost:8080/docs` for Swagger UI and `http://localhost:8080/openapi.json` for the generated OpenAPI document.

## Documentation

Detailed documentation has been split by feature area. Start with the [English documentation index](./docs/en/README.md), or jump to a guide:

- [Overview](./docs/en/overview.md)
- [Getting Started](./docs/en/getting-started.md)
- [Project and CRUD Scaffolding](./docs/en/scaffolding.md)
- [Configuration, Bootstrap, and Lifecycle](./docs/en/configuration.md)
- [Middleware and Security](./docs/en/middleware-security.md)
- [Data, Binding, and Responses](./docs/en/data-and-responses.md)
- [File Transfer and OpenAPI Controls](./docs/en/files-and-openapi.md)
- [Advanced Features](./docs/en/advanced-features.md)
- [Testing APIs with TestClient](./docs/en/testing.md)
- [Admin and Full Example](./docs/en/admin-and-examples.md)

Chinese documentation starts at [README-zh.md](./README-zh.md) or the [中文文档索引](./docs/zh/README.md).

## Examples

Runnable examples live under the separate [`examples/`](./examples/) module, including `basic`, `users`, `features`, `admin`, and `full`.

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
