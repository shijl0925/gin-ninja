# Request Body and Form Data

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/request-body-and-forms.md)

gin-ninja uses one input struct to describe request bodies, path parameters, query parameters, headers, cookies, and files. For request bodies, the common content types are:

| Content type | Struct tag | Common use |
| --- | --- | --- |
| `application/json` | `json:"..."` | Create or update API resources |
| `application/x-www-form-urlencoded` | `form:"..."` | HTML forms and OAuth-style submissions |
| `multipart/form-data` | `form:"..."` + `file:"..."` | File uploads with normal fields |

The framework chooses binding behavior from the HTTP method and `Content-Type`. `POST`, `PUT`, and `PATCH` bind request bodies. `GET` and `DELETE` usually use path/query/header/cookie only.

## JSON Request Body

Use `json:"..."` for JSON bodies:

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

Request:

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

Rules such as `binding:"required,email"` come from `go-playground/validator`. When validation fails, the handler does not run and the framework returns the standard error response.

## Mix Path And JSON Body

Update endpoints often need both a path parameter and a JSON body:

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

`json:"-"` means `UserID` is not part of the JSON body and should only be read from `/:id`. gin-ninja binds path/query/header/cookie/form/file values first, then JSON body values, and restores non-body fields after JSON decoding.

Request:

```http
PATCH /users/42
Content-Type: application/json

{
  "name": "Alice Zhang"
}
```

Use pointer fields when you need to distinguish "not provided" from "provided as zero value":

```go
type PatchUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`

    Name  *string `json:"name"  binding:"omitempty"`
    Email *string `json:"email" binding:"omitempty,email"`
    Age   *int    `json:"age"   binding:"omitempty,min=0,max=150"`
}
```

That is clearer than guessing updates from empty strings or `0`.

## JSON Body Limit

JSON bodies are limited to `32 MiB` by default. Oversized bodies return `413 Request Entity Too Large` with the message `request body too large`.

For large data, prefer file uploads, direct-to-object-storage uploads, or streaming designs instead of JSON bodies.

## Form URL Encoded

Use `form:"..."` for `application/x-www-form-urlencoded` bodies:

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

Request:

```http
POST /auth/login
Content-Type: application/x-www-form-urlencoded

email=alice%40example.com&password=password123
```

`form:"..."` means form request body. It does not mean query parameters. Use `query:"..."` for query strings.

## Form Defaults

`default:"..."` works on `form` fields. If the client does not submit the field, gin-ninja fills the default:

```go
type SearchFormInput struct {
    Scope string `form:"scope" default:"public"`
    Limit int    `form:"limit" default:"20" binding:"omitempty,min=1,max=100"`
}
```

Defaults also work for `query`, `header`, and `cookie` fields. They do not apply to JSON body fields.

## Header And Cookie Binding

Use `header:"..."` for request headers and `cookie:"..."` for cookies. They can live in the same input struct as path, query, and body fields:

```go
type TraceInput struct {
    TraceID string `header:"X-Trace-ID" default:"generated-by-client"`
    Locale  string `cookie:"locale"     default:"en"`
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

In tests, use `ninjatest.Header(...)` and `ninjatest.Cookie(...)`:

```go
resp := client.Get("/trace",
    ninjatest.Header("X-Trace-ID", "trace-1"),
    ninjatest.Cookie(&http.Cookie{Name: "locale", Value: "en"}),
)
```

Header and cookie binding are useful for request metadata such as trace IDs, tenant IDs, feature flags, and language preferences. Prefer middleware for authentication; avoid parsing `Authorization` by hand inside business handlers.

## Multipart Form

Use `multipart/form-data` for requests that include files. Normal fields still use `form:"..."`; file fields use `file:"..."`. For full upload, download, and streaming guidance, read [File Transfer: Uploads and Downloads](./file-transfer.md).

Basic example:

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

Request:

```bash
curl -X POST http://localhost:8080/uploads/single \
  -F "title=Avatar" \
  -F "file=@avatar.png"
```

Repeated multipart file fields can bind to `[]*ninja.UploadedFile`. `UploadedFile.Bytes()` is convenient for small files, and `UploadedFile.Open()` is better for large streamed files. See [File Transfer: Uploads and Downloads](./file-transfer.md) for more examples.

## How Content-Type Affects Binding

For body methods, gin-ninja selects the body binder from the request `Content-Type`:

| Content-Type | Binding | OpenAPI requestBody |
| --- | --- | --- |
| `multipart/form-data` | `form:"..."` + `file:"..."` | `multipart/form-data` |
| `application/x-www-form-urlencoded` | `form:"..."` | `application/x-www-form-urlencoded` |
| other body requests | `json:"..."` | `application/json` |

Do not expect JSON body and form body in the same request. Send `application/json` for JSON, or the appropriate form content type for forms and files.

## Test Body And Form

`ninjatest.TestClient` automatically encodes structs, maps, slices, and scalar values as JSON:

```go
resp := client.Post("/users/", CreateUserInput{
    Name:  "alice",
    Email: "alice@example.com",
    Age:   30,
})
```

Form request:

```go
form := url.Values{}
form.Set("email", "alice@example.com")
form.Set("password", "password123")

resp := client.Post("/auth/login", form)
```

Multipart request:

```go
resp := client.Post("/uploads/single",
    ninjatest.Multipart(
        url.Values{"title": {"Avatar"}},
        ninjatest.File("file", "avatar.png", "image bytes"),
    ),
)
```

Multiple files:

```go
resp := client.Post("/uploads/many",
    ninjatest.Multipart(
        url.Values{"category": {"docs"}},
        ninjatest.File("files", "a.txt", "A"),
        ninjatest.File("files", "b.txt", "B"),
    ),
)
```

## Common Choices

| Need | Recommended shape |
| --- | --- |
| Create/update a normal resource | JSON body + `json:"..."` |
| Form login or OAuth-style submission | URL encoded form + `form:"..."` |
| Read a request header or cookie | `header:"..."` / `cookie:"..."` |
| Upload one file with a title | multipart + `form:"title"` + `file:"file"`; see [File Transfer](./file-transfer.md) |
| Upload several files | multipart + `[]*ninja.UploadedFile`; see [File Transfer](./file-transfer.md) |
| Distinguish omitted PATCH fields | Pointer fields in JSON body |

Previous: [NinjaAPI and Instance Configuration](./api-and-configuration.md) | Next: [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md)
