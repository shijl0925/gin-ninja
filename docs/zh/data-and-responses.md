# 响应、错误与 Schema

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/data-and-responses.md)

请求输入由 input struct 负责，业务处理后返回的公开数据建议用 DTO struct 表达。gin-ninja 会把 handler 返回值写成 JSON，并用同一份类型生成 OpenAPI response schema。

这一页只关注“如何返回数据”。请求体请看 [Request Body 与 Form Data](./request-body-and-forms.md)，列表查询请看 [查询 Schema：分页、过滤与排序](./query-schemas.md)，完整增删改查请看 [CRUD 教程：Users 增删改查](./crud-tutorial.md)。

## 类型化响应

普通 handler 返回 `(*Out, error)`：

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

成功时框架会按 operation 的成功状态码写出 JSON。返回 `nil, nil` 时，框架写出 `204 No Content`。

## 默认成功状态码

| 注册函数 | 默认成功状态码 | 响应体 |
| --- | --- | --- |
| `ninja.Get` | `200 OK` | handler 返回的 `*Out` |
| `ninja.Post` | `201 Created` | handler 返回的 `*Out` |
| `ninja.Put` | `200 OK` | handler 返回的 `*Out` |
| `ninja.Patch` | `200 OK` | handler 返回的 `*Out` |
| `ninja.Delete` | `204 No Content` | 无响应体 |

需要改成功状态码时使用 `SuccessStatus`：

```go
ninja.Post(router, "/exports", createExport,
    ninja.Summary("Create export job"),
    ninja.SuccessStatus(202),
)
```

## DTO 优先

不要直接把 GORM model 暴露给公开 API。推荐用 DTO 明确响应字段：

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

好处是：

- 不会意外暴露 `Password`、内部状态、软删除字段。
- OpenAPI response schema 更稳定。
- 公开 API 可以和数据库结构独立演进。

## 分页响应

列表接口通常返回 `*pagination.Page[T]`：

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

响应结构：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "size": 20,
  "pages": 0
}
```

`ninja.Paginated[UserOut]()` 用于声明 OpenAPI 中的分页响应 schema。分页输入、过滤和排序的完整说明见 [查询 Schema：分页、过滤与排序](./query-schemas.md)。

游标分页使用 `*pagination.CursorPage[T]` 和 `ninja.CursorPaginated[T]()`。

## 额外响应文档

`Response` 用来声明额外的 OpenAPI response，例如 `404` 或 `409`：

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

`Response` 只补充文档；实际错误仍然由 handler 返回的 `error` 和框架错误处理写出。

## 错误响应

handler 返回 `error` 时，gin-ninja 会写出统一错误结构：

```json
{"code":404,"message":"not found","data":null}
```

常用内置错误：

```go
return nil, ninja.BadRequestError()
return nil, ninja.UnauthorizedError()
return nil, ninja.ForbiddenError()
return nil, ninja.NotFoundError()
return nil, ninja.ConflictError()
```

自定义状态码和消息：

```go
return nil, ninja.NewError(422, "email already exists")
```

请求绑定或 `binding:"..."` 校验失败时，handler 不会执行，框架会返回 `422 Unprocessable Entity`：

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

无效 JSON body 返回 `400 Bad Request`；超过 JSON body 限制返回 `413 Request Entity Too Large`。OpenAPI 会自动为 operation 补充 `422 Validation Error` 和 `500 Internal Server Error` 响应说明。若需要把 repository/service 层的通用错误统一映射成 HTTP 错误，见 [错误映射](./api-and-configuration.md#错误映射)。

在删除接口中，`ninja.Delete` 的 handler 只返回 `error`：

```go
func deleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
    if err := service.DeleteUser(ctx.Request.Context(), in.UserID); err != nil {
        return err
    }
    return nil
}
```

成功时返回 `204 No Content`。

## 原生 Gin 响应

类型化 handler 推荐直接返回 DTO。若你在原生 Gin handler 中工作，可以使用 `pkg/response`：

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

func health(c *gin.Context) {
    response.Success(c, gin.H{"status": "ok"})
}
```

常见 helper：

```go
response.Success(c, data)
response.NotFound(c, "user not found")
response.JSON(c, response.OKWithMessage("created", data))
```

## 可选高级用法：ModelSchema

公开 API 默认仍推荐显式 DTO。如果你在写内部后台、重复 CRUD、生成器代码，或确实希望根据 model 规则裁剪响应字段，可以使用 `ModelSchema`、`ResponseModel`、`ResponseSchema`、`PaginatedSchema` 等高级能力。

这部分机制会同时影响运行时 JSON 输出和 OpenAPI schema，也能配合 `Depth(n)` 自动推导 GORM preload。完整说明见 [ModelSchema 高级用法](./model-schema.md)。

## 选择建议

| 场景 | 推荐 |
| --- | --- |
| 对外业务 API | 显式 DTO struct |
| 列表接口 | `pagination.Page[T]` + `ninja.Paginated[T]()` |
| 需要声明额外错误响应 | `ninja.Response(...)` |
| 内部 CRUD 想复用 model schema | [ModelSchema 高级用法](./model-schema.md) |
| 原生 Gin handler | `pkg/response` |

上一篇：[查询 Schema：分页、过滤与排序](./query-schemas.md) | 下一篇：[CRUD 教程：Users 增删改查](./crud-tutorial.md)
