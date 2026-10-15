# 使用 TestClient 测试 API

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/testing.md)

`ninjatest.TestClient` 可以在内存里测试 `Router`、`NinjaAPI` 或普通 `http.Handler`，不需要启动真实端口，也不需要每个测试都手写 `httptest.NewRecorder`。

## 测试一个 Router

```go
import (
    "net/http"
    "testing"

    ninja "github.com/shijl0925/gin-ninja"
    ninjatest "github.com/shijl0925/gin-ninja/testing"
)

func TestGetUser(t *testing.T) {
    router := ninja.NewRouter("/users")
    ninja.Get(router, "/:id", getUser)

    client := ninjatest.NewWithT(t, router)
    resp := client.Get("/users/1")

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.String())
    }

    var out UserOut
    if err := resp.DecodeJSON(&out); err != nil {
        t.Fatalf("decode response: %v", err)
    }
    if out.ID != 1 {
        t.Fatalf("expected user 1, got %d", out.ID)
    }
}
```

测试 Router 时，TestClient 会临时创建一个 `NinjaAPI` 并挂载该 router。

如果这个临时 API 需要特定配置，可以传入 `WithConfig`：

```go
client := ninjatest.NewWithT(t, router,
    ninjatest.WithConfig(ninja.Config{
        Title:  "Test API",
        Prefix: "/api",
    }),
)
```

## 测试完整 API

当测试需要全局中间件、多个 router、配置或数据库上下文时，直接传入 `*ninja.NinjaAPI`：

```go
func TestAPI(t *testing.T) {
    api := buildAPI(testDB(t))
    client := ninjatest.NewWithT(t, api)

    resp := client.Get("/api/v1/users/", ninjatest.Query("page", "1"))
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("unexpected response: %s", resp.String())
    }
}
```

如果只测 `http.Handler`，也可以传入普通 handler：

```go
client := ninjatest.NewWithT(t, api.Handler())
```

## 发请求

常用方法：

```go
client.Get("/users/")
client.Post("/users/", CreateUserInput{Name: "alice"})
client.Put("/users/1", UpdateUserInput{Name: "Alice"})
client.Patch("/users/1", map[string]any{"name": "Alice"})
client.Delete("/users/1")
```

请求选项：

```go
resp := client.Post("/users/",
    CreateUserInput{Name: "alice"},
    ninjatest.Header("Authorization", "Bearer "+token),
    ninjatest.Query("preview", "true"),
    ninjatest.Cookie(&http.Cookie{Name: "mode", Value: "test"}),
)
```

多个 header 可以用 `ninjatest.Headers(http.Header{...})` 一次传入。`ninjatest.Query(...)` 会向 URL 追加 query 参数，适合测试分页、过滤和排序。

`struct`、`map`、`slice` 和标量会自动编码为 JSON，并设置 `Content-Type: application/json`。

表单请求：

```go
form := url.Values{}
form.Set("name", "alice")

resp := client.Post("/users/", form)
```

## 测试上传

```go
resp := client.Post("/uploads/",
    ninjatest.Multipart(
        url.Values{"title": {"demo"}},
        ninjatest.File("file", "demo.txt", "hello multipart"),
    ),
)
```

多文件：

```go
resp := client.Post("/uploads/",
    ninjatest.Multipart(nil,
        ninjatest.File("files", "a.txt", "A"),
        ninjatest.File("files", "b.txt", "B"),
    ),
)
```

## 默认请求头

如果多个请求都需要相同 header，可以在创建 client 时设置：

```go
client := ninjatest.NewWithT(t, api,
    ninjatest.WithHeader("Authorization", "Bearer "+token),
)
```

需要一次设置多个默认 header 时，用 `WithHeaders`：

```go
client := ninjatest.NewWithT(t, api,
    ninjatest.WithHeaders(http.Header{
        "Authorization": {"Bearer " + token},
        "X-Tenant-ID":   {"tenant-a"},
    }),
)
```

单个请求中的 `ninjatest.Header(...)` 会覆盖默认 header。

## 响应对象

响应对象常用字段和方法：

| 成员 | 作用 |
| --- | --- |
| `StatusCode` | HTTP 状态码 |
| `Header` | 响应头 |
| `Body` | 原始 body 字节 |
| `Cookies` | 响应 Cookie |
| `String()` | 方便打印 body |
| `DecodeJSON(&out)` | 解码 JSON 响应 |

常见断言：

```go
if resp.StatusCode != http.StatusCreated {
    t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.String())
}

if got := resp.Header.Get("ETag"); got == "" {
    t.Fatal("expected ETag")
}
```

## 测试建议

- handler 逻辑简单时，优先测试 `Router`，速度快、依赖少。
- 涉及中间件、认证、事务、OpenAPI 路由前缀时，测试完整 `NinjaAPI`。
- 上传、认证、分页、过滤和错误响应都建议至少有一条集成测试。
- 数据库测试使用临时 SQLite 或事务回滚，避免测试之间互相污染。

上一篇：[文件传输：上传与下载](./file-transfer.md) | 下一篇：[高级功能](./advanced-features.md)
