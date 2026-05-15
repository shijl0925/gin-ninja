# File Transfer and OpenAPI Controls

## Multipart File Upload & Download

### Single-file upload

Use `file:"..."` with `*ninja.UploadedFile`:

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

`UploadedFile` wraps `multipart.FileHeader` and exposes:

- `in.File.Filename`
- `in.File.Size`
- `in.File.Open()`
- `in.File.Bytes()`

### Multi-file upload

Use `[]*ninja.UploadedFile` for repeated multipart fields:

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

### Mixed form + file binding

`form:"..."` and `file:"..."` can be mixed in the same input struct. When the request uses `multipart/form-data`, gin-ninja binds regular form fields and uploaded files together and generates the matching OpenAPI request body automatically.

### File download responses

Return `*ninja.Download` when the handler should write a binary response instead of JSON:

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

Available helpers:

- `ninja.NewDownload(filename, contentType, data)` – byte-slice backed download
- `ninja.NewDownloadReader(filename, contentType, size, reader)` – reader-backed download
- `Download.Inline = true` – switch `Content-Disposition` from `attachment` to `inline`
- `Download.Headers` – add custom response headers

OpenAPI will describe upload inputs as `multipart/form-data`, and `*ninja.Download` responses as binary `application/octet-stream`.

### Example routes

The full example app includes ready-to-run routes:

- `POST /api/v1/examples/upload-single`
- `POST /api/v1/examples/upload-many`
- `GET /api/v1/examples/download`
- `GET /api/v1/examples/download-reader`

---

## OpenAPI Operation Controls

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

Use `Response(...)` / `PaginatedResponse[...]` to document non-default OpenAPI responses, `ExcludeFromDocs()` for internal endpoints, `Timeout(...)` for context-based per-operation deadlines, and `RateLimit(...)` for per-operation throttling. `Timeout(...)` is cooperative: the framework returns 408 early, but handlers still need to honor context cancellation.

---
