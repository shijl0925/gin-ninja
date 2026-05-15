# Advanced Features

## Route-Level Cache / ETag / Cache-Control

For read-only endpoints, you can enable built-in response caching and conditional requests:

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

Behavior:

- `Cache(ttl)` enables route caching with the default in-memory backend
- successful GET/HEAD responses automatically include `ETag`
- when `CacheControl(...)` is not set explicitly, `Cache(ttl)` emits `Cache-Control: public, max-age=<ttl>`
- requests with `If-None-Match` return `304 Not Modified` when the cached entity tag matches
- the same API can target Redis by passing `CacheWithStore(...)`

Useful options:

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

Redis-backed store:

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

Notes:

- cache support is intended for safe read endpoints
- SSE / WebSocket routes are not cached
- `NewCacheInvalidator(store)` provides a unified delete / tag-invalidation / lock entry point
- OpenAPI automatically documents `ETag` and `Cache-Control` response headers

---

## API Version Management

gin-ninja now supports version-aware routing in addition to a global prefix.

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

v1Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v1"))
v2Users := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithVersion("v2"))

ninja.Get(v1Users, "/", listUsersV1, ninja.Summary("List users (v1)"))
ninja.Get(v2Users, "/", listUsersV2, ninja.Summary("List users (v2)"))

api.AddRouter(v1Users)
api.AddRouter(v2Users)
```

This registers:

- `GET /api/v1/users`
- `GET /api/v2/users`
- `GET /openapi/v1.json`
- `GET /openapi/v2.json`
- `GET /docs/v1`
- `GET /docs/v2`

Deprecation behavior:

- when a version is marked `Deprecated: true`, responses include `Deprecation: true`
- `Sunset` is emitted when configured
- `Link: <...>; rel="deprecation"` is emitted when `MigrationURL` is configured
- versioned OpenAPI output marks operations in deprecated versions as `deprecated: true`

Recommended pattern:

- keep `Config.Prefix` for a shared top-level namespace such as `/api`
- use `WithVersion("v1")`, `WithVersion("v2")` on routers that belong to a specific API generation
- use separate handlers/schema types when versions diverge semantically

---

## SSE (Server-Sent Events)

Use `ninja.SSE(...)` for one-way server push / streaming text output:

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

Default response headers:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

You can send:

- plain strings
- byte slices
- structs / maps (encoded as JSON)
- `ID`, `Event`, and `Retry` metadata via `ninja.SSEEvent`

Example client:

```js
const source = new EventSource("/events/stream?topic=system");
source.addEventListener("message", (event) => {
  console.log(event.data);
});
```

---

## WebSocket

Use `ninja.WebSocket(...)` for bidirectional realtime communication:

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

Convenience helpers:

- `conn.SendText(...)`
- `conn.ReceiveText()`
- `conn.SendJSON(...)`
- `conn.ReceiveJSON(...)`

Example client:

```js
const ws = new WebSocket("ws://localhost:8080/ws/chat?room=lobby");
ws.onopen = () => ws.send("ping");
ws.onmessage = (event) => console.log(event.data);
```

OpenAPI documents the route as a `101 Switching Protocols` response so the upgrade is visible in generated docs.

---
