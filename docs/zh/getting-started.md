# 快速开始

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/getting-started.md)

本页从一个空 Go 项目开始，写出第一个类型化 API。先跟着代码跑起来，再回头看每一部分的作用。

## 创建项目

```bash
mkdir hello-api
cd hello-api
go mod init example.com/hello-api
go get github.com/shijl0925/gin-ninja
```

创建 `main.go`：

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

运行：

```bash
go run .
```

访问：

```bash
curl "http://localhost:8080/hello/?name=alice"
```

响应：

```json
{"message":"Hello, alice!"}
```

浏览器打开：

- `http://localhost:8080/`：欢迎页
- `http://localhost:8080/docs`：Swagger UI
- `http://localhost:8080/openapi.json`：OpenAPI JSON

## 这段代码做了什么

| 代码 | 作用 |
| --- | --- |
| `ninja.New(...)` | 创建 API 实例，并准备 Gin engine、首页、OpenAPI 和文档 UI |
| `HelloInput` | 声明请求输入；`query:"name"` 表示从查询参数读取 |
| `HelloOutput` | 声明 JSON 响应结构 |
| `func sayHello(...)` | 类型化处理函数，输入和输出都是普通 Go 结构体 |
| `ninja.NewRouter("/hello")` | 创建路由分组，所有接口都带 `/hello` 前缀 |
| `ninja.Get(...)` | 注册 `GET /hello/` |
| `api.AddRouter(router)` | 把这组接口挂到 API 上 |

处理函数统一是这个形状：

```go
func handler(ctx *ninja.Context, in *Input) (*Output, error)
```

如果返回 `error`，框架会写出标准错误响应；如果返回 `nil` 输出，框架会写出 `204 No Content`。

## 添加路径参数和请求体

路径参数用 `path:"..."`，JSON 请求体用 `json:"..."`：

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

`binding:"required,email"` 使用的是 `go-playground/validator` 的规则。校验失败时不需要在处理函数里手动判断。

## 组织多个接口

项目稍微变大后，不建议把所有接口都写在 `main.go`。最常见的两种组织方式是 Router 和 Controller。

### Router：轻量分组

适合小项目或函数式 handler：

```go
func NewUserRouter() *ninja.Router {
    r := ninja.NewRouter("/users", ninja.WithTags("Users"))
    ninja.Get(r, "/:id", getUser)
    ninja.Post(r, "/", createUser)
    return r
}

api.AddRouter(NewUserRouter())
```

### Controller：同一资源集中管理

适合需要注入数据库、service 或其他依赖的资源：

```go
type UserController struct {
    db *gorm.DB
}

func (c *UserController) Register(r *ninja.Router) {
    ninja.Get(r, "/:id", c.Get, ninja.Summary("Get user"))
    ninja.Post(r, "/", c.Create, ninja.Summary("Create user"))
}

func (c *UserController) Get(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    // 使用 c.db 查询用户
    return &UserOut{ID: in.ID, Name: "alice", Email: "alice@example.com"}, nil
}

func (c *UserController) Create(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    // 使用 c.db 创建用户
    return &UserOut{ID: 1, Name: in.Name, Email: in.Email}, nil
}

api.AddController("/users", &UserController{db: db}, ninja.WithTags("Users"))
```

选型建议：

| 情况 | 推荐 |
| --- | --- |
| 只有几个简单接口 | `Router` + 普通函数 |
| 同一资源有完整 CRUD | `Controller` |
| handler 需要共享 DB/service | `Controller` |
| 临时测试或 demo | `Router` 或 `ninja.ControllerFunc` |

## 接下来读什么

下一步建议阅读 [NinjaAPI 与实例配置](./api-and-configuration.md)，先理解 API 实例、全局配置、文档路由和 server 行为。
