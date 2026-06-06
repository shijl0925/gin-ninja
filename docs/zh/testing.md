# 使用 TestClient 测试 API

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/testing.md)

`ninjatest.TestClient` 可以直接测试 `NinjaAPI`、`Router` 或 `http.Handler`，不需要在每个测试里手动组合 `httptest.NewRecorder`、`httptest.NewRequest` 和 `api.Handler()`。

## 基本用法

```go
import (
    "net/http"
    "testing"

    ninja "github.com/shijl0925/gin-ninja"
    ninjatest "github.com/shijl0925/gin-ninja/testing"
)

func TestUsers(t *testing.T) {
    router := ninja.NewRouter("/users")
    ninja.Get(router, "/", listUsers)

    client := ninjatest.NewWithT(t, router)
    resp := client.Get("/users/", ninjatest.Query("page", "1"))

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.String())
    }

    var out []UserOut
    if err := resp.DecodeJSON(&out); err != nil {
        t.Fatalf("decode response: %v", err)
    }
}
```

## 支持的测试目标

`ninjatest.New(...)` 和 `ninjatest.NewWithT(...)` 支持：

- `*ninja.Router`：TestClient 会创建临时 `NinjaAPI` 并挂载该路由。
- `*ninja.NinjaAPI`：适合需要全局中间件、配置或多个路由的测试。
- `http.Handler`：适合自定义 Gin engine 或标准库 handler。

测试 `Router` 时如果需要 API 前缀、文档开关等配置，可以传入 `ninjatest.WithConfig(...)`。

## 请求与响应

- `Get`、`Post`、`Put`、`Patch`、`Delete` 和 `Request` 会发起内存内请求。
- struct、map、slice 和标量请求体会自动编码为 JSON，并默认设置 `Content-Type: application/json`。
- `url.Values` 请求体会编码为表单，并默认设置 `application/x-www-form-urlencoded`。
- `ninjatest.Multipart(...)` 可构造用于上传测试的 `multipart/form-data` 请求体。
- `io.Reader`、`[]byte` 和 `string` 请求体会原样发送；需要时可用 `ninjatest.Header(...)` 设置请求头。
- `ninjatest.WithHeader(...)` 会设置默认请求头；同名 header 的多次调用会覆盖前值。
- `NewRequest` 配合 `Do` 可自定义原始 `*http.Request`。
- 响应对象提供 `StatusCode`、`Header`、`Body`、`Cookies`、`String()` 和 `DecodeJSON(...)`。

```go
resp := client.Post("/users/",
    CreateUserInput{Name: "alice"},
    ninjatest.Header("X-Trace-ID", "test-1"),
    ninjatest.Cookie(&http.Cookie{Name: "mode", Value: "test"}),
)
```

```go
resp := client.Post("/uploads/",
    ninjatest.Multipart(
        url.Values{"title": {"demo"}},
        ninjatest.File("file", "demo.txt", "hello multipart"),
    ),
)
```
