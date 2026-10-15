# Advanced Features

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/advanced-features.md)

These features are optional for a first API, but useful once the service is running in production: response caching, ETag, per-operation timeout, rate limiting, request transactions, and API versioning.

## Route-Level Cache

Enable response caching on read endpoints:

```go
func getArticle(ctx *ninja.Context, in *GetArticleInput) (*ArticleOut, error) {
    return loadArticle(in.Slug)
}

ninja.Get(router, "/:slug", getArticle,
    ninja.Cache(5*time.Minute),
)
```

Default behavior:

- Safe read responses are cached.
- Successful responses include `ETag`.
- Without `CacheControl(...)`, `Cache(ttl)` emits `Cache-Control: private, max-age=<ttl>`.
- Matching `If-None-Match` requests return `304 Not Modified`.
- The default cache key varies by `Authorization` and `Accept-Language`.

Custom store and key:

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

Invalidate a key:

```go
invalidator := ninja.NewCacheInvalidator(store)
invalidator.Delete("article:welcome")
```

Limit how much response body data is buffered for cache storage and ETag generation:

```go
ninja.Get(router, "/large-report", report,
    ninja.Cache(5*time.Minute, ninja.CacheWithMaxBodyBytes(1<<20)),
)
```

Responses larger than the limit are written to the client but are not cached, and no cache ETag is generated for that response. Use a negative value only for trusted routes with predictably small responses.

For list endpoints with many cache keys, prefer a short TTL or an application-level invalidation strategy.

## Operation Timeout

```go
ninja.Get(router, "/slow", slowHandler,
    ninja.Timeout(2*time.Second),
)
```

`Timeout` cancels the request context and returns early. Handlers should still observe context cancellation:

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

## Operation Rate Limit

```go
ninja.Post(router, "/login", login,
    ninja.RateLimit(5, 10),
)
```

This uses an in-memory token bucket:

- `5` tokens are added per second.
- burst is `10`.
- if burst is omitted, it defaults to the requests-per-second value.

For multi-instance deployments, use a gateway, Redis, or a dedicated rate-limit service when you need a global limit.

Buckets are tracked per client IP for the current operation. One client IP shares its operation quota, while different IPs do not consume each other's tokens.

## Request Transactions

With `orm.Middleware(db)` and transaction handlers configured on the API, one operation can run inside a request-scoped transaction:

```go
api := ninja.New(ninja.Config{
    TransactionHandlers: orm.TransactionHandlers(),
})

api.UseGin(orm.Middleware(db))

ninja.Post(router, "/", createUser,
    ninja.WithTransaction(),
)
```

The transaction is rolled back when the handler returns an error or the request context is canceled; otherwise it is committed.

Manual control is also available:

```go
func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    if err := ctx.BeginTx(); err != nil {
        return nil, err
    }
    defer ctx.RollbackTx()

    // write database rows

    if err := ctx.CommitTx(); err != nil {
        return nil, err
    }
    return &UserOut{}, nil
}
```

For most handlers, `ninja.WithTransaction()` is shorter and less error-prone.

## API Versioning

Use versioned routers when v1 and v2 paths, schemas, or behavior have diverged:

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

This registers:

- `/api/v1/users`
- `/api/v2/users`
- `/openapi/v1.json`
- `/openapi/v2.json`
- `/docs/v1`
- `/docs/v2`

Deprecated versions automatically emit `Deprecation`, `Sunset`, and `Link: <...>; rel="deprecation"` headers when configured, and their versioned OpenAPI operations are marked as deprecated.

## When To Use What

| Feature | Use when |
| --- | --- |
| `Cache` / `ETag` | reads are frequent or expensive |
| `Timeout` | an endpoint has a clear latency budget |
| `RateLimit` | login, SMS, export, or other abuse-prone endpoints need protection |
| `WithTransaction` | one request performs multiple database writes atomically |
| `WithVersion` | an API version has incompatible behavior or schemas |

Previous: [Testing APIs with TestClient](./testing.md) | Next: [Optional Admin and Full Example](./admin-and-examples.md)
