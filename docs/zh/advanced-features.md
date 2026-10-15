# 高级功能

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/advanced-features.md)

本页介绍不一定每个项目都需要、但上线后很常用的能力：缓存、ETag、操作级超时、限流、事务和 API 版本管理。

## 路由级缓存

只读接口可以启用响应缓存：

```go
func getArticle(ctx *ninja.Context, in *GetArticleInput) (*ArticleOut, error) {
    return loadArticle(in.Slug)
}

ninja.Get(router, "/:slug", getArticle,
    ninja.Cache(5*time.Minute),
)
```

默认行为：

- 只缓存安全读请求。
- 成功响应会带 `ETag`。
- 未设置 `CacheControl(...)` 时，默认输出 `Cache-Control: private, max-age=<ttl>`。
- 携带匹配 `If-None-Match` 的请求会返回 `304 Not Modified`。
- 默认缓存键会考虑 `Authorization` 和 `Accept-Language`。

自定义缓存存储和 key：

```go
store := ninja.NewMemoryCacheStore()

cacheKey := func(ctx *ninja.Context) string {
    return "article:" + ctx.Param("slug")
}

ninja.Get(router, "/:slug", getArticle,
    ninja.Cache(5*time.Minute,
        ninja.CacheWithStore(store),
        ninja.CacheWithKey(cacheKey),
    ),
    ninja.CacheControl("public, max-age=300"),
    ninja.ETag(),
)
```

失效指定 key：

```go
invalidator := ninja.NewCacheInvalidator(store)
invalidator.Delete("article:welcome")
```

限制缓存和 ETag 生成时缓冲的响应体大小：

```go
ninja.Get(router, "/large-report", report,
    ninja.Cache(5*time.Minute, ninja.CacheWithMaxBodyBytes(1<<20)),
)
```

超过限制的响应会直接写给客户端，不会进入缓存，也不会为该响应生成缓存用 ETag。对确定很小且可信的响应，可以传入负数关闭这个限制。

如果列表接口 key 很多且不容易枚举，优先使用短 TTL 或应用层失效策略。

## 操作级超时

对慢接口设置超时：

```go
ninja.Get(router, "/slow", slowHandler,
    ninja.Timeout(2*time.Second),
)
```

`Timeout` 会取消请求 context 并尽早返回超时响应。处理器里的数据库、HTTP 调用或循环也要主动监听 context：

```go
func slowHandler(ctx *ninja.Context, _ *struct{}) (*Out, error) {
    select {
    case <-time.After(5 * time.Second):
        return &Out{OK: true}, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

## 操作级限流

对某个接口使用内存 token bucket：

```go
ninja.Post(router, "/login", login,
    ninja.RateLimit(5, 10),
)
```

含义：

- 每秒补充 `5` 个 token。
- burst 为 `10`。
- 不传 burst 时，burst 默认等于每秒请求数。

这是单进程内存限流。多实例部署时，如果需要全局限流，应在网关、Redis 或专门限流服务里实现。

限流桶按客户端 IP 维护。同一个客户端 IP 会共享当前 operation 的配额，不同 IP 之间互不占用。

## 请求事务

如果你使用 `orm.Middleware(db)`，并在 API 配置中设置事务处理器，就可以让单个 operation 自动包在事务里：

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})

api.UseGin(orm.Middleware(db))

ninja.Post(router, "/", createUser,
    ninja.WithTransaction(),
)
```

当 handler 返回错误或请求 context 已取消时，事务会回滚；成功时提交。

handler 内也可以手动控制事务：

```go
func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    if err := ctx.BeginTx(); err != nil {
        return nil, err
    }
    defer ctx.RollbackTx()

    // 写数据库

    if err := ctx.CommitTx(); err != nil {
        return nil, err
    }
    return &UserOut{}, nil
}
```

日常更推荐使用 `ninja.WithTransaction()`，代码更短也更不容易漏掉回滚。

## API 版本管理

当 v1 和 v2 的路径、schema 或行为已经分化时，用版本路由：

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
            SunsetTime:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
            MigrationURL: "https://example.com/migrate-to-v2",
        },
        "v2": {
            Prefix:      "/v2",
            Description: "Current API",
        },
    },
})

v1 := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v1"))
v2 := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v2"))

ninja.Get(v1, "/", listUsersV1, ninja.Summary("List users v1"))
ninja.Get(v2, "/", listUsersV2, ninja.Summary("List users v2"))

api.AddRouter(v1)
api.AddRouter(v2)
```

会注册：

- `/api/v1/users`
- `/api/v2/users`
- `/openapi/v1.json`
- `/openapi/v2.json`
- `/docs/v1`
- `/docs/v2`

废弃版本会自动输出：

- `Deprecation`
- `Sunset`
- `Link: <...>; rel="deprecation"`

并且版本化 OpenAPI 中的操作会标记为 deprecated。

## 什么时候使用高级功能

| 功能 | 使用时机 |
| --- | --- |
| `Cache` / `ETag` | 读多写少、响应生成成本高或客户端会重复请求 |
| `Timeout` | 外部依赖慢、接口需要明确 SLA |
| `RateLimit` | 登录、短信、导出等容易被滥用的接口 |
| `WithTransaction` | 一个请求里有多次数据库写入，需要原子性 |
| `WithVersion` | API 版本已经出现不兼容差异 |

上一篇：[使用 TestClient 测试 API](./testing.md) | 下一篇：[可选 Admin 与完整示例](./admin-and-examples.md)
