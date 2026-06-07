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
- 可通过传入 `CacheWithStore(...)` 让同一套 API 使用 Redis

常用选项：

```go
store := ninja.NewMemoryCacheStore()

ninja.Get(articles, "/:slug", getArticle,
    ninja.Cache(5*time.Minute,
        ninja.CacheWithStore(store),
        ninja.CacheWithKey(func(ctx *ninja.Context) string {
            return "article:" + ctx.Param("slug")
        }),
        ninja.CacheWithTags(func(ctx *ninja.Context) []string {
            return []string{"articles", "article:" + ctx.Param("slug")}
        }),
    ),
    ninja.CacheControl("public, max-age=300, stale-while-revalidate=60"),
    ninja.ETag(),
)
```

Redis 后端存储：

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

说明：

- 缓存支持面向安全的只读端点
- 只有确认响应可被共享代理/CDN 缓存时，才显式使用 `CacheControl("public, ...")`
- SSE / WebSocket 路由不会被缓存
- `NewCacheInvalidator(store)` 提供统一的删除 / 标签失效 / 锁入口
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

---

## SSE（Server-Sent Events）

使用 `ninja.SSE(...)` 实现单向服务端推送 / 文本流输出：

```go
type EventsInput struct {
    Topic string `query:"topic" default:"system"`
}

events := ninja.NewRouter("/events", ninja.WithTags("Events"))

ninja.SSE(events, "/stream", func(ctx *ninja.Context, in *EventsInput, stream *ninja.SSEStream) error {
    if err := stream.Send(ninja.SSEEvent{
        Event: "ready",
        Data: map[string]string{
            "topic": in.Topic,
            "status": "connected",
        },
    }); err != nil {
        return err
    }

    return stream.Send(ninja.SSEEvent{
        Event: "message",
        Data:  "hello from gin-ninja",
    })
})
```

默认响应头：

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

可以发送：

- 普通字符串
- 字节切片
- 结构体 / map（编码为 JSON）
- 通过 `ninja.SSEEvent` 发送 `ID`、`Event` 和 `Retry` 元数据

客户端示例：

```js
const source = new EventSource("/events/stream?topic=system");
source.addEventListener("message", (event) => {
  console.log(event.data);
});
```

---

## WebSocket

使用 `ninja.WebSocket(...)` 实现双向实时通信：

```go
type ChatInput struct {
    Room string `query:"room" default:"lobby"`
}

ws := ninja.NewRouter("/ws", ninja.WithTags("Realtime"))

ninja.WebSocket(ws, "/chat", func(ctx *ninja.Context, in *ChatInput, conn *ninja.WebSocketConn) error {
    text, err := conn.ReceiveText()
    if err != nil {
        return err
    }
    return conn.SendText(in.Room + ":" + text)
})
```

便捷辅助方法：

- `conn.SendText(...)`
- `conn.ReceiveText()`
- `conn.SendJSON(...)`
- `conn.ReceiveJSON(...)`

客户端示例：

```js
const ws = new WebSocket("ws://localhost:8080/ws/chat?room=lobby");
ws.onopen = () => ws.send("ping");
ws.onmessage = (event) => console.log(event.data);
```

OpenAPI 会把该路由记录为 `101 Switching Protocols` 响应，以便在生成的文档中显示升级行为。

---

上一篇: [文件传输与 OpenAPI 控制](./files-and-openapi.md) | 下一篇: [Admin 与完整示例](./admin-and-examples.md)
