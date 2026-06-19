# Data, Binding, and Responses

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/README.md)

## ModelSchema-style Responses

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

func getUser(ctx *ninja.Context, in *struct{}) (*User, error) {
    return &User{
        ID:       1,
        Name:     "alice",
        Email:    "alice@example.com",
        Password: "secret",
    }, nil
}

ninja.Get(router, "/users/:id", getUser, ninja.ResponseModel[UserOut]())
```

`ResponseModel[UserOut]()` binds the handler's returned `User` value to `UserOut` at runtime, then applies response validation and field pruning. `fields:"..."` keeps only the listed serializable fields, while `exclude:"..."` removes sensitive fields from both the JSON response and generated OpenAPI schema.

If you prefer a reusable descriptor instead of a dedicated `UserOut` struct, use `ResponseSchema`:

```go
userSchema := ninja.ModelReadSchemaOf[User]().
    Fields("id", "name", "email").
    ComponentName("UserOut")

func getUser(ctx *ninja.Context, in *struct{}) (*User, error) {
    return &User{ID: 1, Name: "alice", Email: "alice@example.com", Password: "secret"}, nil
}

ninja.Get(router, "/users/:id", getUser, ninja.ResponseSchema(userSchema))
```

`ResponseSchema(...)` prunes fields at runtime and documents the same schema in OpenAPI. For slices, `pagination.Page[T]`, and `pagination.CursorPage[T]`, use `ResponseModel[[]UserOut]()`, `Paginated[UserOut]()` / `PaginatedSchema(...)`, and `CursorPaginated[UserOut]()` / `CursorPaginatedSchema(...)`; paginated `items` are bound and validated item by item.

If you only need ad-hoc filtering without defining a response type or route-level schema, use `ninja.NewModelSchema(model, ninja.Fields(...), ninja.Exclude(...))`. If you explicitly need a schema value, `ninja.BindModelSchema[UserOut](model)` and `ninja.BindModelSchemas[UserOut](models)` are still available.

### Relation Depth and GORM Preloads

`Depth(n)` continues the same model schema mode into nested models, which is useful for detail responses:

```go
type UserDetailOut struct {
    ninja.ModelSchema[User] `mode:"read" depth:"1"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*User, error) {
    db := orm.ApplyResponseModelPreloads[UserDetailOut](orm.WithContext(ctx.Context))

    var user User
    if err := db.First(&user, in.ID).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

ninja.Get(router, "/users/:id", getUser, ninja.ResponseModel[UserDetailOut]())
```

With descriptor-style schemas, inspect preload paths with `descriptor.Preloads()` or apply them directly to GORM with `orm.ApplyModelSchemaPreloads(db, descriptor)`.

---

## Standard Response Envelope

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

// Success: {"code": 200, "message": "success", "data": {...}}
response.Success(c, users)

// Error:   {"code": 404, "message": "not found", "data": null}
response.NotFound(c, "user not found")

// Custom:  {"code": 200, "message": "created", "data": {...}}
response.JSON(c, response.OKWithMessage("created", user))
```

Framework errors use the same root-level envelope:

```json
{"code": 404, "message": "not found", "data": null}
```

---

## Parameter Binding

| Tag          | Source                         | Methods            |
|--------------|--------------------------------|--------------------|
| `path:"x"`   | URL path parameter             | all                |
| `query:"x"`  | URL query string               | all                |
| `form:"x"`   | Form body field                | POST / PUT / PATCH |
| `header:"x"` | Request header                 | all                |
| `cookie:"x"` | Request cookie                 | all                |
| `json:"x"`   | JSON request body              | POST / PUT / PATCH |
| `file:"x"`   | Multipart uploaded file(s)     | POST / PUT / PATCH |

`binding:"..."` uses [go-playground/validator](https://github.com/go-playground/validator).

`default:"..."` applies to `query`, `form`, `header`, and `cookie` fields when the client omits the value.

---

## Declarative Filtering & Safe Sorting

### Declarative filtering

Embed `pagination.PageInput` in a list input struct, then add `filter:"column,op"` to query fields that should become database filters. To match one input field against multiple columns, separate the columns with `|`:

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search"   filter:"name|email,like" description:"Filter by name or email (partial match)"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
}
```

Supported operators:

- `eq`
- `ne`
- `gt`
- `ge`
- `lt`
- `le`
- `like`
- `in`

Apply the declared filters in the handler:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[User], error) {
    query, _ := gormx.NewQuery[User]()

    filterOpts, err := filter.BuildOptions(in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    opts := append(filterOpts, query.ToOptions()...)
    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}

ninja.Get(router, "/users", listUsers, ninja.Paginated[UserOut]())
```

Behavior notes:

- only fields tagged with `filter:"..."` participate in filtering
- zero values are ignored, so omitted query params do not add conditions
- `like` is suitable for contains-style fuzzy matching
- `filter:"name|email,like"` means `(name LIKE ? OR email LIKE ?)`; multi-field declarative filters use OR semantics
- for logic that cannot be represented by tags, the input may implement `FilterExpression() clause.Expression`; the returned expression is applied with `AND` alongside tagged filters by `filter.BuildOptions(...)` and `filter.ApplyDB(...)`
- invalid filter declarations return a 400 error when you surface `filter.BuildOptions(...)` or `filter.Apply(...)` errors

### Safe sorting

Use a `sort` query parameter with an `order:"..."` allowlist. Prefix a field with `-` for descending or `+` for ascending:

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

For paginated handlers, keep using `pagination.PageInput` for page/size and declare `Sort` separately:

```go
import "github.com/shijl0925/gin-ninja/order"

type ListUsersInput struct {
    pagination.PageInput
    Sort   string `query:"sort" order:"id|name|email|age|is_admin|created_at"`
    Search string `query:"search" filter:"name|email,like"`
}

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[User], error) {
    query, _ := gormx.NewQuery[User]()

    if err := order.ApplyOrder(query, in); err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), query.ToOptions()...)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(items, total, in.PageInput), nil
}

ninja.Get(router, "/users", listUsers, ninja.Paginated[UserOut]())
```

### Cursor pagination

Use `pagination.CursorPagination` when list endpoints need stable pagination over
large or frequently changing datasets. The cursor is opaque to gin-ninja; encode
the ordered field values your storage layer needs, then return the next cursor
in `pagination.CursorPage[T]`:

```go
type ListEventsInput struct {
    pagination.CursorPagination
}

func listEvents(ctx *ninja.Context, in *ListEventsInput) (*pagination.CursorPage[Event], error) {
    items, nextCursor, err := repo.SelectAfterCursor(in.GetCursor(), in.GetSize(), "-created_at", "-id")
    if err != nil {
        return nil, err
    }
    return pagination.NewCursorPage(items, in.CursorPagination, nextCursor), nil
}
```

Document the response envelope and runtime item schema with `ninja.CursorPaginated[EventOut]()`.

For simple single-column keyset pagination with GORM, use `orm.SelectCursorPage`:

```go
items, nextCursor, err := orm.SelectCursorPage(
    db,
    in.CursorPagination,
    "id",
    strconv.Atoi,
    func(item Event) string { return strconv.Itoa(item.ID) },
)
if err != nil {
    return nil, err
}
return pagination.NewCursorPage(items, in.CursorPagination, nextCursor), nil
```

If you need a public alias that maps to a different database column, use `alias:column` or `alias=column`:

```go
type ListUsersInput struct {
    Sort string `query:"sort" order:"name|created:created_at"`
}
```

Any sort field outside the allowlist is rejected with an error instead of being passed through to the query layer.

### Example

The full example app uses declarative sorting on paginated users:

- `GET /api/v1/users` → paginated filtering + sorting
- `sort` → validated by `order:"..."` allowlists before reaching the query layer

Try requests like:

- `/api/v1/users?search=ali`
- `/api/v1/users?is_admin=true&sort=-age`

---

Previous: [Middleware and Security](./middleware-security.md) | Next: [File Transfer and OpenAPI Controls](./files-and-openapi.md)
