# ModelSchema 高级用法

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/model-schema.md)

`ModelSchema` 是可选的高级能力。它可以从 GORM model 派生响应 schema、运行时裁剪字段、复用字段选择规则，并辅助生成关系 preload。

新手和公开业务 API 不需要一开始就使用它。默认推荐仍然是显式 DTO：

```go
type UserOut struct {
    ID    uint   `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

显式 DTO 更直观、更稳定，也更适合对外 API。`ModelSchema` 适合内部后台、重复 CRUD、生成器代码、快速裁剪 model 字段，或确实希望响应 schema 跟 model 规则保持一致的场景。

## 什么时候使用

| 场景 | 建议 |
| --- | --- |
| 对外业务 API | 优先显式 DTO |
| 新手入门、少量接口 | 优先显式 DTO |
| 内部 admin / 管理后台 | 可以使用 `ModelSchema` |
| 大量重复 CRUD | 可以使用 `ModelSchema` 或生成器 |
| 需要根据关系深度自动 preload | 可以使用 `Depth` + preload helper |
| 需要完全稳定的公开契约 | 优先显式 DTO |

## 核心概念

`ModelSchema` 的实现入口在框架根包中：

- `ModelSchema[T]`：包装一个 model，并按字段规则序列化。
- `ModelSchemaOf[T]()`：创建可复用 descriptor。
- `ModelReadSchemaOf[T]()`、`ModelListSchemaOf[T]()`、`ModelDetailSchemaOf[T]()`：响应侧的便捷 descriptor。
- `ModelCreateSchemaOf[T]()`、`ModelUpdateSchemaOf[T]()`：输入侧的便捷 descriptor。
- `ResponseModel[T]()`、`ResponseSchema(...)`、`PaginatedSchema(...)`：把 ModelSchema 接到 operation 响应上。

这些 API 会同时影响运行时 JSON 输出和 OpenAPI schema。

## 嵌入式响应 Schema

最常见写法是定义一个 schema struct，并嵌入 `ninja.ModelSchema[Model]`：

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

这里 handler 返回 `*User`，但 `ResponseModel[UserOut]()` 会让框架在写响应前把 `User` 绑定到 `UserOut`，并按 `fields` 只输出 `id`、`name`、`email`。

也可以用 `exclude` 排除字段：

```go
type UserOut struct {
    ninja.ModelSchema[User] `exclude:"password"`
}
```

字段名可以写 Go 字段名，也可以写 JSON 字段名。公开 API 中，仍建议优先写清楚 DTO；这个能力主要用于减少重复 CRUD 中的字段裁剪代码。

## Descriptor 写法

如果不想定义 schema struct，可以使用 descriptor：

```go
userSchema := ninja.ModelReadSchemaOf[User]().
    Fields("id", "name", "email").
    ComponentName("UserOut")

ninja.Get(router, "/:id", getUser,
    ninja.ResponseSchema(userSchema),
)
```

`ResponseSchema` 会使用同一份 descriptor：

- 运行时把 handler 返回的 `User` 裁剪成公开字段。
- OpenAPI 中使用同样的 schema。
- 如果响应字段带 `binding:"required"`，运行时也会校验必填字段。

列表响应可以使用 `PaginatedSchema`：

```go
ninja.Get(router, "/", listUsers,
    ninja.PaginatedSchema(userSchema),
)
```

游标分页使用 `CursorPaginatedSchema(userSchema)`。

## 临时包装

如果只想在某个地方临时裁剪字段，可以直接包装 model：

```go
func getUser(ctx *ninja.Context, in *GetUserInput) (*ninja.ModelSchema[User], error) {
    user, err := loadUser(ctx, in.UserID)
    if err != nil {
        return nil, err
    }
    return ninja.NewModelSchema(user, ninja.Fields("id", "name", "email")), nil
}
```

这种方式适合内部工具或临时代码。正式公开 API 更推荐定义 DTO 或独立 schema 类型。

## 字段模式

`ModelSchema` 支持几种字段选择模式：

| 模式 | API | 用途 |
| --- | --- | --- |
| `read` | `ModelReadSchemaOf[T]()` / `.Read()` | 普通响应 |
| `list` | `ModelListSchemaOf[T]()` / `.List()` | 列表响应，默认偏向标量字段 |
| `detail` | `ModelDetailSchemaOf[T]()` / `.Detail()` | 详情响应 |
| `create` | `ModelCreateSchemaOf[T]()` / `.Create()` | 创建输入 |
| `update` | `ModelUpdateSchemaOf[T]()` / `.Update()` | 更新输入 |

框架会参考字段 tag 来判断读写能力：

```go
type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Email     string    `json:"email"`
    Password  string    `json:"password" ninja:"write_only"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

- `ninja:"write_only"`：写入时可用，读取响应中不输出。
- `ninja:"create_only"`：只用于 create 模式。
- `ninja:"update_only"`：只用于 update 模式。
- `gorm:"primaryKey"`、`gorm:"->"`、`gorm:"<-:false"`、`autoCreateTime`、`autoUpdateTime` 等只读字段不会进入 create/update 输入。

这些规则适合复用 model tag，但不要为了省 DTO 而让公开 API 和数据库结构强耦合。

## 输入绑定

`BindInput` 可以把已经由 gin-ninja 绑定和校验过的 input struct 复制到 model：

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

更新时用 `ApplyInput`：

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

`ApplyInput` 会跳过 nil 指针字段，所以很适合 PATCH 风格的部分更新。复杂更新、关联同步、审计字段等业务逻辑仍建议手写清楚。

## 关系深度与 Preload

`Depth(n)` 会让 schema 把嵌套 model 继续按相同模式序列化：

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

为了避免响应需要关系字段但查询没有 preload，可以让 ORM helper 按 schema 生成 preload：

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

Descriptor 写法也可以：

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

`Depth` 不是越大越好。关系越深，查询和响应越重；公开 API 最好只暴露明确需要的关系。

## 和 DTO 的关系

`ModelSchema` 不是 DTO 的替代品，而是高级复用工具：

| 问题 | DTO | ModelSchema |
| --- | --- | --- |
| 新手理解成本 | 低 | 高 |
| 公开 API 稳定性 | 强 | 依赖 model 变化 |
| 字段完全可控 | 强 | 中等，需要理解规则 |
| 重复 CRUD 开发效率 | 中等 | 高 |
| 内部 admin / 生成器 | 可用 | 很适合 |
| 关系 preload 辅助 | 手写 | 内置 helper |

实践建议：

- 公开业务接口：先写 DTO。
- 内部管理接口：可以考虑 `ModelSchema`。
- CRUD 很多且结构重复：可以用 `ModelSchema` 或 CLI 生成器起步。
- 一旦字段规则变复杂，回到显式 DTO 通常更清晰。

上一篇：[CRUD 教程：Users 增删改查](./crud-tutorial.md) | 下一篇：[Router 与 Operation](./routers-and-operations.md)
