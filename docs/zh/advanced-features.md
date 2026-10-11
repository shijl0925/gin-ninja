# 高级功能

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 路由级缓存 / ETag / Cache-Control

对于只读端点，可以启用内置响应缓存和条件请求：

```go
type ArticleInput struct {
    Slug string `path:"slug" binding:"required"`
}

type ArticleOutput struct {
    Slug    string `json:"slug"`
    Title   string `json:"title"`
    Content string `json:"content"`
}

func getArticle(ctx *ninja.Context, in *ArticleInput) (*ArticleOutput, error) {
    return &ArticleOutput{
        Slug:    in.Slug,
        Title:   "gin-ninja cache demo",
        Content: "This response can be cached",
    }, nil
}

articles := ninja.NewRouter("/articles", ninja.WithTags("Articles"))

ninja.Get(articles, "/:slug", getArticle,
    ninja.Summary("Get article"),
    ninja.Cache(5*time.Minute),
)
```

行为：

- `Cache(ttl)` 使用默认内存后端启用路由缓存
- 成功的 GET/HEAD 响应会自动包含 `ETag`
- 未显式设置 `CacheControl(...)` 时，`Cache(ttl)` 会输出 `Cache-Control: private, max-age=<ttl>`
- 默认缓存键与 `Vary` 响应头包含 `Authorization` 和 `Accept-Language`；如需加入更多请求维度，请使用 `CacheWithKey(...)`
- 当缓存实体标签匹配时，携带 `If-None-Match` 的请求会返回 `304 Not Modified`
- 可通过传入 `CacheWithStore(...)` 使用自定义轻量存储

常用选项：

```go
store := ninja.NewMemoryCacheStore()

articleCacheKey := func(ctx *ninja.Context) string {
    return "article:" + ctx.Param("slug")
}

ninja.Get(articles, "/:slug", getArticle,
    ninja.Cache(5*time.Minute,
        ninja.CacheWithStore(store),
        ninja.CacheWithKey(articleCacheKey),
    ),
    ninja.CacheControl("public, max-age=300, stale-while-revalidate=60"),
    ninja.ETag(),
)

invalidator := ninja.NewCacheInvalidator(store)
invalidator.Delete("article:welcome")
```

说明：

- 缓存支持面向安全的只读端点
- 只有确认响应可被共享代理/CDN 缓存时，才显式使用 `CacheControl("public, ...")`
- `NewCacheInvalidator(store)` 只提供显式 key 删除；列表类接口如果无法穷举 key，建议依赖短 TTL 或应用层失效策略
- OpenAPI 会自动记录 `ETag` 和 `Cache-Control` 响应头

---

## API 版本管理

gin-ninja 现在除了全局前缀外，还支持版本感知路由。

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
            SunsetTime:   time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC),
            MigrationURL: "https://example.com/migrate-to-v2",
        },
        "v2": {
            Prefix:      "/v2",
            Description: "Current stable API",
        },
    },
})

v1Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v1"))
v2Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v2"))

ninja.Get(v1Users, "/", listUsersV1, ninja.Summary("List users (v1)"))
ninja.Get(v2Users, "/", listUsersV2, ninja.Summary("List users (v2)"))

api.AddRouter(v1Users)
api.AddRouter(v2Users)
```

这会注册：

- `GET /api/v1/users`
- `GET /api/v2/users`
- `GET /openapi/v1.json`
- `GET /openapi/v2.json`
- `GET /docs/v1`
- `GET /docs/v2`

弃用行为：

- 当版本标记为 `Deprecated: true` 时，响应会包含 `Deprecation: true`
- 配置 `Sunset` 时会输出该响应头
- 配置 `MigrationURL` 时会输出 `Link: <...>; rel="deprecation"`
- 版本化 OpenAPI 输出会把废弃版本中的操作标记为 `deprecated: true`

推荐模式：

- 保留 `Config.Prefix` 用于 `/api` 这类共享顶层命名空间
- 对属于特定 API 代际的路由使用 `WithVersion("v1")`、`WithVersion("v2")`
- 当版本语义发生分化时，使用独立的处理器 / schema 类型

上一篇: [文件传输与 OpenAPI 控制](./files-and-openapi.md) | 下一篇: [Admin 与完整示例](./admin-and-examples.md)
