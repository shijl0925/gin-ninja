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
└── examples/         # basic、users、features、admin 与 full 示例
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

- 首页：`http://localhost:8080/`
- Swagger UI：`http://localhost:8080/docs`
- OpenAPI JSON：`http://localhost:8080/openapi.json`

如果你希望首页展示后台入口按钮，可以在 `ninja.Config` 中设置 `AdminURL`。
如果你希望保留 Swagger UI 路由，但在生产环境隐藏首页里的 API Docs 快捷方式，可以设置 `HideDocsShortcut: true`。

## 项目 / 应用脚手架命令

gin-ninja 也提供了类似 Django 的脚手架命令，可快速创建可运行的项目骨架和新的 app 包。

CLI 会安装到 Go 的可执行目录（优先使用 `$GOBIN`，未设置时使用 `$GOPATH/bin`）：

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest

# 或者通过 Make 安装到当前 Go 的可执行目录
make install-cli

# 或者只在仓库本地构建（产物位于 ./bin/gin-ninja-cli）
make build-cli
./bin/gin-ninja-cli --help
```

```bash
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
gin-ninja-cli makemigrations
gin-ninja-cli migrate
go run .

# 后续新增 app / 模型包
gin-ninja-cli startapp blog
gin-ninja-cli makemigrations -app-dir blog -name add-blog-app
gin-ninja-cli migrate

# 更丰富的模板 / 可选功能
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template admin \
  -app-dir internal/app \
  -with-tests
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp accounts -template standard -with-gormx=false
```

`startproject` 会创建一个新目录，包含：

- `go.mod`
- `main.go`
- `config.yaml`
- `app/models.go`
- `app/migrations.go`
- `app/repos.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

当你启用 `-template standard`、`-template auth`、`-template admin`，或 `-with-tests` 等功能开关时，脚手架还会额外生成更完整的起步文件，例如：

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

`startapp` 会在新的 app package 目录中生成相同的核心 CRUD 文件；更丰富的模板还会额外生成：

- `migrations.go`
- `services.go`
- `errors.go`
- `scaffold_test.go`
- `auth.go`
- `admin.go`
- `permissions.go`

常用脚手架参数：

- `-template minimal|standard|auth|admin`
- `-with-tests`
- `-with-auth`
- `-with-admin`
- `-with-gormx`（默认 `true`；设为 `false` 时生成原生 GORM repo/service，而不是基于 gormx 的代码）
- `-app-dir <path>`（仅 `startproject` 支持）
- `-force`

标准风格项目脚手架还会内置官方 [air](https://github.com/air-verse/air) 预设，方便本地热重载开发：

```bash
cd mysite
make install-air
make dev
```

生成的代码定位为起步骨架，能够作为最小 CRUD 风格模板直接编译；后续你仍可按业务需要继续补充模型、校验、中间件、路由和业务逻辑。

## 数据库迁移命令

CLI 也支持类似 Django 的数据库迁移工作流。对应的 app package 需要导出：

```go
func MigrationModels() []any
```

脚手架生成的 app 已经默认包含该函数。

```bash
gin-ninja-cli makemigrations [-config ./config.yaml] [-app-dir app] [-name add_users]
gin-ninja-cli migrate [target|zero]
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

- `makemigrations` 会通过 GORM `AutoMigrate` 的 dry-run SQL 生成时间戳迁移文件，并写入 `migrations/`
- `migrate` 会应用未执行迁移、迁移到指定版本，或通过 `zero` 回滚全部迁移
- `showmigrations` 会列出所有迁移及其是否已执行
- `sqlmigrate` 会输出指定迁移的 SQL（可通过 `-direction up|down|all` 控制）

## CRUD 脚手架生成器

gin-ninja 现在内置了一个轻量级脚手架 CLI，可基于模型结构体生成 CRUD 接口代码骨架。

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./examples/full/app/models.go \
  -output ./examples/full/app/user_crud_gen.go
```

该生成器会：

- 读取指定文件中的 Go 模型结构体
- 在同一 package 下生成请求/响应结构和 CRUD handler
- 生成 `Register<Model>CRUDRoutes(router)` 路由注册辅助函数
- 对“部分更新”生成 `PATCH /:id` 路由，而不是用 `PUT` 表达部分更新语义
- 可从模型字段的 `crud:"..."` tag 自动生成列表过滤 / 排序 / 关键字搜索输入
- 可识别同一模型文件中的 belongs-to / has-many / many2many 关系，并生成 preload、relation input、relation output 骨架

生成结果定位为“起步骨架”。落地时仍建议根据业务继续补充校验、权限、事务、查询条件和路由组织方式。

### CRUD 生成器 tag 规则

可以在模型字段上声明 `crud:"..."`，控制生成器产出的查询输入：

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

目前支持的生成指令：

- `crud:"filter"`：生成 `filter:"column,eq"` 风格的列表过滤字段
- `crud:"filter:like"`：生成 `filter:"column,like"` 风格的模糊过滤字段
- `crud:"sort"`：把该字段加入生成的 `Sort string \`order:"..."\`` 白名单
- `crud:"search"`：把该字段加入生成的关键字搜索

生成出来的列表 handler 会自动接入：

- `filter.BuildOptions(...)`
- `order.ApplyOrder(...)`

### 生成的关系字段支持

当生成器能在同一个模型文件里解析到关联模型时，会自动补充 relation-aware 的 CRUD 骨架：

- `belongs to`：生成嵌套 relation output，并在需要时生成标量 relation input
- `has many` / `many2many`：生成嵌套 relation output，以及 `...IDs` 形式的 relation input
- 生成的列表 / 详情加载会自动带上 `Preload(...)`
- 自动生成关系同步 helper，减少 handler 中的关联处理样板代码

例如，生成结果现在可以包含：

- `Owner *ProjectOwnerOut`
- `Tasks []ProjectTasksOut`
- `TagsIDs []uint`
- `syncProjectTagsRelations(...)`

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

按功能拆分后的示例：

- [examples/users](./examples/users/)：登录 / 注册、JWT 保护的 users CRUD，以及带缓存失效演示的 v2 users API
- [examples/features](./examples/features/)：请求元数据、缓存 / ETag、限流、超时、版本化路由、SSE、WebSocket、上传、下载等能力演示
- [examples/admin](./examples/admin/)：JWT 保护的 admin 资源 API 与独立 admin 页面
- [examples/full](./examples/full/)：把以上能力组合到一个完整应用中

完整应用可查看 [examples/full](./examples/full/)：

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

### `examples/full` 中的 Admin 控制台原型

完整示例也包含一个基于元数据驱动的 admin 后台体验，它构建在 JWT 保护的 admin 资源 API 之上。

它包括：

- 独立登录页：`/admin/login`
- 独立后台工作台：`/admin`
- 保留旧版沙盒入口：`/admin-prototype`
- 由 `/api/v1/admin/resources` 驱动的资源导航
- 支持搜索、元数据过滤、排序、分页大小和翻页的记录列表
- 详情、创建、更新、删除与批量删除流程
- 带关系字段选项搜索预览的 selector 交互
- 更紧凑的 “Admin Workspace” 头部布局，后台观感更集中

推荐手动体验流程：

1. 启动完整示例：
   ```bash
   cd examples/full
   go run .
   ```
2. 打开 `http://localhost:8080/admin/login`
3. 使用页面展示的演示账号登录
4. 跳转到 `/admin` 后，从左侧选择资源
5. 在工作台中体验：
   - 搜索和过滤当前资源
   - 切换排序与分页大小
   - 浏览分页结果
   - 查看记录详情
   - 创建、编辑、删除或批量删除记录
   - 在关系字段输入时预览候选项

相关路由：

- `/admin/login` — 独立登录页
- `/admin` — 独立后台工作台
- `/admin-prototype` — 旧版原型入口
- `/api/v1/admin/resources` — admin 元数据与 CRUD API 根路径

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
