# gin-ninja

A **django-ninja**-inspired web framework built on top of [Gin](https://github.com/gin-gonic/gin) with automatic OpenAPI 3.0 documentation, type-safe request/response handling, and first-class [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) ORM integration.

## Features

- **Type-safe handlers** – use plain Go structs for request input and response output.
- **Automatic parameter binding** – path params (`path:`), query params (`form:`), request headers (`header:`), and JSON bodies (`json:`) are all bound via struct tags.
- **Validation** – powered by [go-playground/validator](https://github.com/go-playground/validator) using the standard `binding:` tag.
- **Auto-generated OpenAPI 3.0 docs** – served as `/openapi.json`.
- **Swagger UI** – available at `/docs` out of the box.
- **Router groups** – nest routers with shared prefixes and tags.
- **Middleware** – per-router middleware that can abort requests with typed errors.
- **Pagination** – reusable `PageInput` and `Page[T]` types for consistent list responses.
- **ORM integration** – thin helpers around [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) for repository/service patterns.

---

## Installation

```bash
go get github.com/shijl0925/gin-ninja
```

---

## Quick Start

```go
package main

import (
    "log"

    ninja "github.com/shijl0925/gin-ninja"
)

// --- Schemas ---

type HelloInput struct {
    Name string `form:"name" binding:"required"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

// --- Handler ---

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

// --- Main ---

func main() {
    api := ninja.New(ninja.Config{
        Title:   "Hello API",
        Version: "1.0.0",
    })

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

Visit `http://localhost:8080/docs` for the Swagger UI, or `http://localhost:8080/openapi.json` for the raw spec.

---

## Parameter Binding

Struct tags control where each field is sourced from:

| Tag        | Source               | Methods            |
|------------|----------------------|--------------------|
| `path:"x"` | URL path parameter   | all                |
| `form:"x"` | URL query string     | all                |
| `header:"x"` | Request header     | all                |
| `json:"x"` | JSON request body    | POST / PUT / PATCH |

`binding:"..."` tags are validated via [go-playground/validator](https://github.com/go-playground/validator).

```go
// GET /users/:id?include_deleted=true
type GetUserInput struct {
    UserID         uint `path:"id"              binding:"required"`
    IncludeDeleted bool `form:"include_deleted"`
}

// POST /users
type CreateUserInput struct {
    Name  string `json:"name"  binding:"required"`
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age"   binding:"omitempty,min=0"`
}

// PUT /users/:id  (path param + JSON body)
type UpdateUserInput struct {
    UserID uint   `path:"id" binding:"required"`
    Name   string `json:"name"`
    Email  string `json:"email" binding:"omitempty,email"`
}
```

---

## Route Registration

```go
r := ninja.NewRouter("/users", ninja.WithTags("Users"))

ninja.Get(r,    "/",    listUsers,  ninja.Summary("List users"))
ninja.Post(r,   "/",    createUser, ninja.Summary("Create user"))
ninja.Get(r,    "/:id", getUser,    ninja.Summary("Get user"))
ninja.Put(r,    "/:id", updateUser, ninja.Summary("Update user"))
ninja.Delete(r, "/:id", deleteUser, ninja.Summary("Delete user"))

api.AddRouter(r)
```

**DELETE handlers** return no body; use the void signature:

```go
func deleteUser(ctx *ninja.Context, in *DeleteInput) error { ... }
```

All other handlers use:

```go
func handler(ctx *ninja.Context, in *Input) (*Output, error) { ... }
```

---

## Error Handling

Return a `*ninja.Error` to send a specific HTTP status code:

```go
return nil, ninja.NewError(http.StatusNotFound, "user not found")
// or use the pre-built errors:
return nil, ninja.ErrNotFound
```

Validation errors are automatically converted to `422 Unprocessable Entity` with a field-level error list.

---

## Pagination

```go
import "github.com/shijl0925/gin-ninja/pagination"

type ListUsersInput struct {
    pagination.PageInput           // adds ?page=&size= query params
    Search string `form:"search"`
}

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    items, total, err := svc.Page(in.GetPage(), in.GetSize(), opts...)
    return pagination.NewPage(items, total, in.PageInput), err
}
```

---

## ORM Integration (gormx)

```go
import (
    "github.com/shijl0925/gin-ninja/orm"
    "github.com/shijl0925/go-toolkits/gormx"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// 1. Initialise the global gormx instance.
db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
db.AutoMigrate(&User{})
orm.Init(db)

// 2. Attach ORM middleware (optional, for per-request DB access).
api.Engine().Use(orm.Middleware(db))

// 3. Use gormx repository / query builder in handlers.
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    repo := &gormx.BaseRepo[User]{}

    query, u := gormx.NewQuery[User]()
    query.Like(&u.Name, "%"+in.Search+"%").
          Limit(in.GetSize()).Offset(in.Offset())

    items, _ := repo.SelectListByOpts(query.ToOptions()...)
    total, _ := repo.SelectCount()
    ...
}

// 4. Access the request-scoped DB in a handler.
func myHandler(ctx *ninja.Context, _ *struct{}) (*Out, error) {
    db := orm.WithContext(ctx.Context)
    var users []User
    db.Find(&users)
    ...
}
```

---

## Nested Routers

```go
api := ninja.New(ninja.Config{Prefix: "/api/v1"})

users := ninja.NewRouter("/users", ninja.WithTags("Users"))
posts := ninja.NewRouter("/:userID/posts", ninja.WithTags("Posts"))

ninja.Get(posts, "/", listPosts)
users.AddRouter(posts)
api.AddRouter(users)

// Results in: GET /api/v1/users/:userID/posts/
```

---

## Middleware

```go
r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
r.Use(func(ctx *ninja.Context) error {
    if ctx.GetHeader("X-Admin-Token") != "secret" {
        return ninja.ErrUnauthorized
    }
    return nil
})
```

---

## Complete Example

See [examples/basic](./examples/basic/main.go) for a full CRUD API with SQLite.

```bash
cd examples/basic
go run .
# Open http://localhost:8080/docs
```

---

## License

[MIT](./LICENSE)
