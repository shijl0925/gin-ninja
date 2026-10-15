# Router 与 Operation

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/routers-and-operations.md)

`Router` 负责把一组接口放到同一个 URL 前缀下，并共享标签、安全要求、中间件和版本信息。`Operation` 是具体的一个接口，由 HTTP 方法、路径、输入结构体、处理函数、响应类型和一组选项组成。

日常写业务模块时，可以先记住这个结构：

```go
r := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(r, "/", listUsers, ninja.Summary("List users"), ninja.Paginated[UserOut]())
ninja.Post(r, "/", createUser, ninja.Summary("Create user"), ninja.WithTransaction())

api.AddRouter(r)
```

Router 级选项适合放“整个模块都一样”的配置，例如 tag、认证和版本。Operation 级选项适合放“单个接口不同”的配置，例如 summary、响应模型、分页、超时、限流和事务。

## 创建 Router

最小写法：

```go
r := ninja.NewRouter("/users")
api.AddRouter(r)
```

带 OpenAPI 分组：

```go
r := ninja.NewRouter(
    "/users",
    ninja.WithTags("Users"),
    ninja.WithTagDescription("Users", "User management endpoints"),
)
```

`api.AddRouter(r)` 会把 Router 下已经注册的操作挂到 Gin engine，并写入 OpenAPI。Router 挂载后不能再继续添加 operation、中间件或子 Router，否则会 panic。这能避免服务已经启动后路由表被悄悄改变。

如果使用 Controller，`api.AddController(prefix, controller, opts...)` 内部也是先创建 Router，再调用 controller 的 `Register` 方法，最后挂载到 API。因此 `WithTags`、`WithBearerAuth`、`WithVersion` 等 Router 选项同样适用于 Controller：

```go
api.AddController("/users", &UserController{db: db},
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

## Router 选项

| 选项 | 作用 | 常见场景 |
| --- | --- | --- |
| `ninja.WithTags(tags...)` | 给 Router 下的操作设置默认 OpenAPI tags | Swagger UI 按 `Users`、`Orders` 分组 |
| `ninja.WithTagDescription(tag, description)` | 给一个 OpenAPI tag 增加说明 | 给资源分组补充描述 |
| `ninja.WithTagDescriptions(map[string]string)` | 一次设置多个 tag 说明 | 一个 Router 涉及多个 OpenAPI tag |
| `ninja.WithSecurity(name, scopes...)` | 只声明 OpenAPI security requirement | 文档标记接口需要认证，但认证中间件另行挂载 |
| `ninja.WithSecurityMiddleware(name, mw, scopes...)` | 声明 security，并把 Gin middleware 挂到 Router | 自定义认证方案 |
| `ninja.WithBearerAuth(mw...)` | 使用默认 `bearerAuth` security，并挂载认证中间件 | JWT Bearer 认证 |
| `ninja.WithBasicAuth(mw...)` | 使用默认 `basicAuth` security，并挂载认证中间件 | Basic Auth |
| `ninja.WithAPIKeyAuth(name, mw...)` | 使用指定 API Key security，并挂载认证中间件 | `X-API-Key`、网关 key |
| `ninja.WithOAuth2Auth(scopes...)` | 只声明默认 `oauth2` security | OAuth2 文档标记 |
| `ninja.WithOAuth2AuthMiddleware(mw, scopes...)` | 声明默认 `oauth2` security，并挂载认证中间件 | OAuth2 scope 校验 |
| `ninja.WithVersion(version)` | 把 Router 标记到一个 API 版本 | `/v1/users`、`/v2/users` |

`WithBearerAuth`、`WithBasicAuth`、`WithAPIKeyAuth` 这些带 `Auth` 的 helper 需要传入真实的 Gin middleware。只想生成 OpenAPI 安全标记时，用 `WithSecurity(...)`。

安全方案本身需要在 `ninja.Config.SecuritySchemes` 中声明，Swagger UI 才知道 `bearerAuth`、`basicAuth`、`oauth2` 或自定义 API Key 的展示方式。

## Router 中间件

Router 可以挂两类中间件。

`Use` 接收 gin-ninja 的类型化上下文：

```go
r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))

r.Use(func(ctx *ninja.Context) error {
    if ctx.GetHeader("X-Admin") != "true" {
        return ninja.NewError(403, "admin only")
    }
    return nil
})
```

`UseGin` 接收原生 `gin.HandlerFunc`，适合复用已有 Gin 中间件：

```go
r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
r.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))
```

执行顺序上，Router 级 Gin middleware 先进入，然后是 Router 级类型化 middleware，最后才到具体 operation 的 handler。Operation 级认证中间件只作用于当前接口。

## 嵌套 Router

`AddRouter` 可以把子 Router 挂到父 Router 下：

```go
apiRouter := ninja.NewRouter("/api")
v1 := ninja.NewRouter("/v1")
users := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(users, "/:id", getUser, ninja.Summary("Get user"))

v1.AddRouter(users)
apiRouter.AddRouter(v1)
api.AddRouter(apiRouter)
```

最终路径是 `/api/v1/users/:id`。子 Router 会继承父 Router 的路径层级；安全、tag 和版本等默认值应尽量写在最贴近业务模块的位置，避免嵌套太深后不容易看清来源。

## 注册 Operation

gin-ninja 使用 Go 泛型函数注册 operation，而不是 Python 风格 decorator。

| 函数 | HTTP 方法 | Handler 签名 | 默认成功状态码 |
| --- | --- | --- | --- |
| `ninja.Get[TIn, TOut](r, path, handler, opts...)` | `GET` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Post[TIn, TOut](r, path, handler, opts...)` | `POST` | `func(*ninja.Context, *TIn) (*TOut, error)` | `201 Created` |
| `ninja.Put[TIn, TOut](r, path, handler, opts...)` | `PUT` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Patch[TIn, TOut](r, path, handler, opts...)` | `PATCH` | `func(*ninja.Context, *TIn) (*TOut, error)` | `200 OK` |
| `ninja.Delete[TIn](r, path, handler, opts...)` | `DELETE` | `func(*ninja.Context, *TIn) error` | `204 No Content` |

`GET` operation 会同时注册对应的 `HEAD` 路由。`DELETE` 默认没有响应体，所以 handler 只返回 `error`。

```go
type GetUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    user, err := repo.Get(ctx.Request.Context(), in.UserID)
    if err != nil {
        return nil, err
    }
    return toUserOut(user), nil
}

ninja.Get(r, "/:id", getUser,
    ninja.Summary("Get user"),
    ninja.Response(404, "User not found", &ErrorOut{}),
)
```

## Operation 文档选项

| 选项 | 作用 |
| --- | --- |
| `ninja.Summary(text)` | 设置 OpenAPI summary |
| `ninja.Description(text)` | 设置 OpenAPI description |
| `ninja.OperationID(id)` | 设置稳定的 OpenAPI `operationId` |
| `ninja.Tags(tags...)` | 覆盖当前 operation 的 tags |
| `ninja.TagDescription(tag, description)` | 为当前 operation 使用的 tag 补充说明 |
| `ninja.Deprecated()` | 标记接口已废弃 |
| `ninja.ExcludeFromDocs()` | 从 OpenAPI 文档中隐藏接口 |

Router 的 `WithTags(...)` 是默认值；operation 上的 `Tags(...)` 会覆盖它：

```go
r := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(r, "/internal/health", healthz,
    ninja.Tags("Internal"),
    ninja.ExcludeFromDocs(),
)
```

## Operation 响应选项

| 选项 | 作用 |
| --- | --- |
| `ninja.SuccessStatus(code)` | 修改成功响应状态码 |
| `ninja.Response(status, description, model)` | 为 OpenAPI 增加额外响应 |
| `ninja.ResponseModel[T]()` | 用模型 schema 约束成功响应，运行时也会按 schema 输出 |
| `ninja.ResponseSchema(descriptor)` | 使用可复用的 model schema descriptor |
| `ninja.Paginated[T]()` | 声明标准分页成功响应 |
| `ninja.PaginatedSchema(descriptor)` | 声明带 schema descriptor 的分页成功响应 |
| `ninja.CursorPaginated[T]()` | 声明游标分页成功响应 |
| `ninja.CursorPaginatedSchema(descriptor)` | 声明带 schema descriptor 的游标分页成功响应 |
| `ninja.PaginatedResponse[T](status, description)` | 为 OpenAPI 增加额外的分页响应 |
| `ninja.CursorPaginatedResponse[T](status, description)` | 为 OpenAPI 增加额外的游标分页响应 |

列表接口通常返回 `*pagination.Page[T]`，并配合 `Paginated[T]()`：

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    return service.ListUsers(ctx.Request.Context(), in)
}

ninja.Get(r, "/", listUsers,
    ninja.Summary("List users"),
    ninja.Paginated[UserOut](),
)
```

当 handler 返回的是 GORM model，但公开响应需要隐藏字段时，用 `ResponseModel` 或 `ResponseSchema`：

```go
ninja.Get(r, "/:id", getUser,
    ninja.ResponseModel[UserOut](),
)
```

## Operation 安全选项

| 选项 | 作用 |
| --- | --- |
| `ninja.Security(name, scopes...)` | 只声明当前 operation 的 OpenAPI security requirement |
| `ninja.SecurityMiddleware(name, mw, scopes...)` | 声明 security，并给当前 operation 挂 Gin middleware |
| `ninja.BearerAuth(mw...)` | 使用默认 `bearerAuth` security，并挂中间件 |
| `ninja.BasicAuth(mw...)` | 使用默认 `basicAuth` security，并挂中间件 |
| `ninja.APIKeyAuth(name, mw...)` | 使用指定 API Key security，并挂中间件 |
| `ninja.OAuth2Auth(scopes...)` | 只声明默认 `oauth2` security |
| `ninja.OAuth2AuthMiddleware(mw, scopes...)` | 声明默认 `oauth2` security，并挂中间件 |

模块内大多数接口都需要认证时，优先放到 Router：

```go
r := ninja.NewRouter("/users",
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

只有某个接口需要额外认证时，放到 operation：

```go
adminOnly := middleware.JWTAuthWithConfig(cfg.JWT)

ninja.Delete(r, "/:id", deleteUser,
    ninja.Summary("Delete user"),
    ninja.BearerAuth(adminOnly),
)
```

## Operation 运行时选项

| 选项 | 作用 |
| --- | --- |
| `ninja.Timeout(d)` | 为当前 operation 设置协作式超时 |
| `ninja.RateLimit(requestsPerSecond, burst...)` | 为当前 operation 设置按客户端 IP 维护的内存 token bucket 限流 |
| `ninja.WithTransaction()` | 用请求级数据库事务包裹当前 operation |

`Timeout` 会取消请求上下文并尽早返回超时响应；长耗时逻辑仍需要主动监听 `ctx.Request.Context().Done()`。

```go
ninja.Get(r, "/slow", slowReport,
    ninja.Timeout(3*time.Second),
)
```

`WithTransaction()` 需要 API 配置事务处理器，并挂载 `orm.Middleware(db)`：

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})
api.UseGin(orm.Middleware(db))

ninja.Post(r, "/", createUser,
    ninja.WithTransaction(),
)
```

## Operation 缓存选项

| 选项 | 作用 |
| --- | --- |
| `ninja.Cache(ttl, opts...)` | 开启 route 级响应缓存，并默认启用 ETag |
| `ninja.CacheWithMaxBodyBytes(max)` | 限制缓存和 ETag 生成时最多缓冲的响应体大小 |
| `ninja.CacheControl(value)` | 设置成功响应的 `Cache-Control` 头 |
| `ninja.ETag()` | 为成功响应生成 ETag |

缓存适合稳定的读接口，不适合带副作用的写接口：

```go
ninja.Get(r, "/:id", getUser,
    ninja.Cache(30*time.Second),
    ninja.CacheControl("public, max-age=30"),
)
```

## 推荐放置规则

| 配置 | 推荐位置 |
| --- | --- |
| 模块 URL 前缀 | `ninja.NewRouter("/users")` |
| Swagger 分组 | Router 的 `WithTags(...)` |
| 模块级认证 | Router 的 `WithBearerAuth(...)` 或 `WithSecurity(...)` |
| 单接口说明 | Operation 的 `Summary(...)`、`Description(...)` |
| 单接口响应模型 | Operation 的 `ResponseModel(...)`、`Paginated(...)` |
| 单接口事务 | Operation 的 `WithTransaction()` |
| 单接口限流或超时 | Operation 的 `RateLimit(...)`、`Timeout(...)` |
| API 版本 | Router 的 `WithVersion(...)` |

一条实用规则是：读路由注册代码时，应该能一眼看出这个接口的路径、输入、响应、文档名称、认证要求和关键运行时行为。Router 放公共规则，Operation 放例外和细节。

上一篇：[ModelSchema 高级用法](./model-schema.md) | 下一篇：[项目与 CRUD 脚手架](./scaffolding.md)
