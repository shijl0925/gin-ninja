# 文件传输：上传与下载

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/file-transfer.md)

普通 JSON 接口只需要输入/输出 struct。文件上传使用 `multipart/form-data`、`form:"..."` 和 `file:"..."`；文件下载返回 `*ninja.Download`。

如果你想先理解 JSON body、URL encoded form 和 multipart 的绑定规则，请先读 [Request Body 与 Form Data](./request-body-and-forms.md)。这一页只关注上传、下载和文件流处理。

## 单文件上传

上传文件时，普通表单字段继续用 `form:"..."`，文件字段用 `file:"..."`：

```go
type UploadAvatarInput struct {
    UserID uint                `path:"id" binding:"required"`
    Title  string              `form:"title"`
    File   *ninja.UploadedFile `file:"file" binding:"required"`
}

type UploadAvatarOutput struct {
    Filename string `json:"filename"`
    Size     int64  `json:"size"`
}

func uploadAvatar(ctx *ninja.Context, in *UploadAvatarInput) (*UploadAvatarOutput, error) {
    data, err := in.File.Bytes()
    if err != nil {
        return nil, err
    }

    // 将 data 保存到对象存储或本地文件系统。
    _ = data

    return &UploadAvatarOutput{
        Filename: in.File.Filename,
        Size:     in.File.Size,
    }, nil
}

ninja.Post(router, "/:id/avatar", uploadAvatar,
    ninja.Summary("Upload avatar"),
)
```

请求示例：

```bash
curl -X POST http://localhost:8080/users/1/avatar \
  -F "title=Avatar" \
  -F "file=@avatar.png"
```

## 多文件上传

重复的 multipart 文件字段可以绑定到 `[]*ninja.UploadedFile`：

```go
type UploadImagesInput struct {
    Album string                `form:"album" binding:"required"`
    Files []*ninja.UploadedFile `file:"files" binding:"required"`
}

type UploadImagesOutput struct {
    Album     string   `json:"album"`
    FileCount int      `json:"file_count"`
    Names     []string `json:"names"`
}

func uploadImages(ctx *ninja.Context, in *UploadImagesInput) (*UploadImagesOutput, error) {
    names := make([]string, 0, len(in.Files))
    for _, file := range in.Files {
        names = append(names, file.Filename)
    }
    return &UploadImagesOutput{
        Album:     in.Album,
        FileCount: len(in.Files),
        Names:     names,
    }, nil
}
```

请求示例：

```bash
curl -X POST http://localhost:8080/albums/summer/images \
  -F "album=summer" \
  -F "files=@a.jpg" \
  -F "files=@b.jpg"
```

## UploadedFile

`ninja.UploadedFile` 包装了 `multipart.FileHeader`：

| 成员 | 用法 |
| --- | --- |
| `Filename` | 原始文件名 |
| `Size` | 文件大小 |
| `Open()` | 打开文件流，适合大文件 |
| `Bytes()` | 一次性读入内存，适合小文件 |

小文件可以直接读取：

```go
data, err := in.File.Bytes()
if err != nil {
    return nil, err
}
```

大文件建议流式处理：

```go
reader, err := in.File.Open()
if err != nil {
    return nil, err
}
defer reader.Close()

// 将 reader 复制到对象存储或本地文件。
```

`Bytes()` 会把整个文件读入内存。生产环境建议优先使用 `Open()`，并配合 [上传限制中间件](./middleware-security.md#上传限制)，避免把过大的请求打进业务逻辑。

## 下载文件

下载小文件或动态生成的字节内容：

```go
func downloadReport(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    return ninja.NewDownload(
        "report.txt",
        "text/plain; charset=utf-8",
        []byte("hello from gin-ninja\n"),
    ), nil
}

ninja.Get(router, "/report", downloadReport,
    ninja.Summary("Download report"),
)
```

下载大文件或流式内容：

```go
func streamReport(ctx *ninja.Context, _ *struct{}) (*ninja.Download, error) {
    body := strings.NewReader("streamed content\n")
    return ninja.NewDownloadReader(
        "report.txt",
        "text/plain; charset=utf-8",
        int64(body.Len()),
        body,
    ), nil
}
```

常用设置：

```go
download := ninja.NewDownload("report.pdf", "application/pdf", data)
download.Inline = true
download.Headers = map[string]string{"X-Report-Version": "v1"}
return download, nil
```

`Inline = true` 会把 `Content-Disposition` 从附件下载切换为浏览器内联预览。

当 handler 返回 `*ninja.Download` 时，OpenAPI 会把成功响应声明为 `application/octet-stream`，schema 为 `type: string`、`format: binary`。实际响应的 `Content-Type` 仍由 `NewDownload` / `NewDownloadReader` 传入的 `contentType` 或框架自动探测结果决定。

## 测试上传

`ninjatest.Multipart` 可以构造 multipart 请求：

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
        url.Values{"album": {"summer"}},
        ninjatest.File("files", "a.jpg", "A"),
        ninjatest.File("files", "b.jpg", "B"),
    ),
)
```

## 选择建议

| 需求 | 推荐 |
| --- | --- |
| 上传一个小文件 | `*ninja.UploadedFile` + `Bytes()` |
| 上传大文件 | `*ninja.UploadedFile` + `Open()` 流式处理 |
| 上传多个文件 | `[]*ninja.UploadedFile` |
| 上传文件并带普通字段 | `file:"..."` + `form:"..."` |
| 返回小文件 | `ninja.NewDownload(...)` |
| 返回流式内容 | `ninja.NewDownloadReader(...)` |
| 浏览器内联预览 | `download.Inline = true` |

上一篇：[中间件与安全](./middleware-security.md) | 下一篇：[使用 TestClient 测试 API](./testing.md)
