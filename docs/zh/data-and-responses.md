# 数据、绑定与响应

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## ModelSchema 风格响应

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

`ResponseModel[UserOut]()` 会在运行时把处理器返回的 `User` 绑定到 `UserOut`，再执行响应校验和字段裁剪。`fields:"..."` 只保留列出的可序列化字段，`exclude:"..."` 会从 JSON 响应和生成的 OpenAPI schema 中移除敏感字段。

如果你更喜欢复用描述符而不是定义单独的 `UserOut` 结构体，可以使用 `ResponseSchema`：

```go
userSchema := ninja.ModelReadSchemaOf[User]().
    Fields("id", "name", "email").
    ComponentName("UserOut")

func getUser(ctx *ninja.Context, in *struct{}) (*User, error) {
    return &User{ID: 1, Name: "alice", Email: "alice@example.com", Password: "secret"}, nil
}

ninja.Get(router, "/users/:id", getUser, ninja.ResponseSchema(userSchema))
```

`ResponseSchema(...)` 同样会在运行时裁剪字段，并让 OpenAPI 使用同一份 schema。对于切片、`pagination.Page[T]` 和 `pagination.CursorPage[T]`，可以分别使用 `ResponseModel[[]UserOut]()`、`Paginated[UserOut]()` / `PaginatedSchema(...)`、`CursorPaginated[UserOut]()` / `CursorPaginatedSchema(...)`，分页响应的 `items` 也会逐项绑定和校验。

如果只需要临时过滤而不想定义响应类型或路由级 schema，可以使用 `ninja.NewModelSchema(model, ninja.Fields(...), ninja.Exclude(...))`。如果需要显式得到一个 schema 值，也可以继续使用 `ninja.BindModelSchema[UserOut](model)` 或 `ninja.BindModelSchemas[UserOut](models)`。

### 关系深度与 GORM preload

`Depth(n)` 会把嵌套 model 按同一模式继续裁剪，适合 detail 响应：

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

如果使用描述符风格，可以用 `descriptor.Preloads()` 查看 preload 路径，或者用 `orm.ApplyModelSchemaPreloads(db, descriptor)` 直接应用到 GORM 查询。

---

## 标准响应信封

```go
import "github.com/shijl0925/gin-ninja/pkg/response"

// 成功：{"code": 200, "message": "success", "data": {...}}
response.Success(c, users)

// 错误：{"code": 404, "message": "not found", "data": null}
response.NotFound(c, "user not found")

// 自定义：{"code": 200, "message": "created", "data": {...}}
response.JSON(c, response.OKWithMessage("created", user))
```

框架错误响应使用相同的根级信封：

```json
{"code": 404, "message": "not found", "data": null}
```

---

## 参数绑定

| 标签 | 来源 | 适用方法 |
| --- | --- | --- |
| `path:"x"` | URL 路径参数 | 全部 |
| `query:"x"` | URL 查询参数 | 全部 |
| `form:"x"` | 表单请求体字段 | POST / PUT / PATCH |
| `header:"x"` | 请求头 | 全部 |
| `cookie:"x"` | 请求 Cookie | 全部 |
| `json:"x"` | JSON 请求体 | POST / PUT / PATCH |
| `file:"x"` | Multipart 上传文件 | POST / PUT / PATCH |

`binding:"..."` 使用 [go-playground/validator](https://github.com/go-playground/validator)。

`default:"..."` 适用于客户端省略值时的 `query`、`form`、`header` 和 `cookie` 字段。

---

## 声明式过滤与安全排序

### 声明式过滤

在列表输入结构体中嵌入 `pagination.PageInput`，然后为需要转换成数据库过滤条件的查询字段添加 `filter:"column,op"`。如果一个输入字段需要匹配多个列，用 `|` 分隔列名：

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search"   filter:"name|email,like" description:"Filter by name or email (partial match)"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
}
```

支持的操作符：

- `eq`
- `ne`
- `gt`
- `ge`
- `lt`
- `le`
- `like`
- `in`

在处理器中应用声明的过滤条件：

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

行为说明：

- 只有带 `filter:"..."` 标签的字段参与过滤
- 零值会被忽略，因此省略的查询参数不会添加条件
- `like` 适合包含式模糊匹配
- `filter:"name|email,like"` 表示 `(name LIKE ? OR email LIKE ?)`；多字段声明式过滤使用 OR 语义
- 对于标签无法表达的逻辑，输入结构体可以实现 `FilterExpression() clause.Expression`；返回的表达式会由 `filter.BuildOptions(...)` 和 `filter.ApplyDB(...)` 与标签过滤条件按 `AND` 一起应用
- 无效过滤声明会在你暴露 `filter.BuildOptions(...)` 或 `filter.Apply(...)` 错误时返回 400

### 安全排序

使用带 `order:"..."` 白名单的 `sort` 查询参数。字段前加 `-` 表示降序，加 `+` 表示升序：

- `sort=name`
- `sort=-created_at`
- `sort=name,-age`

分页处理器继续使用 `pagination.PageInput` 表示 page/size，并单独声明 `Sort`：

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

### 游标分页

当列表端点需要在大数据量或频繁变化的数据集上保持稳定分页时，使用 `pagination.CursorPagination`。游标对 gin-ninja 是不透明的；你可以编码存储层需要的有序字段值，然后通过 `pagination.CursorPage[T]` 返回下一页游标：

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

使用 `ninja.CursorPaginated[EventOut]()` 声明响应信封文档和运行时 item schema。

对于简单的 GORM 单列 keyset 分页，可以使用 `orm.SelectCursorPage`：

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

如果需要把公开别名映射到不同的数据库列，可以使用 `alias:column` 或 `alias=column`：

```go
type ListUsersInput struct {
    Sort string `query:"sort" order:"name|created:created_at"`
}
```

白名单之外的排序字段会被拒绝，不会直接传到查询层。

### 示例

完整示例应用在分页用户列表中使用了声明式排序：

- `GET /api/v1/users` → 分页过滤 + 排序
- `sort` → 在进入查询层前由 `order:"..."` 白名单校验

可以尝试这些请求：

- `/api/v1/users?search=ali`
- `/api/v1/users?is_admin=true&sort=-age`

---

上一篇: [中间件与安全](./middleware-security.md) | 下一篇: [文件传输与 OpenAPI 控制](./files-and-openapi.md)
