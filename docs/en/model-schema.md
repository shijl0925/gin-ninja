# ModelSchema Advanced Usage

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/model-schema.md)

`ModelSchema` is an optional advanced feature. It can derive response schemas from GORM models, prune fields at runtime, reuse field-selection rules, and help generate relation preload paths.

Beginners and public business APIs do not need it at first. The default recommendation is still an explicit DTO:

```go
type UserOut struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

Explicit DTOs are easier to read, more stable, and usually better for public APIs. `ModelSchema` is useful for internal admin tools, repetitive CRUD, generated code, quick model field pruning, or cases where you intentionally want response schemas to follow model rules.

## When To Use It

| Case | Recommendation |
| --- | --- |
| Public business API | Prefer explicit DTOs |
| Beginner code or a few endpoints | Prefer explicit DTOs |
| Internal admin UI | `ModelSchema` can fit |
| Many repetitive CRUD modules | `ModelSchema` or generated code can fit |
| Need relation-depth-driven preload paths | Use `Depth` + preload helpers |
| Need a very stable public contract | Prefer explicit DTOs |

## Core Concepts

The main APIs live in the root package:

- `ModelSchema[T]`: wraps a model and serializes it with field rules.
- `ModelSchemaOf[T]()`: creates a reusable descriptor.
- `ModelReadSchemaOf[T]()`, `ModelListSchemaOf[T]()`, `ModelDetailSchemaOf[T]()` are response-side convenience descriptors.
- `ModelCreateSchemaOf[T]()`, `ModelUpdateSchemaOf[T]()` are input-side convenience descriptors.
- `ResponseModel[T]()`, `ResponseSchema(...)`, `PaginatedSchema(...)`: connect ModelSchema to operation responses.

These APIs affect both runtime JSON output and OpenAPI schemas.

## Embedded Response Schema

The most common shape is a schema struct that embeds `ninja.ModelSchema[Model]`:

```go
type User struct {
    ID       uint   `json:"id" gorm:"primaryKey"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password" ninja:"write_only"`
}

type UserOut struct {
    ninja.ModelSchema[User] `fields:"id,name,email"`
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*User, error) {
    var user User
    if err := orm.WithContext(ctx.Context).First(&user, in.UserID).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

ninja.Get(router, "/:id", getUser,
    ninja.ResponseModel[UserOut](),
)
```

The handler returns `*User`, but `ResponseModel[UserOut]()` makes gin-ninja bind the `User` to `UserOut` before writing the response. Only `id`, `name`, and `email` are serialized.

You can also exclude fields:

```go
type UserOut struct {
    ninja.ModelSchema[User] `exclude:"password"`
}
```

Field names may be Go field names or JSON field names. For public APIs, prefer explicit DTOs unless the model-derived behavior is intentional.

## Descriptor Style

If you do not want a schema struct, use a descriptor:

```go
userSchema := ninja.ModelReadSchemaOf[User]().
    Fields("id", "name", "email").
    ComponentName("UserOut")

ninja.Get(router, "/:id", getUser,
    ninja.ResponseSchema(userSchema),
)
```

`ResponseSchema` uses the same descriptor for:

- Runtime serialization of returned `User` values.
- OpenAPI schema generation.
- Required-field validation when selected response fields use `binding:"required"`.

For paginated responses, use `PaginatedSchema`:

```go
ninja.Get(router, "/", listUsers,
    ninja.PaginatedSchema(userSchema),
)
```

For cursor pagination, use `CursorPaginatedSchema(userSchema)`.

## Temporary Wrapping

For one-off field pruning, wrap the model directly:

```go
func getUser(ctx *ninja.Context, in *GetUserInput) (*ninja.ModelSchema[User], error) {
    user, err := loadUser(ctx, in.UserID)
    if err != nil {
        return nil, err
    }
    return ninja.NewModelSchema(user, ninja.Fields("id", "name", "email")), nil
}
```

This is useful for internal tools or temporary code. For stable public APIs, define a DTO or a dedicated schema type.

## Field Modes

`ModelSchema` supports several field-selection modes:

| Mode | API | Purpose |
| --- | --- | --- |
| `read` | `ModelReadSchemaOf[T]()` / `.Read()` | General responses |
| `list` | `ModelListSchemaOf[T]()` / `.List()` | List responses; prefers scalar fields |
| `detail` | `ModelDetailSchemaOf[T]()` / `.Detail()` | Detail responses |
| `create` | `ModelCreateSchemaOf[T]()` / `.Create()` | Create input |
| `update` | `ModelUpdateSchemaOf[T]()` / `.Update()` | Update input |

gin-ninja reads field tags to infer read/write behavior:

```go
type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Email     string    `json:"email"`
    Password  string    `json:"password" ninja:"write_only"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

- `ninja:"write_only"`: available for writes, omitted from read responses.
- `ninja:"create_only"`: only included in create mode.
- `ninja:"update_only"`: only included in update mode.
- `gorm:"primaryKey"`, `gorm:"->"`, `gorm:"<-:false"`, `autoCreateTime`, and `autoUpdateTime` are treated as read-only for create/update input.

These rules are helpful when reusing model tags, but public APIs should not become accidentally coupled to database structure.

## Input Binding

`BindInput` copies an already-bound and validated input struct into a model:

```go
type CreateUserInput struct {
    Name     string `json:"name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
    user, err := ninja.ModelCreateSchemaOf[User]().
        Fields("name", "email", "password").
        BindInput(in)
    if err != nil {
        return nil, err
    }

    if err := orm.WithContext(ctx.Context).Create(user).Error; err != nil {
        return nil, err
    }
    out := toUserOut(*user)
    return &out, nil
}
```

For updates, use `ApplyInput`:

```go
type UpdateUserInput struct {
    UserID uint `path:"id" json:"-" binding:"required"`

    Name  *string `json:"name"`
    Email *string `json:"email" binding:"omitempty,email"`
    Age   *int    `json:"age" binding:"omitempty,min=0,max=150"`
}

func updateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
    db := orm.WithContext(ctx.Context)

    var user User
    if err := db.First(&user, in.UserID).Error; err != nil {
        return nil, err
    }

    err := ninja.ModelUpdateSchemaOf[User]().
        Fields("name", "email", "age").
        ApplyInput(&user, in)
    if err != nil {
        return nil, err
    }

    if err := db.Save(&user).Error; err != nil {
        return nil, err
    }
    out := toUserOut(user)
    return &out, nil
}
```

`ApplyInput` skips nil pointer fields, which makes it fit PATCH-style partial updates. Complex updates, relation synchronization, audit fields, and business rules should still be written explicitly.

## Relation Depth And Preloads

`Depth(n)` serializes nested models with the same schema mode:

```go
type Role struct {
    ID   uint   `json:"id"`
    Name string `json:"name"`
}

type User struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Roles []Role `json:"roles"`
}

type UserDetailOut struct {
    ninja.ModelSchema[User] `fields:"id,name,roles" mode:"read" depth:"1"`
}
```

To avoid returning relation fields without loading them, use the ORM preload helpers:

```go
func getUser(ctx *ninja.Context, in *GetUserInput) (*User, error) {
    db := orm.ApplyResponseModelPreloads[UserDetailOut](
        orm.WithContext(ctx.Context),
    )

    var user User
    if err := db.First(&user, in.UserID).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

ninja.Get(router, "/:id", getUser,
    ninja.ResponseModel[UserDetailOut](),
)
```

Descriptor style works too:

```go
detailSchema := ninja.ModelDetailSchemaOf[User]().
    Fields("id", "name", "roles").
    Depth(1).
    ComponentName("UserDetailOut")

db := orm.ApplyModelSchemaPreloads(orm.WithContext(ctx.Context), detailSchema)

ninja.Get(router, "/:id", getUser,
    ninja.ResponseSchema(detailSchema),
)
```

Keep depth shallow. Deeper relations increase query cost and response size; public APIs should expose only the relations they really need.

## DTOs Versus ModelSchema

`ModelSchema` is not a replacement for DTOs. It is an advanced reuse tool:

| Question | DTO | ModelSchema |
| --- | --- | --- |
| Beginner readability | High | Lower |
| Public API stability | Strong | Coupled to model changes |
| Full field control | Strong | Medium; depends on rules |
| Repetitive CRUD speed | Medium | High |
| Internal admin / generated code | Works | Very good fit |
| Relation preload helper | Manual | Built in |

Practical guidance:

- Public business endpoints: start with DTOs.
- Internal management endpoints: consider `ModelSchema`.
- Many repetitive CRUD modules: use `ModelSchema` or the CLI generator as a starting point.
- Once field rules become hard to reason about, an explicit DTO is usually clearer.

Previous: [CRUD Tutorial: Users](./crud-tutorial.md) | Next: [Routers and Operations](./routers-and-operations.md)
