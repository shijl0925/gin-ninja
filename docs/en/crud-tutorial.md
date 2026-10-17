# CRUD Tutorial: Users

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/crud-tutorial.md)

This page connects the earlier guides by building a complete CRUD API for a `User` resource with native GORM.

For response behavior only, read [Responses, Errors, and Schemas](./data-and-responses.md). For pagination, filtering, and ordering tags only, read [Query Schemas: Pagination, Filtering, and Ordering](./query-schemas.md).

## Target Endpoints

This tutorial implements five endpoints:

| Method | Path | Handler | Purpose |
| --- | --- | --- | --- |
| `GET` | `/users/` | `ListUsers` | List users with pagination |
| `GET` | `/users/:id` | `GetUser` | Get one user |
| `POST` | `/users/` | `CreateUser` | Create a user |
| `PATCH` | `/users/:id` | `UpdateUser` | Partially update a user |
| `DELETE` | `/users/:id` | `DeleteUser` | Delete a user |

Keep route registration in one small function:

```go
func RegisterUserCRUDRoutes(router *ninja.Router) {
    ninja.Get(router, "/", ListUsers,
        ninja.Summary("List users"),
        ninja.Paginated[UserOut](),
    )
    ninja.Get(router, "/:id", GetUser,
        ninja.Summary("Get user"),
    )
    ninja.Post(router, "/", CreateUser,
        ninja.Summary("Create user"),
        ninja.WithTransaction(),
    )
    ninja.Patch(router, "/:id", UpdateUser,
        ninja.Summary("Patch user"),
        ninja.WithTransaction(),
    )
    ninja.Delete(router, "/:id", DeleteUser,
        ninja.Summary("Delete user"),
        ninja.WithTransaction(),
    )
}
```

When an operation uses `WithTransaction()`, configure transaction handlers on the API and attach the GORM middleware so the request-scoped database is available:

```go
api := ninja.New(ninja.Config{
    Title:               "Users API",
    Prefix:              "/api/v1",
    TransactionHandlers: orm.TransactionHandlers(),
})

api.UseGin(orm.Middleware(db))

router := ninja.NewRouter("/users", ninja.WithTags("Users"))
RegisterUserCRUDRoutes(router)
api.AddRouter(router)
```

## Model And Response DTO

The GORM model represents the database shape. The response DTO represents the public API contract. Public APIs should usually return DTOs instead of raw models:

```go
type User struct {
    gorm.Model
    Name    string `gorm:"column:name;not null"             json:"name"`
    Email   string `gorm:"column:email;uniqueIndex;not null" json:"email"`
    Age     int    `gorm:"column:age"                       json:"age"`
    IsAdmin bool   `gorm:"column:is_admin"                  json:"is_admin"`
}

type UserOut struct {
    ID      uint   `json:"id"`
    Name    string `json:"name"`
    Email   string `json:"email"`
    Age     int    `json:"age"`
    IsAdmin bool   `json:"is_admin"`
}

func toUserOut(u User) UserOut {
    return UserOut{
        ID:      u.ID,
        Name:    u.Name,
        Email:   u.Email,
        Age:     u.Age,
        IsAdmin: u.IsAdmin,
    }
}

func toUserOuts(users []User) []UserOut {
    if users == nil {
        return []UserOut{}
    }
    out := make([]UserOut, 0, len(users))
    for _, u := range users {
        out = append(out, toUserOut(u))
    }
    return out
}
```

## Input Schemas

One input struct can bind path parameters, query parameters, and JSON body fields. List endpoints embed `pagination.PageInput`, then add filter and ordering fields as needed:

```go
type ListUsersInput struct {
    pagination.PageInput

    Sort    string `query:"sort"     order:"id|name|email|age|is_admin|created_at" description:"Sort by allowed fields"`
    Search  string `query:"search"   description:"Filter by name or email"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
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

type GetUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`
}

type CreateUserInput struct {
    Name  string `json:"name"  binding:"required"`
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

type UpdateUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`

    Name  *string `json:"name"  binding:"omitempty"`
    Email *string `json:"email" binding:"omitempty,email"`
    Age   *int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

type DeleteUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`
}
```

`UpdateUserInput` uses pointer fields. `nil` means the client omitted the field, while `""` or `0` means the client explicitly wants to set that value. This is more reliable for PATCH endpoints than plain `string` and `int` fields.

## Native GORM Version

Native GORM handlers are a good starting point for small projects and for cases where you want direct control over the query.

Start with two helpers:

```go
func userDB(ctx *ninja.Context) *gorm.DB {
    return orm.WithContext(ctx.Context)
}

func loadUserByID(db *gorm.DB, id uint) (User, error) {
    var user User
    if err := db.Where("id = ?", id).First(&user).Error; err != nil {
        return User{}, err
    }
    return user, nil
}
```

### List

`filter.ApplyDB` reads `filter` tags and `FilterExpression()`. `order.ApplyDB` reads the `order` tag and keeps sorting inside the allowlist:

```go
func ListUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    db := userDB(ctx)

    query := db.Model(&User{})

    var err error
    query, err = filter.ApplyDB(query, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    countQuery := query.Session(&gorm.Session{})

    query, err = order.ApplyDB(query, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    var total int64
    if err := countQuery.Count(&total).Error; err != nil {
        return nil, err
    }

    var items []User
    if err := query.Limit(in.GetSize()).Offset(in.Offset()).Find(&items).Error; err != nil {
        return nil, err
    }

    return pagination.NewPage(toUserOuts(items), total, in.PageInput), nil
}
```

### Get

Map `gorm.ErrRecordNotFound` to the framework `404` error:

```go
func GetUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
    user, err := loadUserByID(userDB(ctx), in.UserID)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ninja.NotFoundError()
        }
        return nil, err
    }
    out := toUserOut(user)
    return &out, nil
}
```

### Create

`ninja.Post` defaults to `201 Created`. If a database unique constraint fails, convert it to `409 Conflict` when that is useful for your API:

```go
func CreateUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    user := &User{
        Name:  in.Name,
        Email: in.Email,
        Age:   in.Age,
    }

    if err := userDB(ctx).Create(user).Error; err != nil {
        return nil, err
    }

    out := toUserOut(*user)
    return &out, nil
}
```

### Update

PATCH updates only the fields sent by the client:

```go
func UpdateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
    db := userDB(ctx)

    user, err := loadUserByID(db, in.UserID)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ninja.NotFoundError()
        }
        return nil, err
    }

    updateColumns := []string{}
    if in.Name != nil {
        user.Name = *in.Name
        updateColumns = append(updateColumns, "name")
    }
    if in.Email != nil {
        user.Email = *in.Email
        updateColumns = append(updateColumns, "email")
    }
    if in.Age != nil {
        user.Age = *in.Age
        updateColumns = append(updateColumns, "age")
    }

    if len(updateColumns) > 0 {
        if err := db.Model(&User{}).
            Where("id = ?", in.UserID).
            Select(updateColumns).
            Updates(&user).Error; err != nil {
            return nil, err
        }
        user, err = loadUserByID(db, in.UserID)
        if err != nil {
            return nil, err
        }
    }

    out := toUserOut(user)
    return &out, nil
}
```

Use `PUT` when your endpoint replaces the whole resource. Use `PATCH` with pointer fields when it only changes the provided fields.

### Delete

A delete handler returns only `error`. On success, `ninja.Delete` writes `204 No Content`:

```go
func DeleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
    db := userDB(ctx)

    if _, err := loadUserByID(db, in.UserID); err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return ninja.NotFoundError()
        }
        return err
    }

    return db.Model(&User{}).Where("id = ?", in.UserID).Delete(&User{}).Error
}
```

## Repository Style With Native GORM

As CRUD modules grow, moving data access into repositories keeps handlers smaller and more consistent. The users module in `examples/full/app` uses this style without adding another ORM layer.

```go
type IUserRepo interface {
    SelectPage(page, size int, db *gorm.DB) ([]User, int64, error)
    SelectOneByID(id uint, db *gorm.DB) (User, error)
    Insert(item *User, db *gorm.DB) error
    UpdateByID(id uint, updates map[string]interface{}, db *gorm.DB) error
    DeleteByID(id uint, db *gorm.DB) error
}
```

Handlers can still assemble safe filters and ordering with `filter.ApplyDB` and `order.ApplyDB`, then pass the resulting `*gorm.DB` to the repository:

```go
func ListUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
    query := userDB(ctx).Model(&User{})

    query, err := filter.ApplyDB(query, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }
    query, err = order.ApplyDB(query, in)
    if err != nil {
        return nil, ninja.NewError(400, err.Error())
    }

    items, total, err := NewUserRepo().SelectPage(in.GetPage(), in.GetSize(), query)
    if err != nil {
        return nil, err
    }
    return pagination.NewPage(toUserOuts(items), total, in.PageInput), nil
}
```

## Start With The Generator

If the model already exists, let the CLI generate starter CRUD code:

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./app/models.go \
  -output ./app/user_crud_gen.go
```

The generator creates:

- `UserOut`, `ListUsersInput`, `CreateUserInput`, `UpdateUserInput`, and `DeleteUserInput`
- `ListUsers`, `GetUser`, `CreateUser`, `UpdateUser`, and `DeleteUser`
- `RegisterUserCRUDRoutes(router)`

Generated code is a runnable skeleton, not final business logic. Real projects usually add unique-error conversion, permissions, audit logs, events, relation synchronization, and domain-specific validation.

## Choosing A Style

| Case | Recommendation |
| --- | --- |
| Small example or quick validation | Native GORM |
| Many CRUD modules | Native GORM repository/service |
| Need direct control over every query | Native GORM |
| Want consistent pagination, filtering, ordering, and repository style | Native GORM repository/service |
| Starting a new module | `gin-ninja-cli generate crud` |

Previous: [Responses, Errors, and Schemas](./data-and-responses.md) | Next: [ModelSchema Advanced Usage](./model-schema.md)
