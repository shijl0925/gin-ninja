# 文件传输与 OpenAPI 控制

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## Multipart 文件上传与下载

### 单文件上传

使用 `file:"..."` 绑定 `*ninja.UploadedFile`：

```go
type UploadSingleInput struct {
    Title string              `form:"title" binding:"required"`
    File  *ninja.UploadedFile `file:"file"  binding:"required"`
}

type UploadDemoOutput struct {
    Title     string   `json:"title,omitempty"`
    Category  string   `json:"category,omitempty"`
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
    }, nil
}

ninja.Post(router, "/upload-single", uploadSingle,
    ninja.Summary("Single file upload"),
    ninja.Description("Demonstrates multipart form-data binding with one file and extra form fields."),
)
```

`UploadedFile` 包装了 `multipart.FileHeader`，并暴露：

- `in.File.Filename`
- `in.File.Size`
- `in.File.Open()`
- `in.File.Bytes()`

### 多文件上传

使用 `[]*ninja.UploadedFile` 绑定重复的 multipart 字段：

```go
type UploadManyInput struct {
    Category string                `form:"category" binding:"required"`
    Files    []*ninja.UploadedFile `file:"files"    binding:"required"`
}

func uploadMany(ctx *ninja.Context, in *UploadManyInput) (*UploadDemoOutput, error) {
    names := make([]string, 0, len(in.Files))
    for _, file := range in.Files {
        names = append(names, file.Filename)
    }
    return &UploadDemoOutput{
        Category:  in.Category,
        FileCount: len(in.Files),
        Names:     names,
    }, nil
}
```

### 混合表单与文件绑定

`form:"..."` 和 `file:"..."` 可以混在同一个输入结构体中。当请求使用 `multipart/form-data` 时，gin-ninja 会同时绑定普通表单字段和上传文件，并自动生成匹配的 OpenAPI request body。

### 文件下载响应

当处理器需要写出二进制响应而不是 JSON 时，返回 `*ninja.Download`：

```go
func download(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    return ninja.NewDownload(
        "report.txt",
        "text/plain; charset=utf-8",
        []byte("hello from gin-ninja\n"),
    ), nil
}

func downloadReader(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    body := strings.NewReader("streamed content\n")
    return ninja.NewDownloadReader(
        "stream.txt",
        "text/plain; charset=utf-8",
        int64(body.Len()),
        body,
    ), nil
}
```

可用辅助方法：

- `ninja.NewDownload(filename, contentType, data)` – 基于字节切片的下载
- `ninja.NewDownloadReader(filename, contentType, size, reader)` – 基于 reader 的下载
- `Download.Inline = true` – 将 `Content-Disposition` 从 `attachment` 切换为 `inline`
- `Download.Headers` – 添加自定义响应头

OpenAPI 会把上传输入描述为 `multipart/form-data`，把 `*ninja.Download` 响应描述为二进制 `application/octet-stream`。

### 示例路由

完整示例应用包含可直接运行的路由：

- `POST /api/v1/examples/upload-single`
- `POST /api/v1/examples/upload-many`
- `GET /api/v1/examples/download`
- `GET /api/v1/examples/download-reader`

---

## OpenAPI 操作级控制

### 文档元信息与服务器地址

通过 `ninja.Config` 配置标准 OpenAPI info 字段和服务器地址：

```go
api := ninja.New(ninja.Config{
    Title:          "My API",
    Version:        "1.0.0",
    Description:    "Public API documentation.",
    TermsOfService: "https://example.com/terms",
    Contact: &ninja.Contact{
        Name:  "Support",
        URL:   "https://example.com/support",
        Email: "support@example.com",
    },
    LicenseInfo: &ninja.LicenseInfo{
        Name: "MIT",
        URL:  "https://opensource.org/license/mit",
    },
    Servers: []ninja.Server{
        {URL: "https://api.example.com", Description: "Production"},
        {URL: "https://staging-api.example.com", Description: "Staging"},
    },
})
```

这些配置会输出为标准 OpenAPI `info.termsOfService`、`info.contact`、`info.license` 以及根级 `servers` 字段，供 Swagger UI 和 ReDoc 使用。

### 操作级控制

```go
users := ninja.NewRouter(
    "/users",
    ninja.WithTags("Users"),
    ninja.WithTagDescription("Users", "User management endpoints"),
)

type SessionInput struct {
    Session string `cookie:"session" binding:"required" default:"guest"`
}

type SessionOutput struct {
    Session string `json:"session"`
}

ninja.Get(router, "/session", getSession,
    ninja.Response(401, "Unauthorized", nil),
    ninja.Response(404, "Session not found", &SessionOutput{}),
)

ninja.Get(router, "/internal/health", healthz,
    ninja.ExcludeFromDocs(),
)

ninja.Get(users, "/", listUsers,
    ninja.Timeout(2*time.Second),
    ninja.RateLimit(20, 40),
    ninja.PaginatedResponse[UserOut](200, "Paginated users"),
)
```

使用 `Response(...)` / `PaginatedResponse[...]` 声明非默认 OpenAPI 响应，使用 `ExcludeFromDocs()` 隐藏内部端点，使用 `Timeout(...)` 设置基于 context 的操作级超时，使用 `RateLimit(...)` 设置操作级限流。`Timeout(...)` 是协作式的：框架会提前返回 408，但处理器仍需要主动响应 context 取消。

---

上一篇: [数据、绑定与响应](./data-and-responses.md) | 下一篇: [高级功能](./advanced-features.md)
