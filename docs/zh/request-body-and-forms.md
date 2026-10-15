# Request Body 与 Form Data

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/request-body-and-forms.md)

gin-ninja 用同一个输入结构体描述请求体、路径参数、查询参数、请求头、Cookie 和文件。对于请求体，最常见的三种内容类型是：

| 内容类型 | Struct tag | 常见场景 |
| --- | --- | --- |
| `application/json` | `json:"..."` | API 创建、更新资源 |
| `application/x-www-form-urlencoded` | `form:"..."` | HTML 表单、OAuth 风格提交 |
| `multipart/form-data` | `form:"..."` + `file:"..."` | 上传文件，同时带普通字段 |

框架根据 HTTP 方法和 `Content-Type` 决定如何绑定。`POST`、`PUT`、`PATCH` 会绑定请求体；`GET` 和 `DELETE` 通常只使用 path/query/header/cookie。

## JSON Request Body

JSON body 使用 `json:"..."`：

```go
type CreateUserInput struct {
    Name     string `json:"name"     binding:"required"        description:"Full name"`
    Email    string `json:"email"    binding:"required,email"  description:"Email address"`
    Password string `json:"password" binding:"required,min=8"  description:"Password (min 8 chars)"`
    Age      int    `json:"age"      binding:"omitempty,min=0,max=150"`
}

func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    user, err := service.CreateUser(ctx.Request.Context(), in.Name, in.Email, in.Password, in.Age)
    if err != nil {
        return nil, err
    }
    out := toUserOut(user)
    return &out, nil
}

ninja.Post(router, "/", createUser,
    ninja.Summary("Create user"),
    ninja.WithTransaction(),
)
```

请求：

```http
POST /users/
Content-Type: application/json

{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "password123",
  "age": 30
}
```

`binding:"required,email"` 等规则来自 `go-playground/validator`。校验失败时 handler 不会执行，框架会直接返回统一错误响应。

## 混合 Path 与 JSON Body

更新接口经常同时需要 path 参数和 JSON body：

```go
type UpdateUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`

    Name  string `json:"name"  binding:"omitempty"`
    Email string `json:"email" binding:"omitempty,email"`
    Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

func updateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
    return service.UpdateUser(ctx.Request.Context(), in.UserID, in)
}

ninja.Patch(router, "/:id", updateUser,
    ninja.Summary("Update user"),
    ninja.WithTransaction(),
)
```

`json:"-"` 表示 `UserID` 不属于 JSON body，只从 `/:id` 读取。框架会先绑定 path/query/header/cookie/form/file，再绑定 JSON body，并恢复非 body 字段，所以 JSON 解码不会把 `UserID` 清掉。

请求：

```http
PATCH /users/42
Content-Type: application/json

{
  "name": "Alice Zhang"
}
```

如果你要表达“字段未传”和“字段传了零值”的区别，建议用指针字段：

```go
type PatchUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`

    Name  *string `json:"name"  binding:"omitempty"`
    Email *string `json:"email" binding:"omitempty,email"`
    Age   *int    `json:"age"   binding:"omitempty,min=0,max=150"`
}
```

这比用空字符串或 `0` 判断是否更新更清楚。

## JSON Body 限制

JSON body 默认限制为 `32 MiB`。超过限制时，框架返回 `413 Request Entity Too Large`，错误消息为 `request body too large`。

大体积数据不建议放 JSON body，优先使用文件上传、对象存储直传，或流式处理。

## Form URL Encoded

`application/x-www-form-urlencoded` 使用 `form:"..."`：

```go
type LoginFormInput struct {
    Email    string `form:"email"    binding:"required,email"`
    Password string `form:"password" binding:"required"`
}

func loginWithForm(ctx *ninja.Context, in *LoginFormInput) (*LoginOutput, error) {
    return auth.Login(ctx.Request.Context(), in.Email, in.Password)
}

ninja.Post(router, "/login", loginWithForm,
    ninja.Summary("Login with form data"),
)
```

请求：

```http
POST /auth/login
Content-Type: application/x-www-form-urlencoded

email=alice%40example.com&password=password123
```

`form:"..."` 只表示表单请求体，不表示 query 参数。查询参数仍然使用 `query:"..."`。

## Form 默认值

`default:"..."` 可用于 `form` 字段。客户端没有提交该字段时，框架会填入默认值：

```go
type SearchFormInput struct {
    Scope string `form:"scope" default:"public"`
    Limit int    `form:"limit" default:"20" binding:"omitempty,min=1,max=100"`
}
```

默认值也适用于 `query`、`header`、`cookie` 字段，但不适用于 JSON body 字段。

## Header 与 Cookie 绑定

请求头使用 `header:"..."`，Cookie 使用 `cookie:"..."`。它们可以和 path、query、body 字段放在同一个 input struct 里：

```go
type TraceInput struct {
    TraceID string `header:"X-Trace-ID" default:"generated-by-client"`
    Locale  string `cookie:"locale"     default:"zh"`
}

type TraceOutput struct {
    TraceID string `json:"trace_id"`
    Locale  string `json:"locale"`
}

func trace(ctx *ninja.Context, in *TraceInput) (*TraceOutput, error) {
    return &TraceOutput{
        TraceID: in.TraceID,
        Locale:  in.Locale,
    }, nil
}

ninja.Get(router, "/trace", trace,
    ninja.Summary("Read header and cookie values"),
)
```

测试时可以用 `ninjatest.Header(...)` 和 `ninjatest.Cookie(...)`：

```go
resp := client.Get("/trace",
    ninjatest.Header("X-Trace-ID", "trace-1"),
    ninjatest.Cookie(&http.Cookie{Name: "locale", Value: "zh"}),
)
```

Header/Cookie 绑定适合读取追踪 ID、租户 ID、实验开关、语言偏好等请求元数据。认证信息优先使用中间件处理，不建议在业务 handler 里手写解析 `Authorization`。

## Multipart Form

`multipart/form-data` 用于表单里带文件的请求。普通字段继续用 `form:"..."`，文件字段用 `file:"..."`。完整上传、下载和流式处理说明见 [文件传输：上传与下载](./file-transfer.md)。

基础示例：

```go
type UploadSingleInput struct {
    Title string              `form:"title" binding:"required" description:"File title"`
    File  *ninja.UploadedFile `file:"file"  binding:"required" description:"Single uploaded file"`
}

type UploadDemoOutput struct {
    Title     string   `json:"title,omitempty"`
    Filename  string   `json:"filename,omitempty"`
    Size      int64    `json:"size,omitempty"`
    FileCount int      `json:"file_count"`
    Names     []string `json:"names,omitempty"`
}

func uploadSingle(ctx *ninja.Context, in *UploadSingleInput) (*UploadDemoOutput, error) {
    return &UploadDemoOutput{
        Title:     in.Title,
        Filename:  in.File.Filename,
        Size:      in.File.Size,
        FileCount: 1,
        Names:     []string{in.File.Filename},
    }, nil
}

ninja.Post(router, "/uploads/single", uploadSingle,
    ninja.Summary("Upload one file"),
)
```

请求：

```bash
curl -X POST http://localhost:8080/uploads/single \
  -F "title=Avatar" \
  -F "file=@avatar.png"
```

重复的 multipart 文件字段可以绑定到 `[]*ninja.UploadedFile`。`UploadedFile.Bytes()` 适合小文件，`UploadedFile.Open()` 适合大文件流式处理。更多示例见 [文件传输：上传与下载](./file-transfer.md)。

## Content-Type 如何影响绑定

同一个 body method 中，框架按请求 `Content-Type` 选择绑定方式：

| Content-Type | 绑定方式 | OpenAPI requestBody |
| --- | --- | --- |
| `multipart/form-data` | `form:"..."` + `file:"..."` | `multipart/form-data` |
| `application/x-www-form-urlencoded` | `form:"..."` | `application/x-www-form-urlencoded` |
| 其他 body 请求 | `json:"..."` | `application/json` |

不要在同一个请求里同时期待 JSON body 和 form body。需要 JSON 就发 `application/json`；需要表单或文件就发对应的 form Content-Type。

## 测试 Body 与 Form

`ninjatest.TestClient` 会把 struct、map、slice 和标量自动编码成 JSON：

```go
resp := client.Post("/users/", CreateUserInput{
    Name:  "alice",
    Email: "alice@example.com",
    Age:   30,
})
```

表单请求：

```go
form := url.Values{}
form.Set("email", "alice@example.com")
form.Set("password", "password123")

resp := client.Post("/auth/login", form)
```

multipart 请求：

```go
resp := client.Post("/uploads/single",
    ninjatest.Multipart(
        url.Values{"title": {"Avatar"}},
        ninjatest.File("file", "avatar.png", "image bytes"),
    ),
)
```

多文件：

```go
resp := client.Post("/uploads/many",
    ninjatest.Multipart(
        url.Values{"category": {"docs"}},
        ninjatest.File("files", "a.txt", "A"),
        ninjatest.File("files", "b.txt", "B"),
    ),
)
```

## 常见选择

| 需求 | 推荐 |
| --- | --- |
| 创建/更新普通资源 | JSON body + `json:"..."` |
| 表单登录、OAuth 风格提交 | URL encoded form + `form:"..."` |
| 读取请求头或 Cookie | `header:"..."` / `cookie:"..."` |
| 上传一个文件并带标题 | multipart + `form:"title"` + `file:"file"`，更多见 [文件传输](./file-transfer.md) |
| 上传多个文件 | multipart + `[]*ninja.UploadedFile`，更多见 [文件传输](./file-transfer.md) |
| 需要区分 PATCH 字段是否提交 | JSON body 使用指针字段 |

上一篇：[NinjaAPI 与实例配置](./api-and-configuration.md) | 下一篇：[查询 Schema：分页、过滤与排序](./query-schemas.md)
