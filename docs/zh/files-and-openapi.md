# 文件传输与 OpenAPI 控制

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 文件上传与下载

### 单文件上传

```go
type UploadSingleInput struct {
    Title string              `form:"title" binding:"required"`
    File  *ninja.UploadedFile `file:"file"  binding:"required"`
}
```

### 多文件上传

```go
type UploadManyInput struct {
    Category string                `form:"category" binding:"required"`
    Files    []*ninja.UploadedFile `file:"files"    binding:"required"`
}
```

### 下载响应

```go
func download(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    return ninja.NewDownload(
        "report.txt",
        "text/plain; charset=utf-8",
        []byte("hello from gin-ninja\n"),
    ), nil
}
```

还支持：

- `ninja.NewDownloadReader(...)`
- `Download.Inline = true`
- `Download.Headers`

## OpenAPI 与操作级控制

```go
ninja.Get(users, "/", listUsers,
    ninja.Timeout(2*time.Second),
    ninja.RateLimit(20, 40),
    ninja.PaginatedResponse[UserOut](200, "Paginated users"),
)

ninja.Get(router, "/internal/health", healthz,
    ninja.ExcludeFromDocs(),
)
```

常用操作选项：

- `Summary(...)`
- `Description(...)`
- `Response(...)`
- `Paginated[...]()` / `PaginatedResponse[...]()`
- `ExcludeFromDocs()`
- `Timeout(...)`
- `RateLimit(...)`
- `Security(...)` / `BearerAuth(authMiddleware)`
- `Cache(...)` / `CacheControl(...)` / `ETag()`

`Timeout(...)` 是协作式超时：框架会提前返回 408 并取消 context，但业务代码仍需主动监听 context 取消并尽快退出。

---

上一篇: [中间件与安全](./middleware-security.md) | 下一篇: [高级功能](./advanced-features.md)
