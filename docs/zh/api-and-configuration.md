# NinjaAPI 与实例配置

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/api-and-configuration.md)

`NinjaAPI` 是 gin-ninja 应用的入口。它包装一个 `*gin.Engine`，负责全局中间件、Router 挂载、Controller 挂载、OpenAPI 生成、文档 UI、生命周期钩子和 HTTP server。

最小形态：

```go
api := ninja.New(ninja.Config{
    Title:   "Demo API",
    Version: "1.0.0",
})

api.AddRouter(usersRouter)
log.Fatal(api.Run(":8080"))
```

`ninja.Config` 是创建 `NinjaAPI` 时的框架实例配置；`settings.Config` 是从 YAML、环境变量等加载出来的应用配置。真实项目里常见做法是先加载 `settings.Config`，再把其中需要影响框架行为的字段填入 `ninja.Config`。

## NinjaAPI 负责什么

| 能力 | 入口 |
| --- | --- |
| 创建 API 实例 | `ninja.New(ninja.Config{...})` |
| 挂全局 Gin 中间件 | `api.UseGin(...)` |
| 挂业务 Router | `api.AddRouter(router)` |
| 挂 Controller | `api.AddController(prefix, controller, opts...)` |
| 访问底层 Gin engine | `api.Engine()` |
| 作为 `http.Handler` 使用 | `api.Handler()` |
| 启动并处理系统信号 | `api.Run(":8080")` |
| 使用自定义 listener | `api.Serve(listener)` |
| 优雅关闭 | `api.Shutdown(ctx)` |
| 注册启动/关闭钩子 | `api.OnStartup(...)`、`api.OnShutdown(...)` |
| 注册错误映射 | `api.RegisterErrorMapper(...)` |

一个完整一点的启动骨架：

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

## 创建时的默认值

`ninja.New` 会给空字段补上安全默认值：

| 字段 | 默认值 |
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

如果设置 `DisableOpenAPI: true`，框架也会禁用 docs UI，因为 Swagger UI/ReDoc 需要读取 OpenAPI JSON。

## OpenAPI 信息选项

这些字段会写入 OpenAPI `info` 或 `servers`：

| 字段 | 作用 |
| --- | --- |
| `Title` | API 标题，显示在 Swagger UI/ReDoc 顶部 |
| `Version` | API 版本字符串 |
| `Description` | API 长说明 |
| `TermsOfService` | 服务条款 URL |
| `Contact` | OpenAPI contact 信息 |
| `LicenseInfo` | OpenAPI license 信息 |
| `Servers` | OpenAPI server 列表 |

示例：

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

## 文档与 OpenAPI 路由

| 字段 | 作用 |
| --- | --- |
| `DocsURL` | 文档 UI 路径，默认 `/docs` |
| `Docs` | 文档 UI 渲染器，默认 `ninja.Swagger()` |
| `DisableDocs` | 禁用文档 UI |
| `HideDocsShortcut` | 只隐藏首页里的 Docs 按钮，不关闭文档路由 |
| `OpenAPIURL` | OpenAPI JSON 路径，默认 `/openapi.json` |
| `DisableOpenAPI` | 禁用 OpenAPI JSON，并同时禁用文档 UI |

把 Swagger UI 改到 `/swagger`：

```go
api := ninja.New(ninja.Config{
    DocsURL:    "/swagger",
    OpenAPIURL: "/schema/openapi.json",
})
```

使用 ReDoc：

```go
api := ninja.New(ninja.Config{
    Docs: ninja.Redoc(),
})
```

生产环境完全关闭文档和 OpenAPI：

```go
api := ninja.New(ninja.Config{
    DisableOpenAPI: true,
})
```

OpenAPI 文档主要由这些信息生成：

- `ninja.Config`：API 标题、版本、描述、联系人、许可证、server、安全方案、文档路径。
- 输入结构体 tag：决定 path/query/header/cookie/body/file 参数。
- handler 返回类型：决定成功响应 schema。
- Router 选项：决定 tags、公共安全要求和分组信息。
- Operation 选项：决定 summary、description、operationId、额外响应、安全要求、是否隐藏。

OpenAPI 基础信息放在 `ninja.Config`；单个接口的文档选项放在路由注册处。Operation 级配置见 [Router 与 Operation](./routers-and-operations.md)。

## 路由前缀与版本

| 字段 | 作用 |
| --- | --- |
| `Prefix` | 全局路径前缀，会加到所有业务路由前面 |
| `Versions` | 声明命名版本的路径、说明和废弃信息 |

全局前缀适合把所有业务接口放在 `/api` 下：

```go
api := ninja.New(ninja.Config{
    Prefix: "/api",
})
```

版本配置和 Router 的 `WithVersion` 配合使用：

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

最终路径类似 `/api/v1/users` 和 `/api/v2/users`。版本还会生成版本范围内的 OpenAPI 文档。

## 首页与 Admin 快捷入口

| 字段 | 作用 |
| --- | --- |
| `HomepageURL` | 欢迎首页路径，默认 `/` |
| `DisableHomepage` | 关闭欢迎首页 |
| `AdminURL` | 首页 Admin 按钮跳转地址；框架不会自动挂载 admin |

```go
api := ninja.New(ninja.Config{
    HomepageURL: "/",
    AdminURL:   "/admin/",
})
```

`AdminURL` 只是首页快捷入口。如果需要 admin API 或 admin UI，仍需要显式挂载 admin 相关 Router/UI。

## 安全方案

`SecuritySchemes` 声明 OpenAPI 里的可复用安全方案。Router 或 operation 上的 `WithSecurity`、`WithBearerAuth`、`Security`、`BearerAuth` 等选项会引用这些方案。

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

只想让 OpenAPI 标记“需要认证”，用 `WithSecurity("bearerAuth")`；需要真正执行认证时，用 `WithBearerAuth(middleware)` 或 `UseGin` 挂认证中间件。

## 事务配置

`TransactionHandlers` 为 `ninja.WithTransaction()` 提供开始、提交、回滚事务的能力。GORM 项目通常这样接：

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})

api.UseGin(orm.Middleware(db))

ninja.Post(users, "/", createUser,
    ninja.WithTransaction(),
)
```

缺少 `TransactionHandlers` 时，使用 `WithTransaction()` 的 operation 无法启动请求级事务。

## 错误映射

`api.RegisterErrorMapper(...)` 可以把第三方库或业务层返回的普通错误转换成 gin-ninja 的 HTTP 错误。mapper 返回 `nil` 表示当前 mapper 不处理这个错误，框架会继续使用后续 mapper 或默认错误处理。

GORM 项目可以直接注册默认映射：

```go
api := ninja.New(ninja.Config{Title: "Demo API"})
orm.RegisterDefaultErrorMappers(api)
```

这会把 `gorm.ErrRecordNotFound` 映射为 `404 Not Found`，把 `gorm.ErrDuplicatedKey` 映射为 `409 Conflict`。

自定义业务错误：

```go
var ErrPlanExpired = errors.New("plan expired")

api.RegisterErrorMapper(func(err error) error {
    if errors.Is(err, ErrPlanExpired) {
        return ninja.NewError(402, "plan expired")
    }
    return nil
})
```

错误映射适合处理 repository/service 层的通用错误。单个 handler 中能明确判断的业务分支，直接返回 `ninja.NewError(...)` 会更直观。

## Gin 与中间件

默认情况下，`ninja.New` 使用 `gin.Default()`，也就是 Gin 自带的 Logger 和 Recovery。若你想完全控制中间件，设置 `DisableGinDefault: true`，然后自己挂：

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

`api.UseGin(...)` 必须在 `api.AddRouter(...)` 或 `api.AddController(...)` 之前调用。全局中间件只影响后续注册的 Router；模块级中间件优先使用 `router.UseGin(...)`。

如果需要使用原生 Gin 能力：

```go
api.Engine().GET("/healthz", func(c *gin.Context) {
    c.String(200, "ok")
})
```

如果要把 gin-ninja 嵌入已有 server 或测试工具：

```go
server := httptest.NewServer(api.Handler())
defer server.Close()
```

## HTTP Server 与生命周期

| 字段 | 作用 |
| --- | --- |
| `ReadTimeout` | 读取完整请求的超时时间 |
| `WriteTimeout` | 写响应的超时时间 |
| `IdleTimeout` | keep-alive 空闲连接超时时间 |
| `GracefulShutdownTimeout` | `Run` 收到 `SIGINT`/`SIGTERM` 后等待关闭的最长时间 |

生命周期钩子：

```go
api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources(ctx)
})
```

`Run(addr)` 会创建 listener、执行 startup hooks、开始接收请求，并在收到 `SIGINT` 或 `SIGTERM` 时调用 `Shutdown(ctx)`。如果你自己管理 listener，可以用：

```go
listener, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatal(err)
}
log.Fatal(api.Serve(listener))
```

## 选项速查

| 场景 | 主要选项 |
| --- | --- |
| 改文档标题、版本、描述 | `Title`、`Version`、`Description` |
| 改 Swagger/ReDoc 地址 | `DocsURL`、`OpenAPIURL`、`Docs` |
| 关闭线上文档 | `DisableOpenAPI` 或 `DisableDocs` |
| 所有接口加 `/api` 前缀 | `Prefix` |
| 配置 `/v1`、`/v2` | `Versions` + Router `WithVersion` |
| 首页显示 Admin 按钮 | `AdminURL` |
| Swagger Authorize 按钮 | `SecuritySchemes` |
| 使用 `WithTransaction()` | `TransactionHandlers` + `orm.Middleware(db)` |
| 映射 repository 错误为 HTTP 错误 | `api.RegisterErrorMapper(...)` / `orm.RegisterDefaultErrorMappers(api)` |
| 自定义 Gin 中间件栈 | `DisableGinDefault` + `api.UseGin(...)` |
| 调整 server 超时 | `ReadTimeout`、`WriteTimeout`、`IdleTimeout` |
| 控制优雅关闭时长 | `GracefulShutdownTimeout` |

上一篇：[快速开始](./getting-started.md) | 下一篇：[Request Body 与 Form Data](./request-body-and-forms.md)
