# Responses, Errors, and Schemas

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/data-and-responses.md)

Request input belongs in input structs. Public output should usually be represented by DTO structs. gin-ninja writes typed handler return values as JSON and reflects the same types into OpenAPI response schemas.

This page focuses only on returning data. For request bodies, read [Request Body and Form Data](./request-body-and-forms.md). For list query parameters, read [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md). For a complete CRUD flow, read [CRUD Tutorial: Users](./crud-tutorial.md).

## Typed Responses

A regular handler returns `(*Out, error)`:

```go
type UserOut struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    user, err := service.GetUser(ctx.Request.Context(), in.UserID)
    if err != nil {
        return nil, err
    }
    out := toUserOut(user)
    return &out, nil
}

ninja.Get(router, "/:id", getUser,
    ninja.Summary("Get user"),
)
```

On success, gin-ninja writes JSON with the operation's success status code. Returning `nil, nil` writes `204 No Content`.

## Default Success Status

| Registration helper | Default success status | Body |
| --- | --- | --- |
| `ninja.Get` | `200 OK` | handler `*Out` |
| `ninja.Post` | `201 Created` | handler `*Out` |
| `ninja.Put` | `200 OK` | handler `*Out` |
| `ninja.Patch` | `200 OK` | handler `*Out` |
| `ninja.Delete` | `204 No Content` | no body |

Use `SuccessStatus` when you need a different success status:

```go
ninja.Post(router, "/exports", createExport,
    ninja.Summary("Create export job"),
    ninja.SuccessStatus(202),
)
```

## Prefer DTOs

Avoid exposing GORM models directly from public APIs. Use DTOs to make response fields explicit:

```go
type User struct {
    ID       uint
    Name     string
    Email    string
    Password string
}

type UserOut struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func toUserOut(u User) UserOut {
    return UserOut{ID: u.ID, Name: u.Name, Email: u.Email}
}
```

Benefits:

- Sensitive fields such as `Password` are not exposed accidentally.
- OpenAPI response schemas remain stable.
- Public API contracts can evolve separately from database models.

## Paginated Responses

List endpoints usually return `*pagination.Page[T]`:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    items, total, err := service.ListUsers(ctx.Request.Context(), in)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}

ninja.Get(router, "/", listUsers,
    ninja.Summary("List users"),
    ninja.Paginated[UserOut](),
)
```

Response shape:

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "size": 20,
  "pages": 0
}
```

`ninja.Paginated[UserOut]()` declares the paginated OpenAPI response schema. Pagination input, filtering, and ordering are covered in [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md).

Cursor pagination uses `*pagination.CursorPage[T]` with `ninja.CursorPaginated[T]()`.

## Extra Response Documentation

Use `Response` to document additional OpenAPI responses such as `404` or `409`:

```go
type ErrorOut struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

ninja.Get(router, "/:id", getUser,
    ninja.Summary("Get user"),
    ninja.Response(404, "User not found", &ErrorOut{}),
)
```

`Response` adds documentation only. Runtime errors are still written from handler-returned `error` values and framework error handling.

## Error Responses

When a handler returns `error`, gin-ninja writes a standard error shape:

```json
{"code":404,"message":"not found","data":null}
```

Common built-in errors:

```go
return nil, ninja.BadRequestError()
return nil, ninja.UnauthorizedError()
return nil, ninja.ForbiddenError()
return nil, ninja.NotFoundError()
return nil, ninja.ConflictError()
```

Custom status and message:

```go
return nil, ninja.NewError(422, "email already exists")
```

When request binding or `binding:"..."` validation fails, the handler is not called and the framework returns `422 Unprocessable Entity`:

```json
{
  "code": 422,
  "message": "request validation failed",
  "data": {
    "errors": [
      {"field": "email", "message": "must be a valid email"}
    ]
  }
}
```

Invalid JSON bodies return `400 Bad Request`; JSON bodies over the size limit return `413 Request Entity Too Large`. OpenAPI automatically includes `422 Validation Error` and `500 Internal Server Error` response descriptions for operations. To map common repository/service errors into HTTP errors, see [Error Mapping](./api-and-configuration.md#error-mapping).

For delete endpoints, `ninja.Delete` handlers return only `error`:

```go
func deleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
    if err := service.DeleteUser(ctx.Request.Context(), in.UserID); err != nil {
        return err
    }
    return nil
}
```

Success writes `204 No Content`.

## Raw Gin Responses

Typed handlers should usually return DTOs directly. If you are writing a raw Gin handler, use `pkg/response`:

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

func health(c *gin.Context) {
    response.Success(c, gin.H{"status": "ok"})
}
```

Common helpers:

```go
response.Success(c, data)
response.NotFound(c, "user not found")
response.JSON(c, response.OKWithMessage("created", data))
```

## Optional Advanced Usage: ModelSchema

Public APIs should still usually prefer explicit DTOs. If you are building internal admin tools, repetitive CRUD, generated code, or a route that intentionally derives response fields from model rules, use advanced APIs such as `ModelSchema`, `ResponseModel`, `ResponseSchema`, and `PaginatedSchema`.

These APIs affect both runtime JSON output and OpenAPI schemas, and can use `Depth(n)` to derive GORM preload paths. Read [ModelSchema Advanced Usage](./model-schema.md) for the full guide.

## Choosing A Response Style

| Case | Recommended shape |
| --- | --- |
| Public business API | Explicit DTO struct |
| List endpoint | `pagination.Page[T]` + `ninja.Paginated[T]()` |
| Document extra error responses | `ninja.Response(...)` |
| Internal CRUD wants reusable model-derived schemas | [ModelSchema Advanced Usage](./model-schema.md) |
| Raw Gin handler | `pkg/response` |

Previous: [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md) | Next: [CRUD Tutorial: Users](./crud-tutorial.md)
