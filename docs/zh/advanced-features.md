# 高级功能

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

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
    Topic string `query:"topic" default:"system"`
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
    Room string `query:"room" default:"lobby"`
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

---

上一篇: [文件传输与 OpenAPI 控制](./files-and-openapi.md) | 下一篇: [Admin 与完整示例](./admin-and-examples.md)
