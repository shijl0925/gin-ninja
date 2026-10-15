# Query Schemas: Pagination, Filtering, and Ordering

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/query-schemas.md)

List endpoints usually need more than a `page` parameter. Real APIs combine pagination, search, filtering, ordering, and defaults. gin-ninja recommends putting those query capabilities in one input struct: that struct is the endpoint's query schema.

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search" description:"Filter by name or email"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
    Sort    string `query:"sort" order:"id|name|email|age|created_at"`
}
```

There are three layers here:

- `query:"..."` binds values from the URL query string.
- `pagination.PageInput` provides standard page-number pagination fields and helpers.
- `filter:"..."` and `order:"..."` describe safe database query capabilities.

## What A Query Schema Is

In gin-ninja, a schema is not a separate DSL. It is a plain Go struct with tags. It serves three purposes at once:

| Goal | Shape | Result |
| --- | --- | --- |
| Request binding | `query:"search"` | Binds `?search=alice` |
| Validation | `binding:"omitempty,min=1"` | Uses validator rules |
| Query declaration | `filter:"is_admin,eq"` / `order:"name|created_at"` | Builds safe GORM/gormx query clauses |

Prefer one input struct per list endpoint instead of a large reusable "universal query" struct. That keeps Swagger/OpenAPI, handler code, and database query capabilities aligned.

## Page Pagination

`pagination.PageInput` is the common page-number pagination input:

```go
type ListUsersInput struct {
    pagination.PageInput
}
```

It contains:

| Field | Query parameter | Default/limit |
| --- | --- | --- |
| `Page` | `page` | default `1`, minimum `1` |
| `Size` | `size` | default `20`, maximum `100` |

Common helpers:

| Method | Effect |
| --- | --- |
| `GetPage()` | Returns the effective page number |
| `GetSize()` | Returns the effective page size |
| `Offset()` | Computes SQL OFFSET |
| `Limit()` | Returns SQL LIMIT |

Responses usually use `pagination.Page[T]`:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    db := orm.WithContext(ctx.Context).Model(&User{})

    var total int64
    if err := db.Count(&total).Error; err != nil {
        return nil, err
    }

    var users []User
    if err := db.
        Limit(in.Limit()).
        Offset(in.Offset()).
        Find(&users).Error; err != nil {
        return nil, err
    }

    return pagination.NewPage(toUserOuts(users), total, in.PageInput), nil
}

ninja.Get(router, "/", listUsers,
    ninja.Summary("List users"),
    ninja.Paginated[UserOut](),
)
```

Response shape:

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "size": 20,
  "pages": 0
}
```

`ninja.Paginated[UserOut]()` documents the success response as a pagination envelope whose items use the `UserOut` schema.

## Cursor Pagination

Cursor pagination fits large datasets, timelines, and message feeds. Use `pagination.CursorPagination` for input:

```go
type ListEventsInput struct {
    pagination.CursorPagination
}
```

It contains:

| Field | Query parameter | Meaning |
| --- | --- | --- |
| `Cursor` | `cursor` | Opaque cursor returned by the previous page |
| `Size` | `size` | default `20`, maximum `100` |

Return `pagination.CursorPage[T]`:

```go
func listEvents(ctx *ninja.Context, in *ListEventsInput) (*pagination.CursorPage[EventOut], error) {
    items, nextCursor, err := service.ListAfterCursor(
        ctx.Request.Context(),
        in.GetCursor(),
        in.GetSize(),
    )
    if err != nil {
        return nil, err
    }
    return pagination.NewCursorPage(items, in.CursorPagination, nextCursor), nil
}

ninja.Get(router, "/events", listEvents,
    ninja.Summary("List events"),
    ninja.CursorPaginated[EventOut](),
)
```

The cursor value is opaque to the framework. You can encode stable ordering values, such as `created_at + id`, and let the service/repository decode them.

When using GORM and a cursor based on one trusted column, `orm.SelectCursorPage` can build the single-column keyset query for you:

```go
func listEvents(ctx *ninja.Context, in *ListEventsInput) (*pagination.CursorPage[EventOut], error) {
    events, nextCursor, err := orm.SelectCursorPage[Event, int64](
        orm.WithContext(ctx.Context).Model(&Event{}),
        in.CursorPagination,
        "id",
        func(cursor string) (int64, error) {
            return strconv.ParseInt(cursor, 10, 64)
        },
        func(event Event) string {
            return strconv.FormatInt(event.ID, 10)
        },
        orm.CursorPageDesc(),
    )
    if err != nil {
        return nil, err
    }
    return pagination.NewCursorPage(toEventOuts(events), in.CursorPagination, nextCursor), nil
}
```

`cursorColumn` must be a server-controlled trusted column name; do not pass client input directly into it. For stable multi-column ordering, keep the custom query and cursor encoding in your repository.

## Filtering

Declare filters with `filter:"column,op"`. Empty fields do not generate filter clauses.

```go
type ListUsersInput struct {
    pagination.PageInput
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq"`
    AgeMin  int    `query:"age_min"  filter:"age,ge"`
    IDs     []uint `query:"ids"      filter:"id,in"`
}
```

Supported operators:

| Operator | Meaning |
| --- | --- |
| `eq` | equals |
| `ne` | not equals |
| `gt` | greater than |
| `ge` | greater than or equal |
| `lt` | less than |
| `le` | less than or equal |
| `like` | LIKE match |
| `in` | IN list |

Use `|` for OR-based multi-field search:

```go
type ListUsersInput struct {
    pagination.PageInput
    Search string `query:"search" filter:"name|email,like"`
}
```

This generates a condition like `name LIKE ? OR email LIKE ?`.

## Custom Filter Expressions

When a filter cannot be expressed with one tag, implement `FilterExpression()` on the input. `filter.ApplyDB` and `filter.BuildOptions` append that expression automatically:

```go
type ListUsersInput struct {
    pagination.PageInput
    Search string `query:"search"`
}

func (in *ListUsersInput) FilterExpression() clause.Expression {
    if in == nil || in.Search == "" {
        return nil
    }
    like := "%" + in.Search + "%"
    return clause.Or(
        clause.Like{Column: clause.Column{Name: "name"}, Value: like},
        clause.Like{Column: clause.Column{Name: "email"}, Value: like},
    )
}
```

Custom expressions are useful for complex search, combined conditions, and tenant/permission filters.

## Ordering

Declare ordering with an `order:"..."` allowlist. Clients can only order by fields in that allowlist:

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `query:"sort" order:"id|name|email|age|created_at"`
}
```

Request examples:

```text
?sort=name
?sort=-created_at
?sort=name,-created_at
?sort=+age
```

Rules:

- `name` means ascending.
- `-created_at` means descending.
- `+age` explicitly means ascending.
- `name,-created_at` supports multiple sort fields.
- Unsupported fields return an error and are never passed into SQL.

If the public parameter name differs from the database column name, use an alias:

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `query:"sort" order:"created:created_at|display_name:name"`
}
```

The client can call `?sort=-created`, and the database uses `created_at DESC`.

## Native GORM

Native GORM uses `filter.ApplyDB` and `order.ApplyDB`:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    db := orm.WithContext(ctx.Context).Model(&User{})

    db, err := filter.ApplyDB(db, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    countDB := db.Session(&gorm.Session{})
    var total int64
    if err := countDB.Count(&total).Error; err != nil {
        return nil, err
    }

    db, err = order.ApplyDB(db, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    var users []User
    if err := db.
        Limit(in.Limit()).
        Offset(in.Offset()).
        Find(&users).Error; err != nil {
        return nil, err
    }

    return pagination.NewPage(toUserOuts(users), total, in.PageInput), nil
}
```

In real projects, the repository/service often owns query assembly, while the handler only reads input, calls the service, and returns DTOs.

## gormx Repository

gormx repositories use `filter.BuildOptions` to create `gormx.DBOption` values and `order.ApplyOrder` to apply ordering to `gormx.Query`:

```go
func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    db := orm.WithContext(ctx.Context)
    repo := NewUserRepo()

    query, _ := gormx.NewQuery[User]()

    filterOpts, err := filter.BuildOptions(in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }
    if err := order.ApplyOrder(query, in); err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    opts := append([]gormx.DBOption{gormx.UseDB(db)}, filterOpts...)
    opts = append(opts, query.ToOptions()...)

    users, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
    if err != nil {
        return nil, err
    }

    return pagination.NewPage(toUserOuts(users), total, in.PageInput), nil
}
```

This style keeps filtering, ordering, and pagination near the repository layer and leaves handlers thin.

## Complete Users Example

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search" description:"Filter by name or email"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
    Sort    string `query:"sort" order:"id|name|email|age|is_admin|created_at"`
}

func (in *ListUsersInput) FilterExpression() clause.Expression {
    if in == nil || in.Search == "" {
        return nil
    }
    like := "%" + in.Search + "%"
    return clause.Or(
        clause.Like{Column: clause.Column{Name: "name"}, Value: like},
        clause.Like{Column: clause.Column{Name: "email"}, Value: like},
    )
}

ninja.Get(router, "/", listUsers,
    ninja.Summary("List users"),
    ninja.Paginated[UserOut](),
)
```

Supported requests:

```text
GET /users/?page=1&size=20
GET /users/?search=alice
GET /users/?is_admin=true
GET /users/?sort=-created_at
GET /users/?search=alice&is_admin=false&sort=name,-created_at
```

## Design Tips

| Tip | Reason |
| --- | --- |
| Define one input per list endpoint | OpenAPI is more accurate and handlers are easier to read |
| Always use an `order:"..."` allowlist | User input should never become raw SQL |
| Prefer database column names in filter tags | Generated queries are easier to inspect |
| Use aliases only when public names differ from database columns | Avoid leaking internal names unnecessarily |
| Use `filter:"name|email,like"` for simple search | Short code and clear OpenAPI |
| Use `FilterExpression()` for complex search | Keeps tags readable |
| Pair list responses with `ninja.Paginated[T]()` | Docs and runtime response shape stay aligned |
| Use cursor pagination for large streams | Avoid deep OFFSET costs |

Previous: [Request Body and Form Data](./request-body-and-forms.md) | Next: [Responses, Errors, and Schemas](./data-and-responses.md)
