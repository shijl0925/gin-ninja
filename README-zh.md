# gin-ninja

[English](./README.md) | [中文](./README-zh.md)

gin-ninja 是一个基于 [Gin](https://github.com/gin-gonic/gin) 的 Web/API 框架，灵感来自 django-ninja。它在保留 Gin 路由能力和生态的同时，提供类型安全处理器、自动 OpenAPI 3.0 文档、生产可用中间件、路由级缓存、API 版本管理、流式能力，以及与 [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) 的集成。

## 亮点

- 使用普通 Go 结构体定义请求输入与响应输出
- 自动绑定 path、query、header、cookie、JSON、form、file 参数
- 默认生成 `/openapi.json` 与 `/docs` Swagger UI
- 支持 Router 分组、API Controller、操作级选项、中间件与生命周期钩子
- 内置分页、过滤、排序、缓存、版本管理、SSE、WebSocket、上传下载、settings、bootstrap 与 admin 能力

## 安装

```bash
go get github.com/shijl0925/gin-ninja
```

## 快速开始

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/middleware"
    "github.com/shijl0925/gin-ninja/settings"
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
    api := ninja.New(ninja.Config{Title: "Hello API", Version: "1.0.0", DisableGinDefault: true})
    api.UseGin(gin.Logger(), gin.Recovery(), middleware.RequestID(), middleware.CORSFromConfig(settings.CORSConfig{}))

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

生产环境必须使用 `middleware.CORSFromConfig(...)` 或显式的 `middleware.CORSConfig`。`middleware.CORS(nil)` 仅适合本地开发；它会允许所有来源，并且在 Gin release mode 下会直接 panic。

启动后可访问 `http://localhost:8080/docs` 查看 Swagger UI，访问 `http://localhost:8080/openapi.json` 获取 OpenAPI 文档。

## 文档导航

详细说明已按功能拆分。建议从 [中文文档索引](./docs/zh/README.md) 开始，也可以直接查看：

- [概览](./docs/zh/overview.md)
- [快速开始](./docs/zh/getting-started.md)
- [项目与 CRUD 脚手架](./docs/zh/scaffolding.md)
- [核心 API、绑定与响应](./docs/zh/core-api.md)
- [配置、Bootstrap 与 ORM](./docs/zh/configuration.md)
- [中间件与安全](./docs/zh/middleware-security.md)
- [文件传输与 OpenAPI 控制](./docs/zh/files-and-openapi.md)
- [高级功能](./docs/zh/advanced-features.md)
- [使用 TestClient 测试 API](./docs/zh/testing.md)
- [Admin 与完整示例](./docs/zh/admin-and-examples.md)

英文文档从 [README.md](./README.md) 或 [English documentation index](./docs/en/README.md) 开始。

## 示例

可运行示例位于 [`examples/`](./examples/)，包括 `basic`、`users`、`features`、`admin` 和 `full`。

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
