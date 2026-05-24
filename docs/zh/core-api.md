# 核心 API、绑定与响应

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 核心 API

### NinjaAPI

常用能力：

- `ninja.New(config)`：创建 API 实例
- `api.AddRouter(router)`：注册路由组
- `api.UseGin(mw...)`：注册 Gin 中间件
- `api.Run(addr)`：启动服务并处理优雅关闭
- `api.OnStartup(...)` / `api.OnShutdown(...)`：生命周期钩子
- `api.Shutdown(ctx)`：手动优雅关闭

### Router

常用能力：

- `ninja.NewRouter(prefix, opts...)`
- `router.AddRouter(sub)`：嵌套子路由
- `router.UseGin(...)`：注册路由级 Gin 中间件
- `ninja.WithTags(...)`、`ninja.WithVersion(...)`、`ninja.WithBearerAuth(authMiddleware)` 等 RouterOption

### Context

常用辅助方法：

- `ctx.RequestID()`：获取请求 ID
- `ctx.GetUserID()`：读取 JWT 中的用户 ID
- `ctx.Locale()` / `ctx.T(...)`：读取协商语言与翻译消息
- `ctx.JSON200(...)` / `ctx.JSON201(...)` / `ctx.JSON204()`：快捷响应
- `ctx.Forbidden(...)` / `ctx.Unauthorized(...)`：快捷错误返回

## 参数绑定与校验

支持的标签如下：

| 标签 | 来源 | 适用方法 |
| --- | --- | --- |
| `path:"x"` | URL 路径参数 | 全部 |
| `query:"x"` | URL 查询参数 | 全部 |
| `form:"x"` | 表单请求体字段 | POST / PUT / PATCH 的 `application/x-www-form-urlencoded` 或 multipart 请求 |
| `header:"x"` | 请求头 | 全部 |
| `cookie:"x"` | Cookie | 全部 |
| `json:"x"` | JSON 请求体 | POST / PUT / PATCH |
| `file:"x"` | Multipart 上传文件 | POST / PUT / PATCH |

补充规则：

- `binding:"..."` 使用 `go-playground/validator`
- `default:"..."` 适用于 `query`、`form`、`header`、`cookie`
- multipart 请求中可以把普通 `form` 字段和 `file` 字段写在同一个结构体里

## 响应模型与错误处理

### ModelSchema 风格响应

```go
type User struct {
    ID       uint   `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type UserOut struct {
    ninja.ModelSchema[User] `fields:"id,name,email" exclude:"password"`
}

func getUser(ctx *ninja.Context, in *struct{}) (*UserOut, error) {
    return ninja.BindModelSchema[UserOut](User{
        ID:       1,
        Name:     "alice",
        Email:    "alice@example.com",
        Password: "secret",
    })
}
```

- `fields:"..."` 控制输出字段白名单
- `exclude:"..."` 用于排除敏感字段
- 过滤规则同时作用于 JSON 响应和 OpenAPI schema

### 协议错误与验证错误

`*ninja.Error` 表示协议级错误，使用对应 HTTP 状态码返回。

```go
return nil, ninja.NewError(http.StatusForbidden, "account is disabled")
```

`ValidationError` 会返回 HTTP 422。

## 标准响应信封

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

response.Success(c, users)
response.NotFound(c, "user not found")
response.JSON(c, response.OKWithMessage("created", user))
```

成功响应默认格式：

```json
{"code": 200, "message": "success", "data": {...}}
```

框架错误响应使用相同的根级信封：

```json
{"code": 404, "message": "not found", "data": null}
```

## 过滤、排序与分页

### 分页

使用 `pagination.PageInput` 和 `pagination.Page[T]`：

```go
type ListUsersInput struct {
    pagination.PageInput
}
```

### 声明式过滤

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search" filter:"name|email,like"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq"`
}
```

支持操作符：`eq`、`ne`、`gt`、`ge`、`lt`、`le`、`like`、`in`。

```go
filterOpts, err := filter.BuildOptions(in)
```

### 安全排序

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `query:"sort" order:"id|name|email|age|created_at"`
}
```

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

白名单之外的排序字段会被拒绝，不会直接传到查询层。

---

上一篇: [项目与 CRUD 脚手架](./scaffolding.md) | 下一篇: [配置、Bootstrap 与 ORM](./configuration.md)
