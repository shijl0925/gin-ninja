# Testing APIs with TestClient

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/testing.md)

`ninjatest.TestClient` tests a `Router`, `NinjaAPI`, or plain `http.Handler` in memory. You do not need to start a real port or manually wire `httptest.NewRecorder` for every test.

## Test A Router

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

When the target is a router, TestClient creates a temporary `NinjaAPI` and mounts the router for you.

If that temporary API needs specific configuration, pass `WithConfig`:

```go
client := ninjatest.NewWithT(t, router,
    ninjatest.WithConfig(ninja.Config{
        Title:  "Test API",
        Prefix: "/api",
    }),
)
```

## Test A Full API

Use `*ninja.NinjaAPI` when the test needs global middleware, multiple routers, configuration, or database context:

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

Plain `http.Handler` also works:

```go
client := ninjatest.NewWithT(t, api.Handler())
```

## Send Requests

Common helpers:

```go
client.Get("/users/")
client.Post("/users/", CreateUserInput{Name: "alice"})
client.Put("/users/1", UpdateUserInput{Name: "Alice"})
client.Patch("/users/1", map[string]any{"name": "Alice"})
client.Delete("/users/1")
```

Request options:

```go
resp := client.Post("/users/",
    CreateUserInput{Name: "alice"},
    ninjatest.Header("Authorization", "Bearer "+token),
    ninjatest.Query("preview", "true"),
    ninjatest.Cookie(&http.Cookie{Name: "mode", Value: "test"}),
)
```

Use `ninjatest.Headers(http.Header{...})` to copy several headers at once. `ninjatest.Query(...)` appends query parameters and is handy for pagination, filtering, and ordering tests.

`struct`, `map`, `slice`, and scalar bodies are JSON-encoded automatically with `Content-Type: application/json`.

Form request:

```go
form := url.Values{}
form.Set("name", "alice")

resp := client.Post("/users/", form)
```

## Test Uploads

```go
resp := client.Post("/uploads/",
    ninjatest.Multipart(
        url.Values{"title": {"demo"}},
        ninjatest.File("file", "demo.txt", "hello multipart"),
    ),
)
```

Multiple files:

```go
resp := client.Post("/uploads/",
    ninjatest.Multipart(nil,
        ninjatest.File("files", "a.txt", "A"),
        ninjatest.File("files", "b.txt", "B"),
    ),
)
```

## Default Headers

Set shared headers when creating the client:

```go
client := ninjatest.NewWithT(t, api,
    ninjatest.WithHeader("Authorization", "Bearer "+token),
)
```

Use `WithHeaders` when several default headers should be sent on every request:

```go
client := ninjatest.NewWithT(t, api,
    ninjatest.WithHeaders(http.Header{
        "Authorization": {"Bearer " + token},
        "X-Tenant-ID":   {"tenant-a"},
    }),
)
```

Per-request `ninjatest.Header(...)` overrides the default header with the same name.

## Response Object

Common fields and methods:

| Member | Purpose |
| --- | --- |
| `StatusCode` | HTTP status code |
| `Header` | response headers |
| `Body` | raw body bytes |
| `Cookies` | response cookies |
| `String()` | printable body string |
| `DecodeJSON(&out)` | decode JSON response |

Typical assertions:

```go
if resp.StatusCode != http.StatusCreated {
    t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.String())
}

if got := resp.Header.Get("ETag"); got == "" {
    t.Fatal("expected ETag")
}
```

## Testing Tips

- Prefer router tests for simple handlers; they are fast and low-dependency.
- Test the full `NinjaAPI` when middleware, auth, transactions, or OpenAPI prefixes matter.
- Add at least one integration test for uploads, auth, pagination, filtering, and error responses.
- For database tests, use temporary SQLite or transaction rollback to avoid cross-test pollution.

Previous: [File Transfer: Uploads and Downloads](./file-transfer.md) | Next: [Advanced Features](./advanced-features.md)
