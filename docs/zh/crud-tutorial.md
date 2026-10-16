# CRUD 教程：Users 增删改查

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/crud-tutorial.md)

这一页把前面几章串起来：用 `User` 资源和原生 GORM 实现一组完整的 CRUD 接口。

如果只想了解响应结构，请看 [响应、错误与 Schema](./data-and-responses.md)。如果只想查分页、过滤和排序的标签，请看 [查询 Schema：分页、过滤与排序](./query-schemas.md)。

## 目标接口

本教程实现下面 5 个接口：

| 方法 | 路径 | Handler | 说明 |
| --- | --- | --- | --- |
| `GET` | `/users/` | `ListUsers` | 分页查询用户 |
| `GET` | `/users/:id` | `GetUser` | 查询单个用户 |
| `POST` | `/users/` | `CreateUser` | 创建用户 |
| `PATCH` | `/users/:id` | `UpdateUser` | 部分更新用户 |
| `DELETE` | `/users/:id` | `DeleteUser` | 删除用户 |

路由注册可以集中放在一个函数里：

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

如果使用 `WithTransaction()`，API 实例需要配置事务处理器，并通过 GORM middleware 把数据库放进请求上下文：

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

## Model 与响应 DTO

GORM model 表示数据库结构，响应 DTO 表示公开 API 契约。对外接口建议返回 DTO，不要直接返回 model：

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

## 输入 Schema

一个 input struct 可以同时绑定 path、query 和 JSON body。列表接口嵌入 `pagination.PageInput`，再按需增加过滤和排序字段：

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

这里的 `UpdateUserInput` 使用指针字段。这样 `nil` 表示“客户端没有传这个字段”，而 `""` 或 `0` 表示“客户端明确要改成空字符串或 0”。这比在 PATCH 接口里使用普通 `string` / `int` 更可靠。

## 原生 GORM 写法

原生 GORM 写法适合从小项目开始，也适合需要直接控制 SQL 细节的场景。

先准备两个 helper：

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

`filter.ApplyDB` 读取 input 上的 `filter` tag 和 `FilterExpression()`，`order.ApplyDB` 读取 `order` tag 并限制可排序字段：

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

没有找到记录时，把 `gorm.ErrRecordNotFound` 转成框架的 `404`：

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

`ninja.Post` 默认成功状态码是 `201 Created`。如果数据库唯一索引报错，可以按项目需要转成 `409 Conflict`：

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

PATCH 更新只修改客户端传入的字段：

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

如果你的接口语义是完整替换资源，可以用 `PUT`；如果是只更新传入字段，推荐用 `PATCH` 和指针字段。

### Delete

删除接口的 handler 只返回 `error`，成功时由 `ninja.Delete` 写出 `204 No Content`：

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

## 原生 GORM Repository 写法

当 CRUD 变多后，把数据访问收敛到 repository 会更好维护。`examples/full/app` 中的 users 模块使用这种风格，但不引入额外 ORM 层。

```go
type IUserRepo interface {
    SelectPage(page, size int, db *gorm.DB) ([]User, int64, error)
    SelectOneByID(id uint, db *gorm.DB) (User, error)
    Insert(item *User, db *gorm.DB) error
    UpdateByID(id uint, updates map[string]interface{}, db *gorm.DB) error
    DeleteByID(id uint, db *gorm.DB) error
}
```

handler 仍然可以使用 `filter.ApplyDB` 和 `order.ApplyDB` 组装安全查询，再把得到的 `*gorm.DB` 交给 repository：

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

## 用生成器起步

如果 model 已经存在，可以先让 CLI 生成 CRUD 起步代码：

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./app/models.go \
  -output ./app/user_crud_gen.go
```

生成器会创建：

- `UserOut`、`ListUsersInput`、`CreateUserInput`、`UpdateUserInput`、`DeleteUserInput`
- `ListUsers`、`GetUser`、`CreateUser`、`UpdateUser`、`DeleteUser`
- `RegisterUserCRUDRoutes(router)`

生成代码是“可运行骨架”，不是最终业务代码。正式项目里通常还需要补唯一性错误转换、权限判断、审计日志、事件发布、复杂关系同步等业务逻辑。

## 选择建议

| 场景 | 推荐 |
| --- | --- |
| 小型示例、快速验证 | 原生 GORM |
| CRUD 模块很多 | 原生 GORM repository/service |
| 需要显式控制每条 SQL | 原生 GORM |
| 希望统一分页、过滤、排序与仓储风格 | 原生 GORM repository/service |
| 新建模块起步 | `gin-ninja-cli generate crud` |

上一篇：[响应、错误与 Schema](./data-and-responses.md) | 下一篇：[ModelSchema 高级用法](./model-schema.md)
