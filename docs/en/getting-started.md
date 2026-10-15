# Getting Started

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/getting-started.md)

This page starts from an empty Go project and builds the first typed API. Run the code first, then read what each piece does.

## Create a Project

```bash
mkdir hello-api
cd hello-api
go mod init example.com/hello-api
go get github.com/shijl0925/gin-ninja
```

Create `main.go`:

```go
package main

import (
    "log"

    ninja "github.com/shijl0925/gin-ninja"
)

type HelloInput struct {
    Name string `query:"name" default:"world"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    api := ninja.New(ninja.Config{
        Title:   "Hello API",
        Version: "1.0.0",
    })

    router := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(router, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(router)

    log.Fatal(api.Run(":8080"))
}
```

Run it:

```bash
go run .
```

Call it:

```bash
curl "http://localhost:8080/hello/?name=alice"
```

Response:

```json
{"message":"Hello, alice!"}
```

Open in a browser:

- `http://localhost:8080/` — welcome page
- `http://localhost:8080/docs` — Swagger UI
- `http://localhost:8080/openapi.json` — OpenAPI JSON

## What The Code Does

| Code | Purpose |
| --- | --- |
| `ninja.New(...)` | Creates the API and prepares Gin, the homepage, OpenAPI, and docs UI |
| `HelloInput` | Declares request input; `query:"name"` reads from the query string |
| `HelloOutput` | Declares the JSON response shape |
| `func sayHello(...)` | A typed handler with plain Go input and output structs |
| `ninja.NewRouter("/hello")` | Creates a route group under `/hello` |
| `ninja.Get(...)` | Registers `GET /hello/` |
| `api.AddRouter(router)` | Mounts the route group on the API |

Typed handlers use this shape:

```go
func handler(ctx *ninja.Context, in *Input) (*Output, error)
```

If the handler returns an `error`, gin-ninja writes a standard error response. If the output is `nil`, gin-ninja writes `204 No Content`.

## Add Path Params And Request Body

Use `path:"..."` for path parameters and `json:"..."` for JSON request bodies:

```go
type GetUserInput struct {
    ID uint `path:"id" binding:"required"`
}

type CreateUserInput struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

type UserOut struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    return &UserOut{ID: in.ID, Name: "alice", Email: "alice@example.com"}, nil
}

func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    return &UserOut{ID: 1, Name: in.Name, Email: in.Email}, nil
}

router := ninja.NewRouter("/users", ninja.WithTags("Users"))
ninja.Get(router, "/:id", getUser, ninja.Summary("Get user"))
ninja.Post(router, "/", createUser, ninja.Summary("Create user"))
api.AddRouter(router)
```

`binding:"required,email"` uses `go-playground/validator` rules. You do not need to check validation manually in the handler.

## Organize Multiple Endpoints

As projects grow, avoid putting every endpoint in `main.go`. The two common options are Router and Controller.

### Router: lightweight grouping

Good for small projects or function-style handlers:

```go
func NewUserRouter() *ninja.Router {
    r := ninja.NewRouter("/users", ninja.WithTags("Users"))
    ninja.Get(r, "/:id", getUser)
    ninja.Post(r, "/", createUser)
    return r
}

api.AddRouter(NewUserRouter())
```

### Controller: resource-oriented grouping

Good when handlers share a database, service, or other dependency:

```go
type UserController struct {
    db *gorm.DB
}

func (c *UserController) Register(r *ninja.Router) {
    ninja.Get(r, "/:id", c.Get, ninja.Summary("Get user"))
    ninja.Post(r, "/", c.Create, ninja.Summary("Create user"))
}

func (c *UserController) Get(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    // query c.db
    return &UserOut{ID: in.ID, Name: "alice", Email: "alice@example.com"}, nil
}

func (c *UserController) Create(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    // create with c.db
    return &UserOut{ID: 1, Name: in.Name, Email: in.Email}, nil
}

api.AddController("/users", &UserController{db: db}, ninja.WithTags("Users"))
```

Selection guide:

| Situation | Use |
| --- | --- |
| A few simple endpoints | `Router` + plain functions |
| Full CRUD for one resource | `Controller` |
| Handlers share DB/service state | `Controller` |
| Tests or small demos | `Router` or `ninja.ControllerFunc` |

## Next Step

Read [NinjaAPI and Instance Configuration](./api-and-configuration.md) to understand the API instance, global configuration, docs routes, and server behavior.
