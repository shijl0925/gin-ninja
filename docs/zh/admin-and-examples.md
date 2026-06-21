# Admin 与完整示例

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## Admin 后台

`admin` 子包提供了元数据驱动的后台管理 API 层，以及一个与该 API 配套的内置单页后台 UI Shell。Site（注册中心）、API 路由、UI 页面三者独立挂载，可按需组合使用。

### 1. 创建 Site

```go
import admin "github.com/shijl0925/gin-ninja/admin"

site := admin.NewSite(
    // 可选：在每个操作前进行鉴权
    admin.WithPermissionChecker(func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        if ctx.GetUserID() == 0 {
            return ninja.UnauthorizedError()
        }
        return nil
    }),
)
```

`NewSite` 接受零或多个 `Option`。内置选项只有 `WithPermissionChecker`，它会在每次 list / detail / create / update / delete 操作前运行。

### 2. 使用 MustRegisterModel 注册 Model

每个 GORM Model 对应一个 `ModelResource` 描述符，用于控制哪些字段出现在哪些视图中，以及允许哪些操作。

```go
site.MustRegisterModel(&admin.ModelResource{
    // Model 是 GORM Model 结构体（值类型，非指针）
    Model: User{},

    // 可选：内置 UI 使用这些元数据进行分组、排序和说明展示
    Icon:        "users",
    Group:       "Identity",
    Description: "管理后台用户、管理员标记和角色关系。",
    Order:       10,

    // Preloads 列出每次查询时需要 Preload 的 GORM 关联名
    Preloads: []string{"Roles"},

    // 字段列表控制各视图显示的字段
    ListFields:   []string{"id", "name", "email", "is_admin", "createdAt"},
    DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt"},
    CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    FilterFields: []string{"is_admin", "age", "createdAt"},
    SortFields:   []string{"id", "name", "email", "age", "createdAt"},
    SearchFields: []string{"name", "email"},

    // 可选：字段级显示和组件覆盖
    FieldOptions: map[string]admin.FieldOptions{
        "is_admin": {Label: "管理员？", Component: "switch"},
        "role_ids": {Help: "搜索并选择一个或多个角色。", Width: "full"},
    },

    // 可选：资源级权限钩子，每次操作前调用
    Permissions: func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        return nil
    },

    // 可选：行级查询范围（例如多租户过滤）
    RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, res *admin.Resource, db *gorm.DB) *gorm.DB {
        return db.Where("owner_id = ?", ctx.GetUserID())
    }),

    // 可选：生命周期钩子
    BeforeCreate: func(ctx *ninja.Context, data map[string]any) error { return nil },
    AfterCreate:  func(ctx *ninja.Context, record any) error { return nil },
})
```

`MustRegisterModel` 在配置错误（如资源名重复）时会 panic；如需自行处理错误，可改用 `RegisterModel`。

指向另一个已注册 Model 的关联字段会被自动解析：框架会从目标资源推断 `value_field`、`label_field` 与 `search_fields`。

### 3. 注册 Admin API 路由

`site.Mount` 会为每个资源在指定的 `*ninja.Router` 下注册 REST 端点。该 Router 是标准的 gin-ninja Router，可以附加 JWT 中间件或其他任意 Gin 中间件。

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

以上配置（假设 `NinjaAPI` 的 `Prefix` 为 `"/api"`）会在 `/api/v1/admin` 下注册如下端点：

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/resources` | 列出所有已注册资源 |
| `GET` | `/resources/stats` | 获取当前可见资源的记录数聚合统计 |
| `GET` | `/search?q=关键词` | 跨当前可见且可搜索的资源执行聚合搜索 |
| `GET` | `/resources/{path}/meta` | 获取资源字段元数据 |
| `GET` | `/resources/{path}` | 分页列表（支持搜索 / 过滤 / 排序） |
| `GET` | `/resources/{path}/export` | 按当前搜索 / 过滤 / 排序条件导出 CSV |
| `GET` | `/resources/{path}/{id}` | 获取单条记录详情 |
| `POST` | `/resources/{path}` | 创建记录 |
| `PUT` | `/resources/{path}/{id}` | 更新记录 |
| `DELETE` | `/resources/{path}/{id}` | 删除记录 |
| `POST` | `/resources/{path}/bulk-delete` | 批量删除 |
| `GET` | `/resources/{path}/fields/{field}/options` | 关联字段选项列表 |

### 4. 挂载内置 Admin UI Shell

`admin.MountUI` 将独立登录页和 Admin 工作台作为普通 HTML 路由注册到任意 `gin.IRoutes`（包括 `api.Engine()`，以便使用 API 前缀之外的顶级路径）。

```go
// 使用全部默认值：/admin/login、/admin
admin.MountUI(api.Engine(), admin.DefaultUIConfig())

// 或自定义路径和标题：
admin.MountUI(api.Engine(), admin.UIConfig{
    Title:         "My App Admin",
    BrandName:     "My App",
    LogoText:      "MA",
    Locale:        "zh-CN",
    DefaultTheme:  "system",
    TokenStorage:  "session",
    APIBasePath:   "/api/v1/admin",
    AuthLoginPath: "/api/v1/auth/login",
    AdminPath:     "/admin",
    LoginPath:     "/admin/login",
})
```

`UIConfig` 字段及其默认值：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `Title` | `"Gin Ninja Admin"` | 浏览器标签标题 |
| `BrandName` | `"Gin Ninja"` | Admin 页面品牌名 |
| `LogoText` | `"G"` | 文本 Logo，最多显示 3 个字符 |
| `Locale` | `"en"` | HTML 语言和本地化格式化提示 |
| `DefaultTheme` | `"light"` | 默认主题，可选 `light` / `dark` / `system` |
| `TokenStorage` | `"local"` | token 存储位置，可选 `local` / `session` |
| `APIBasePath` | `"/api/v1/admin"` | Admin API 根路径（用于资源导航） |
| `AuthLoginPath` | `"/api/v1/auth/login"` | 登录表单调用的接口路径 |
| `AdminPath` | `"/admin"` | Admin 工作台页面路径 |
| `LoginPath` | `"/admin/login"` | 独立登录页路径 |
| `TokenExtractExpr` | `"payload.token"` | 从登录响应中提取 token 的 JS 表达式 |
| `UserNameExtractExpr` | `"payload.name"` | 从登录响应中提取展示名的 JS 表达式 |
| `UserIDExtractExpr` | `"payload.user_id \|\| payload.userID"` | 从登录响应中提取用户 ID 的 JS 表达式 |

#### 自定义 token 提取表达式

默认情况下，UI 从登录响应的 `payload.token` 字段读取 token。如果你的认证接口将 token 放在不同路径（例如 `{"data": {"accessToken": "..."}}` ），可以通过 `TokenExtractExpr` 配置：

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:    "/api/v1/user/login",
    // 适用于 {"data": {"accessToken": "..."}} 格式
    TokenExtractExpr: "payload.data && payload.data.accessToken",
})
```

该表达式会作为字符串配置传入内置的受限路径解析器，目前只支持 `payload.foo.bar` 这样的路径，以及用 `&&` / `||` 组合的兜底表达式；它不会被作为任意 JavaScript 代码执行。同样，`UserNameExtractExpr` 和 `UserIDExtractExpr` 用于自定义展示名和用户 ID 的读取路径：

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:       "/api/v1/user/login",
    TokenExtractExpr:    "payload.data && payload.data.accessToken",
    UserNameExtractExpr: "payload.data && payload.data.userName",
    UserIDExtractExpr:   "payload.data && payload.data.id",
})
```

> **安全说明：** 这些表达式仍应只来自可信的开发者配置，绝不能接受用户输入；不符合受限路径语法的表达式会解析失败。

## 完整示例

按功能拆分后的示例：

- [examples/users](../../examples/users/)：登录 / 注册、JWT 保护的 users CRUD，以及带缓存失效演示的 v2 users API
- [examples/features](../../examples/features/)：请求元数据、缓存 / ETag、限流、超时、版本化路由、SSE、WebSocket、上传、下载等能力演示
- [examples/admin](../../examples/admin/)：JWT 保护的 admin 资源 API 与独立 admin 页面
- [examples/full](../../examples/full/)：把以上能力组合到一个完整应用中
- [examples/compact](../../examples/compact/)：`examples/full` 的压缩版对照实现，用更少的本地文件展示同一套能力

完整应用可查看 [examples/full](../../examples/full/)：

- 基于 `config.yaml` 的配置加载
- 日志与数据库初始化
- JWT 保护的用户 CRUD
- 登录 / 注册接口
- 结构化日志
- 缓存 / ETag / Cache-Control 示例
- 版本化 API 与版本化文档
- SSE / WebSocket 示例
- 单文件、多文件上传
- 二进制下载与流式下载

### `examples/full` 中的 Admin 控制台原型

完整示例也包含一个基于元数据驱动的 admin 后台体验，它构建在 JWT 保护的 admin 资源 API 之上。

它包括：

- 独立登录页：`/admin/login`
- 独立后台工作台：`/admin`
- 由 `/api/v1/admin/resources` 驱动的资源导航
- 由 `/api/v1/admin/resources/stats` 一次性加载的 dashboard 资源记录数
- 由 `/api/v1/admin/search` 驱动的顶部全局聚合搜索
- 按资源 `Group` / `Order` 分组排序的侧边栏导航
- 支持搜索、元数据过滤、排序、分页大小和翻页的记录列表
- 空列表状态支持快速清除列表条件或创建首条记录
- 用状态 chip 展示当前搜索、排序、筛选与分页大小
- 切换资源时会在本地记住各资源自己的列表状态
- 弹窗自动聚焦关键控件，提升键盘优先的创建、编辑和详情流转体验
- 支持复制当前视图链接，便于分享或收藏列表状态
- 按当前搜索 / 过滤 / 排序条件导出 CSV
- 删除或批量删除清空当前页时，会自动回退到上一页重新加载
- 表格密度切换与列显隐设置
- 搜索、排序、分页与筛选会同步到 URL query，刷新或分享链接后可恢复当前列表状态
- 筛选区支持折叠，并会记住折叠状态
- 详情、创建、更新、删除与批量删除流程
- 详情弹窗支持快速编辑所选记录、复制 JSON payload
- 创建 / 更新表单支持内联校验、提交中状态和未保存变更提醒
- 表单提交失败后会自动聚焦第一个高亮错误字段
- 批量选择区展示已选数量，并支持一键清空选择
- 表格行支持点击和键盘 Enter 快速打开记录详情
- 带关系字段选项搜索预览的 selector 交互
- 更紧凑的 “Admin Workspace” 头部布局，后台观感更集中

推荐手动体验流程：

1. 启动完整示例：
   ```bash
   cd examples/full
   go run .
   ```
2. 打开 `http://localhost:8080/admin/login`
3. 使用页面展示的演示账号登录
4. 跳转到 `/admin` 后，从左侧选择资源
5. 在工作台中体验：
   - 搜索和过滤当前资源
   - 切换排序与分页大小
   - 浏览分页结果
   - 查看记录详情
   - 创建、编辑、删除或批量删除记录
   - 在关系字段输入时预览候选项

相关路由：

- `/admin/login` — 独立登录页
- `/admin` — 独立后台工作台
- `/api/v1/admin/resources` — admin 元数据与 CRUD API 根路径

运行：

```bash
cd examples/full
go run .
```

常用访问地址：

- `http://localhost:8080/docs`
- `http://localhost:8080/docs/v2`
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/openapi/v2.json`

---

上一篇: [高级功能](./advanced-features.md)
