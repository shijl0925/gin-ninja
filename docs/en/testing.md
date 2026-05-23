# Testing APIs with TestClient

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/testing.md)

`ninjatest.TestClient` lets tests call a `NinjaAPI`, `Router`, or `http.Handler` without manually creating `httptest.NewRecorder`, `httptest.NewRequest`, and `api.Handler()`.

## Basic usage

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

## Targets

`ninjatest.New(...)` and `ninjatest.NewWithT(...)` accept:

- `*ninja.Router` — the client creates a temporary `NinjaAPI` and mounts the router.
- `*ninja.NinjaAPI` — useful when tests need global middleware, config, or multiple routers.
- `http.Handler` — useful for custom Gin engines or standard library handlers.

When testing a router and you need API config such as a prefix or custom docs settings, pass `ninjatest.WithConfig(...)`.

## Requests and responses

- `Get`, `Post`, `Put`, `Patch`, `Delete`, and `Request` execute in-memory requests.
- Structs, maps, slices, and scalar request bodies are encoded as JSON and default to `Content-Type: application/json`.
- `url.Values` bodies are encoded as forms and default to `application/x-www-form-urlencoded`.
- `ninjatest.Multipart(...)` builds `multipart/form-data` bodies for upload tests.
- `io.Reader`, `[]byte`, and `string` bodies are sent as-is; set headers with `ninjatest.Header(...)` when needed.
- `ninjatest.WithHeader(...)` sets a default header; repeated calls with the same name overwrite the previous value.
- `NewRequest` plus `Do` lets tests customize a raw `*http.Request`.
- Responses expose `StatusCode`, `Header`, `Body`, `Cookies`, `String()`, and `DecodeJSON(...)`. `Code` remains as a deprecated alias of `StatusCode`.

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
