# gin-ninja

[English](./README.md) | [中文](./README-zh.md)

gin-ninja 是一个基于 [Gin](https://github.com/gin-gonic/gin) 的 Web/API 框架，灵感来自 django-ninja。它在保留 Gin 路由能力和生态的同时，增加了类型安全的处理器、自动 OpenAPI 3.0 文档、生产可用中间件、路由级缓存、API 版本管理，以及与 [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) 的集成。

## 概览

gin-ninja 适合希望继续使用 Gin，但又想要更强结构化 API 开发体验的 Go 项目：

- 使用普通 Go 结构体定义请求输入与响应输出
- 自动生成 OpenAPI 3.0 文档和 Swagger UI
- 通过中间件和操作选项复用认证、日志、安全、限流、超时等横切能力
- 同时支持基础 CRUD、版本化 API、缓存接口、SSE、WebSocket 等场景

典型使用场景：

- RESTful API 服务
- 需要严格请求/响应契约的项目
- 需要自动接口文档的内部平台或开放平台
- 需要 JWT、Session、CSRF、安全头、上传限制等生产能力的应用
- 需要版本隔离、路由缓存或实时推送的服务

## 主要特性

- **类型安全处理器**：基于泛型，直接使用 Go 结构体作为输入输出
- **自动参数绑定**：支持 `path`、`form`、`header`、`cookie`、`json`、`file`
- **默认值与校验**：支持 `default:"..."` 与 `binding:"..."`
- **自动 OpenAPI / Swagger**：默认暴露 `/openapi.json` 与 `/docs`
- **路由分组**：支持嵌套路由、标签、版本、路由级中间件
- **操作级控制**：支持 `Timeout`、`RateLimit`、额外响应声明、隐藏文档等
- **ModelSchema 风格响应**：支持字段白名单/黑名单过滤
- **路由级缓存**：支持 `Cache(...)`、`ETag()`、`CacheControl(...)`、标签失效、内存/Redis 存储
- **版本管理**：支持版本路由、版本文档、弃用与迁移头部
- **流式能力**：支持 SSE 与 WebSocket
- **分页、过滤、排序**：支持 `pagination`、声明式过滤与安全排序
- **文件传输**：支持 multipart 上传与下载响应
- **配置与引导**：内置 settings、bootstrap、logger、ORM 集成
- **内置中间件**：CORS、JWT、i18n、Session、CSRF、安全头、请求日志、Request ID、上传限制、Recovery
- **统一错误模型**：支持协议级错误和业务错误

## 架构与请求流程

运行时的核心流程如下：

1. Gin 接收 HTTP 请求。
2. 引擎级与路由级中间件先执行。
3. gin-ninja 将路径、查询、头、Cookie、JSON、multipart 参数绑定到输入结构体。
4. 处理器以 `*ninja.Context` 和强类型输入结构体执行业务逻辑。
5. 框架统一输出 JSON、下载响应、SSE 或 WebSocket。
6. 路由元数据会被复用于 OpenAPI 文档与 Swagger UI 生成。

核心组件：

- **NinjaAPI**：管理 Gin 引擎、全局中间件、生命周期和文档端点
- **Router**：组织路由前缀、标签、版本和路由级中间件
- **Operation**：包装具体端点的入参绑定、选项控制与响应输出
- **Context**：扩展 `*gin.Context`，增加请求 ID、用户 ID、语言等能力
- **Middleware**：处理鉴权、日志、安全、国际化、上传等横切逻辑

## 包结构

```text
gin-ninja/
├── ninja.go          # NinjaAPI 核心实例
├── router.go         # Router 路由组
├── operation.go      # 类型化处理器包装与操作选项
├── binding.go        # 参数绑定
├── context.go        # 扩展上下文
├── errors.go         # 错误模型与错误写出
├── cache.go          # 路由缓存 / ETag / 缓存失效
├── openapi.go        # OpenAPI 3.0 生成与 Swagger UI
├── schema.go         # JSON Schema 生成
├── stream.go         # SSE 与 WebSocket
├── transfer.go       # 上传与下载抽象
├── versioning.go     # API 版本与弃用头部
│
├── middleware/       # 生产级 HTTP 中间件
├── bootstrap/        # 日志、数据库等初始化辅助
├── filter/           # 声明式过滤
├── order/            # 安全排序
├── orm/              # gormx 集成
├── pagination/       # 分页类型
├── pkg/              # i18n / logger / response 辅助包
├── settings/         # 基于 Viper 的配置加载
└── examples/         # basic 与 full 示例
```

模块职责概览：

| 模块 | 主要职责 |
| --- | --- |
| `NinjaAPI` | 管理应用入口、Gin 引擎、OpenAPI/Swagger、生命周期 |
| `Router` | 按前缀、标签、版本组织端点 |
| `operation.go` | 绑定输入、调用处理器、输出响应、应用操作级选项 |
| `binding.go` | 解析 path/query/header/cookie/json/file 输入 |
| `middleware/` | 提供 JWT、日志、安全、Session、CSRF、i18n 等通用能力 |
| `cache.go` / `versioning.go` / `stream.go` | 提供缓存、版本、SSE、WebSocket 等高级特性 |

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
        DisableGinDefault: true,
    })

    api.UseGin(
        gin.Logger(),
        gin.Recovery(),
        middleware.RequestID(),
        middleware.CORS(nil),
    )

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

启动后可访问：

- Swagger UI：`http://localhost:8080/docs`
- OpenAPI JSON：`http://localhost:8080/openapi.json`

## 核心 API

### NinjaAPI

常用能力：

- `ninja.New(config)`：创建 API 实例
- `api.AddRouter(router)`：注册路由组
- `api.UseGin(mw...)`：注册 Gin 中间件
- `api.Run(addr)`：启动服务并处理优雅关闭
- `api.OnStartup(...)` / `api.OnShutdown(...)`：生命周期钩子
- `api.Shutdown(ctx)`：手动优雅关闭

### Router

常用能力：

- `ninja.NewRouter(prefix, opts...)`
- `router.AddRouter(sub)`：嵌套子路由
- `router.UseGin(...)`：注册路由级 Gin 中间件
- `ninja.WithTags(...)`、`ninja.WithVersion(...)`、`ninja.WithBearerAuth()` 等 RouterOption

### Context

常用辅助方法：

- `ctx.RequestID()`：获取请求 ID
- `ctx.GetUserID()`：读取 JWT 中的用户 ID
- `ctx.Locale()` / `ctx.T(...)`：读取协商语言与翻译消息
- `ctx.JSON200(...)` / `ctx.JSON201(...)` / `ctx.JSON204()`：快捷响应
- `ctx.Forbidden(...)` / `ctx.Unauthorized(...)`：快捷错误返回

## 参数绑定与校验

支持的标签如下：

| 标签 | 来源 | 适用方法 |
| --- | --- | --- |
| `path:"x"` | URL 路径参数 | 全部 |
| `form:"x"` | 查询参数 / 表单字段 | 全部 |
| `header:"x"` | 请求头 | 全部 |
| `cookie:"x"` | Cookie | 全部 |
| `json:"x"` | JSON 请求体 | POST / PUT / PATCH |
| `file:"x"` | Multipart 上传文件 | POST / PUT / PATCH |

补充规则：

- `binding:"..."` 使用 `go-playground/validator`
- `default:"..."` 适用于 `form`、`header`、`cookie`
- multipart 请求中可以把普通 `form` 字段和 `file` 字段写在同一个结构体里

## 响应模型与错误处理

### ModelSchema 风格响应

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

- `fields:"..."` 控制输出字段白名单
- `exclude:"..."` 用于排除敏感字段
- 过滤规则同时作用于 JSON 响应和 OpenAPI schema

### 业务错误与协议错误

`gin-ninja` 区分两类错误：

1. **`*ninja.Error`**：协议级错误，使用对应 HTTP 状态码返回
2. **`*ninja.BusinessError`**：业务级错误，始终返回 HTTP 200，响应体为标准业务信封

```go
return nil, ninja.NewBusinessError(10001, "account is disabled")
return nil, ninja.NewBusinessErrorWithDetail(10002, "quota exceeded", map[string]int{"limit": 100})
```

响应示例：

```json
{"code": 10001, "message": "account is disabled", "data": null}
```

`ValidationError` 会返回 HTTP 422。

## 配置管理（settings）

```go
import "github.com/shijl0925/gin-ninja/settings"

cfg := settings.MustLoad("config.yaml")
// 或
cfg := settings.MustLoadForEnv("config.yaml")
```

示例配置：

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

补充说明：

- 环境变量使用双下划线覆盖配置，例如：`SERVER__PORT=9090`
- `MustLoadWithOverrides` 支持加载基础配置后再叠加覆盖文件
- `MustLoadForEnv` 会根据 `app.env` 自动合并 `config.<env>.yaml`
- MySQL/PostgreSQL 既支持原始 DSN，也支持结构化字段配置

## Bootstrap 与 ORM

```go
import (
    "github.com/shijl0925/gin-ninja/bootstrap"
    "github.com/shijl0925/gin-ninja/orm"
    "github.com/shijl0925/gin-ninja/pkg/logger"
)

cfg := settings.MustLoad("config.yaml")
log := bootstrap.InitLogger(&cfg.Log)
defer logger.Sync()

db := bootstrap.MustInitDB(&cfg.Database)
orm.Init(db)
```

- `bootstrap.MustInitDB` 直接支持 `sqlite`、`mysql`、`postgres`
- `orm.Middleware(db)` 可把数据库句柄注入请求上下文
- 事务场景可以在操作上使用 `ninja.WithTransaction()`

## 中间件

### 引擎级中间件

```go
api.UseGin(
    middleware.RequestID(),
    middleware.Recovery(log),
    middleware.Logger(log),
    middleware.CORS(nil),
    orm.Middleware(db),
)
```

### 路由级中间件

```go
protected := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
protected.UseGin(middleware.JWTAuth())
```

### 常用内置中间件

- **JWT**：`middleware.JWTAuth()`、`middleware.GenerateToken(...)`
- **i18n**：`middleware.I18n()`，支持 `en` / `zh`
- **Session**：HMAC-SHA256 签名 Cookie Session
- **CSRF**：双重提交 Cookie 模式
- **SecureHeaders**：统一设置安全响应头
- **UploadLimit**：限制请求体大小与 MIME 白名单
- **RequestID / Logger / Recovery / CORS**：常见基础中间件

### i18n

注册后会根据 `Accept-Language` 自动协商语言，并自动翻译校验错误。

```go
api.UseGin(middleware.I18n())

func myHandler(ctx *ninja.Context, in *MyInput) (*MyOutput, error) {
    locale := ctx.Locale()
    msg := ctx.T("not_found")
    _ = locale
    _ = msg
    return nil, nil
}
```

### Session / CSRF / 安全头

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret:   "change-me-in-production",
    MaxAge:   86400,
    Secure:   true,
    HTTPOnly: true,
}))

api.UseGin(middleware.CSRF(nil))
api.UseGin(middleware.SecureHeadersStrict())
```

生产建议：

- 使用强随机密钥，不要保留示例密钥
- 全链路启用 HTTPS，并正确传递 `X-Forwarded-Proto`
- Cookie 配置 `Secure`、`HTTPOnly`、合适的 `SameSite`
- 对所有修改类接口启用 CSRF 防护
- 上传接口同时配置大小上限和 MIME 白名单

## 标准响应信封

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

response.Success(c, users)
response.NotFound(c, "user not found")
response.JSON(c, response.OKWithMessage("created", user))
```

成功响应默认格式：

```json
{"code": 0, "message": "success", "data": {...}}
```

## 过滤、排序与分页

### 分页

使用 `pagination.PageInput` 和 `pagination.Page[T]`：

```go
type ListUsersInput struct {
    pagination.PageInput
}
```

### 声明式过滤

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `form:"search" filter:"name|email,like"`
    IsAdmin *bool  `form:"is_admin" filter:"is_admin,eq"`
}
```

支持操作符：`eq`、`ne`、`gt`、`ge`、`lt`、`le`、`like`、`in`。

```go
filterOpts, err := filter.BuildOptions(in)
```

### 安全排序

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `form:"sort" order:"id|name|email|age|created_at"`
}
```

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

白名单之外的排序字段会被拒绝，不会直接传到查询层。

## 文件上传与下载

### 单文件上传

```go
type UploadSingleInput struct {
    Title string              `form:"title" binding:"required"`
    File  *ninja.UploadedFile `file:"file"  binding:"required"`
}
```

### 多文件上传

```go
type UploadManyInput struct {
    Category string                `form:"category" binding:"required"`
    Files    []*ninja.UploadedFile `file:"files"    binding:"required"`
}
```

### 下载响应

```go
func download(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    return ninja.NewDownload(
        "report.txt",
        "text/plain; charset=utf-8",
        []byte("hello from gin-ninja\n"),
    ), nil
}
```

还支持：

- `ninja.NewDownloadReader(...)`
- `Download.Inline = true`
- `Download.Headers`

## OpenAPI 与操作级控制

```go
ninja.Get(users, "/", listUsers,
    ninja.Timeout(2*time.Second),
    ninja.RateLimit(20, 40),
    ninja.PaginatedResponse[UserOut](200, "Paginated users"),
)

ninja.Get(router, "/internal/health", healthz,
    ninja.ExcludeFromDocs(),
)
```

常用操作选项：

- `Summary(...)`
- `Description(...)`
- `Response(...)`
- `Paginated[...]()` / `PaginatedResponse[...]()`
- `ExcludeFromDocs()`
- `Timeout(...)`
- `RateLimit(...)`
- `Security(...)` / `BearerAuth()`
- `Cache(...)` / `CacheControl(...)` / `ETag()`

## 路由缓存 / ETag / Cache-Control

```go
ninja.Get(articles, "/:slug", getArticle,
    ninja.Summary("Get article"),
    ninja.Cache(5*time.Minute),
)
```

行为要点：

- `Cache(ttl)` 默认启用内存缓存
- 成功的 GET/HEAD 响应可自动附带 `ETag`
- 请求携带 `If-None-Match` 时支持 `304 Not Modified`
- 可通过 `CacheWithStore(...)` 切换到 Redis
- 支持 `CacheWithKey(...)`、`CacheWithTags(...)`
- `NewCacheInvalidator(store)` 提供统一失效入口

Redis 示例：

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

## API 版本管理

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
```

对应文档路由：

- `/openapi.json`
- `/docs`
- `/openapi/v1.json`
- `/openapi/v2.json`
- `/docs/v1`
- `/docs/v2`

补充说明：

- `WithVersion("v1")` 用于把 Router 归属到某个 API 版本
- 当版本配置 `Deprecated: true` 时，会输出 `Deprecation` 头
- 配置 `Sunset` / `SunsetTime` 时会输出 `Sunset` 头
- 配置 `MigrationURL` 时会输出 `Link: <...>; rel="deprecation"`
- 版本化 OpenAPI 会自动标记废弃接口

## SSE

```go
type EventsInput struct {
    Topic string `form:"topic" default:"system"`
}

ninja.SSE(events, "/stream", func(ctx *ninja.Context, in *EventsInput, stream *ninja.SSEStream) error {
    return stream.Send(ninja.SSEEvent{
        Event: "message",
        Data:  "hello from gin-ninja",
    })
})
```

默认头部：

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

## WebSocket

```go
type ChatInput struct {
    Room string `form:"room" default:"lobby"`
}

ninja.WebSocket(ws, "/chat", func(ctx *ninja.Context, in *ChatInput, conn *ninja.WebSocketConn) error {
    text, err := conn.ReceiveText()
    if err != nil {
        return err
    }
    return conn.SendText(in.Room + ":" + text)
})
```

常用辅助方法：

- `conn.SendText(...)`
- `conn.ReceiveText()`
- `conn.SendJSON(...)`
- `conn.ReceiveJSON(...)`

## 生命周期钩子

```go
api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources()
})
```

`Run()` 会处理 `SIGINT` / `SIGTERM` 并执行优雅关闭。

## 完整示例

查看 [examples/full](./examples/full/)：

- 基于 `config.yaml` 的配置加载
- 日志与数据库初始化
- JWT 保护的用户 CRUD
- 登录 / 注册接口
- 结构化日志
- 缓存 / ETag / Cache-Control 示例
- 版本化 API 与版本化文档
- SSE / WebSocket 示例
- 单文件、多文件上传
- 二进制下载与流式下载

运行：

```bash
cd examples/full
go run .
```

常用访问地址：

- `http://localhost:8080/docs`
- `http://localhost:8080/docs/v2`
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/openapi/v2.json`

## License

[MIT](./LICENSE)
