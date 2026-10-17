# 查询 Schema：分页、过滤与排序

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/query-schemas.md)

列表接口通常不是只接收一个 `page` 参数。真实业务里会同时有分页、搜索、过滤、排序和默认值。gin-ninja 推荐把这些查询能力写在一个输入结构体里：这个结构体就是列表查询的 query schema。

```go
type ListUsersInput struct {
    pagination.PageInput
    Search  string `query:"search" description:"Filter by name or email"`
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
    Sort    string `query:"sort" order:"id|name|email|age|created_at"`
}
```

这里有三层含义：

- `query:"..."` 决定从 URL query string 绑定参数。
- `pagination.PageInput` 提供标准页码分页字段和 helper。
- `filter:"..."` 与 `order:"..."` 描述安全的数据库查询能力。

## Query Schema 是什么

在 gin-ninja 里，schema 不是单独的 DSL，而是普通 Go struct 加 tag。它同时服务三件事：

| 目标 | 写法 | 结果 |
| --- | --- | --- |
| 请求绑定 | `query:"search"` | 从 `?search=alice` 绑定字段 |
| 参数校验 | `binding:"omitempty,min=1"` | 使用 validator 校验 |
| 查询声明 | `filter:"is_admin,eq"` / `order:"name|created_at"` | 生成安全的 GORM 查询条件 |

建议每个列表接口单独定义 input struct，不要复用过大的“万能查询结构体”。这样 Swagger/OpenAPI、handler 代码和数据库查询能力会保持一致。

## 页码分页

`pagination.PageInput` 是最常用的分页输入：

```go
type ListUsersInput struct {
    pagination.PageInput
}
```

它包含：

| 字段 | query 参数 | 默认/限制 |
| --- | --- | --- |
| `Page` | `page` | 默认 `1`，最小 `1` |
| `Size` | `size` | 默认 `20`，最大 `100` |

常用 helper：

| 方法 | 作用 |
| --- | --- |
| `GetPage()` | 返回有效页码 |
| `GetSize()` | 返回有效 page size |
| `Offset()` | 计算 SQL OFFSET |
| `Limit()` | 返回 SQL LIMIT |

返回值通常使用 `pagination.Page[T]`：

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

响应结构：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "size": 20,
  "pages": 0
}
```

`ninja.Paginated[UserOut]()` 会让 OpenAPI 把成功响应展示成分页信封，并对 items 使用 `UserOut` schema。

## 游标分页

游标分页适合大数据量、时间线、消息流等场景。输入使用 `pagination.CursorPagination`：

```go
type ListEventsInput struct {
    pagination.CursorPagination
}
```

它包含：

| 字段 | query 参数 | 说明 |
| --- | --- | --- |
| `Cursor` | `cursor` | 上一页返回的不透明游标 |
| `Size` | `size` | 默认 `20`，最大 `100` |

返回值使用 `pagination.CursorPage[T]`：

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

游标值对框架是不透明的。你可以把稳定排序字段编码进去，例如 `created_at + id`，再由 service/repository 解码。

如果使用 GORM，并且游标只依赖一个可信列，可以用 `orm.SelectCursorPage` 做单列 keyset 查询：

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

`cursorColumn` 必须是服务端写死的可信列名，不要直接使用客户端传入的字段。需要多列稳定排序时，可以在 repository 中自定义查询和 cursor 编解码。

## 过滤

过滤通过 `filter:"column,op"` 声明。字段为空时不会生成过滤条件。

```go
type ListUsersInput struct {
    pagination.PageInput
    IsAdmin *bool  `query:"is_admin" filter:"is_admin,eq"`
    AgeMin  int    `query:"age_min"  filter:"age,ge"`
    IDs     []uint `query:"ids"      filter:"id,in"`
}
```

支持的操作符：

| 操作符 | 含义 |
| --- | --- |
| `eq` | 等于 |
| `ne` | 不等于 |
| `gt` | 大于 |
| `ge` | 大于等于 |
| `lt` | 小于 |
| `le` | 小于等于 |
| `like` | LIKE 匹配 |
| `in` | IN 列表 |

多字段搜索可以用 `|` 表达 OR：

```go
type ListUsersInput struct {
    pagination.PageInput
    Search string `query:"search" filter:"name|email,like"`
}
```

这会生成类似 `name LIKE ? OR email LIKE ?` 的条件。

## 自定义过滤表达式

当过滤逻辑无法用一个 tag 表达时，让 input 实现 `FilterExpression()`。`filter.ApplyDB` 会自动追加这个表达式：

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

自定义表达式适合复杂搜索、组合条件、按当前租户/权限补充过滤等场景。

## 排序

排序通过 `order:"..."` 声明白名单。客户端只能按白名单里的字段排序：

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `query:"sort" order:"id|name|email|age|created_at"`
}
```

请求示例：

```text
?sort=name
?sort=-created_at
?sort=name,-created_at
?sort=+age
```

规则：

- `name` 表示升序。
- `-created_at` 表示降序。
- `+age` 显式表示升序。
- `name,-created_at` 支持多个排序字段。
- 不在 `order:"..."` 白名单内的字段会返回错误，不会拼进 SQL。

如果公开参数名和数据库列名不同，可以使用别名映射：

```go
type ListUsersInput struct {
    pagination.PageInput
    Sort string `query:"sort" order:"created:created_at|display_name:name"`
}
```

这样客户端用 `?sort=-created`，实际按 `created_at DESC` 排序。

## 原生 GORM 用法

原生 GORM 使用 `filter.ApplyDB` 和 `order.ApplyDB`：

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

实际项目里通常会把查询组装放进 repository/service，handler 只负责读取 input、调用 service、返回 DTO。

## 完整 users 示例

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

可用请求：

```text
GET /users/?page=1&size=20
GET /users/?search=alice
GET /users/?is_admin=true
GET /users/?sort=-created_at
GET /users/?search=alice&is_admin=false&sort=name,-created_at
```

## 设计建议

| 建议 | 原因 |
| --- | --- |
| 每个列表接口定义自己的 input | OpenAPI 更准确，handler 更容易读 |
| 排序必须使用 `order:"..."` 白名单 | 避免用户输入直接进入 SQL |
| 过滤字段尽量使用数据库列名 | 生成的查询更直观 |
| 公开参数和数据库列不同才用别名 | 避免 API 泄漏内部字段名 |
| 简单搜索用 `filter:"name|email,like"` | 代码短，OpenAPI 也清楚 |
| 复杂搜索用 `FilterExpression()` | 保持 tag 简洁 |
| 列表响应配 `ninja.Paginated[T]()` | 文档和运行时响应一致 |
| 大数据流使用游标分页 | 避免深翻页 OFFSET 成本 |

上一篇：[Request Body 与 Form Data](./request-body-and-forms.md) | 下一篇：[响应、错误与 Schema](./data-and-responses.md)
