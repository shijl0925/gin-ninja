# 可选 Admin 与完整示例

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/admin-and-examples.md)

`admin` 是可选模块。普通 API 项目不需要一开始就接入它；当你需要内部管理后台、资源列表、详情、创建、更新、删除、导出和简单 UI 时，再阅读本页。

## Admin 由三部分组成

- `Site`：注册后台资源和全局权限。
- Admin API：把资源操作挂到 gin-ninja `Router` 上。
- Admin UI Shell：一个内置 HTML/JS 后台壳，调用 Admin API 工作。

这三部分可以分开使用。你可以只挂 API，也可以同时挂 API 和 UI。

## 创建 Site

```go
import (
    ninja "github.com/shijl0925/gin-ninja"
    admin "github.com/shijl0925/gin-ninja/admin"
)

site := admin.NewSite(
    admin.WithPermissionChecker(func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        if ctx.GetUserID() == 0 {
            return ninja.UnauthorizedError()
        }
        return nil
    }),
)
```

`WithPermissionChecker` 会在 list、detail、create、update、delete、bulk delete 等操作前运行。没有统一权限需求时，可以直接 `admin.NewSite()`。

## 注册资源

每个 GORM model 对应一个资源：

```go
site.MustRegisterModel(&admin.ModelResource{
    Model:       User{},
    Icon:        "users",
    Group:       "Identity",
    Description: "Manage users and roles.",
    Order:       10,

    ListFields:   []string{"id", "name", "email", "is_admin", "createdAt"},
    DetailFields: []string{"id", "name", "email", "age", "is_admin", "createdAt"},
    CreateFields: []string{"name", "email", "password", "age", "is_admin"},
    UpdateFields: []string{"name", "email", "password", "age", "is_admin"},
    FilterFields: []string{"is_admin", "age"},
    SortFields:   []string{"id", "name", "email", "createdAt"},
    SearchFields: []string{"name", "email"},

    FieldOptions: map[string]admin.FieldOptions{
        "password": {Placeholder: "Leave blank to keep current password."},
        "is_admin": {Label: "Admin", Component: "switch"},
    },
})
```

`MustRegisterModel` 遇到配置错误会 panic，适合启动期快速暴露问题。需要自己处理错误时，用 `RegisterModel`。

资源里最常用的字段：

| 字段 | 作用 |
| --- | --- |
| `Model` | GORM model 值，例如 `User{}` |
| `ListFields` | 列表页展示字段 |
| `DetailFields` | 详情页展示字段 |
| `CreateFields` / `UpdateFields` | 创建和编辑表单字段 |
| `FilterFields` / `SortFields` / `SearchFields` | 列表查询能力 |
| `FieldOptions` | 字段标签、组件、placeholder、帮助文案、格式等 |
| `Permissions` | 单资源权限 |
| `RowPermissions` | 行级查询范围，例如多租户 |
| `BeforeCreate` / `AfterCreate` 等 | 生命周期钩子 |

行级权限示例：

```go
RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, res *admin.Resource, db *gorm.DB) *gorm.DB {
    return db.Where("owner_id = ?", ctx.GetUserID())
}),
```

## 挂载 Admin API

Admin API 挂在普通 gin-ninja router 上，所以可以照常加 JWT、版本、标签和其他中间件：

```go
adminRouter := ninja.NewRouter(
    "/admin",
    ninja.WithTags("Admin"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
    ninja.WithVersion("v1"),
)

site.Mount(adminRouter)
api.AddRouter(adminRouter)
```

假设 `api.Prefix` 是 `/api`，上面会得到 `/api/v1/admin/...` 这一组端点。

常用端点：

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| `GET` | `/resources` | 资源列表 |
| `GET` | `/resources/stats` | 资源记录数统计 |
| `GET` | `/search?q=keyword` | 跨资源搜索 |
| `GET` | `/resources/{path}/meta` | 字段元数据 |
| `GET` | `/resources/{path}` | 分页列表 |
| `GET` | `/resources/{path}/export` | CSV 导出 |
| `GET` | `/resources/{path}/{id}` | 详情 |
| `POST` | `/resources/{path}` | 创建 |
| `PUT` | `/resources/{path}/{id}` | 更新 |
| `DELETE` | `/resources/{path}/{id}` | 删除 |
| `POST` | `/resources/{path}/bulk-delete` | 批量删除 |
| `GET` | `/resources/{path}/fields/{field}/options` | 关联字段选项 |

## 挂载内置 UI

默认 UI：

```go
admin.MountUI(api.Engine(), admin.DefaultUIConfig())
```

自定义：

```go
admin.MountUI(api.Engine(), admin.UIConfig{
    Title:         "My App Admin",
    BrandName:     "My App",
    LogoText:      "MA",
    Locale:        "zh-CN",
    DefaultTheme:  "system",
    TokenStorage:  "session",
    APIBasePath:   "/api/v1/admin",
    AuthLoginPath: "/api/v1/auth/login",
    AuthMePath:    "/api/v1/auth/me",
    AdminPath:     "/admin",
    LoginPath:     "/admin/login",
})
```

登录接口返回 token 的字段如果不是默认的 `payload.token`，可以配置提取表达式：

```go
admin.MountUI(api.Engine(), admin.UIConfig{
    AuthLoginPath:       "/api/v1/auth/login",
    TokenExtractExpr:    "payload.data && payload.data.accessToken",
    UserNameExtractExpr: "payload.data && payload.data.userName",
    UserIDExtractExpr:   "payload.data && payload.data.id",
})
```

这些表达式只支持受限的 `payload.foo.bar` 路径和 `&&` / `||` 组合，不会作为任意 JavaScript 执行。配置仍应只来自可信开发者。

### 当前用户接口（`AuthMePath`）

`AuthMePath` 配置用于校验已保存 token 并读取当前登录用户资料的接口，默认值为 `/api/v1/auth/me`。内置 UI 在恢复 token 或用户登录后，会向该路径发送带有 `Authorization: Bearer <token>` 的 `GET` 请求。

接口成功时应返回 JSON（`2xx`），资料可以直接放在响应根对象，也可以嵌套在 `user` 对象中。所有资料字段都可选，但建议至少返回 `name`、`user_id` 或 `email` 中的一个，以便 UI 持久化并展示当前身份：

```json
{
  "user_id": 42,
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "is_admin": true,
  "roles": [
    { "id": 1, "name": "Administrator", "code": "admin", "status": "active" }
  ],
  "expires_at": "2026-12-31T23:59:59Z",
  "issued_at": "2026-01-01T00:00:00Z",
  "issuer": "gin-ninja"
}
```

| UI 信息 | 可接受的响应字段 |
| --- | --- |
| 展示名称 | `name`、`username` 或 `user_name` |
| 用户 ID | `user_id`、`userID` 或 `id` |
| 邮箱 | `email` |
| 管理员标识 | `is_admin`、`isAdmin` 或 `admin` |
| 角色 | `roles` 或 `permissions`；每一项可以是字符串，或包含 `id`/`ID`、`name`/`Name`、`code`/`Code`、`status`/`Status` 的对象 |
| Token 元信息 | `expires_at`/`expiresAt`、`issued_at`/`issuedAt` 和 `issuer` |

例如，`{ "user": { "id": 42, "username": "Ada" } }` 也是合法响应。接口返回 `401` 时，UI 会清除会话并跳转到登录页；返回 `403` 时，UI 会显示无访问权限页面。Admin API 仍必须在服务端执行授权校验，`is_admin` 和 `roles` 仅用于 UI 的资料展示。

## 可运行示例

仓库里的示例按复杂度递增：

| 示例 | 说明 |
| --- | --- |
| `examples/basic` | 一个 users CRUD API，适合读入门代码 |
| `examples/users` | 登录、注册、JWT 保护的 users API |
| `examples/features` | 缓存、ETag、限流、超时、版本、上传、下载 |
| `examples/admin` | 聚焦 admin API 和 UI |
| `examples/full` | 配置、数据库、认证、中间件、admin、版本和文件功能的完整组合 |
| `examples/compact` | full 示例的紧凑版 |
| `examples/controller` | Controller 组织方式示例 |

运行示例：

```bash
go run ./examples/basic
go run ./examples/features
go run ./examples/admin
go run ./examples/full
```

启动后通常可以访问：

- `/docs`：Swagger UI
- `/openapi.json`：OpenAPI JSON
- `/admin/login` 和 `/admin`：启用 Admin UI 的示例

## 什么时候不该用 Admin

- 面向公网用户的完整产品后台，需要复杂交互和自定义设计。
- 权限模型很复杂，字段、行、操作都需要多层策略。
- 资源不是 GORM model，或查询无法自然映射到列表/详情/表单。

这些场景可以只复用 gin-ninja API 能力，单独实现前端后台。

上一篇：[高级功能](./advanced-features.md)
