# 中文文档

[文档首页](../README.md) | [English](../en/README.md)

这套文档按新手学习顺序组织：先写出能跑的接口，再理解输入、响应和路由，最后再接入配置、数据库、中间件、缓存、版本和后台等工程能力。每一页都可以单独阅读，但推荐先按下面路线走一遍。

## 先认识 4 个核心对象

- `NinjaAPI`：应用入口，负责 Gin 引擎、全局中间件、OpenAPI/Swagger、生命周期。
- `Router`：一组有共同前缀、标签和中间件的接口。
- `Operation`：一个 HTTP 方法、路径和 Go 处理函数的组合，例如 `ninja.Get(...)`。
- 输入/输出结构体：用普通 Go struct 声明请求来源、校验规则和响应 JSON。

最小形态如下：

```go
api := ninja.New(ninja.Config{Title: "Demo API"})

router := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
ninja.Get(router, "/", sayHello, ninja.Summary("Say hello"))
api.AddRouter(router)

api.Run(":8080")
```

## 推荐阅读路线

1. [概览](./overview.md)：知道 gin-ninja 解决什么问题，以及各目录负责什么。
2. [快速开始](./getting-started.md)：从空项目写出第一个类型化 API。
3. [NinjaAPI 与实例配置](./api-and-configuration.md)：理解 API 实例、全局中间件、OpenAPI、server 和 `ninja.Config`。
4. [Request Body 与 Form Data](./request-body-and-forms.md)：学习 JSON body、URL encoded form 和 multipart form。
5. [查询 Schema：分页、过滤与排序](./query-schemas.md)：集中学习列表接口的分页、过滤、搜索和排序。
6. [响应、错误与 Schema](./data-and-responses.md)：学习 DTO、分页响应、错误响应和响应 schema。
7. [CRUD 教程：Users 增删改查](./crud-tutorial.md)：用原生 GORM 写完整 users CRUD。
8. [ModelSchema 高级用法](./model-schema.md)：可选阅读，学习 model 派生 schema、字段裁剪、输入绑定和 preload。
9. [Router 与 Operation](./routers-and-operations.md)：集中查阅 Router 配置、operation 注册函数和选项。
10. [项目与 CRUD 脚手架](./scaffolding.md)：用 CLI 创建项目、app、迁移和 CRUD 起步代码。
11. [配置、Bootstrap 与生命周期](./configuration.md)：加载配置、初始化日志/数据库、处理启动和优雅关闭。
12. [中间件与安全](./middleware-security.md)：按需启用 CORS、认证、Session、CSRF、安全头、日志、i18n 和上传限制。
13. [文件传输：上传与下载](./file-transfer.md)：处理文件上传、下载和流式传输。
14. [使用 TestClient 测试 API](./testing.md)：写不依赖真实端口的 API 测试。
15. [高级功能](./advanced-features.md)：缓存、ETag、超时、限流、事务和 API 版本。
16. [可选 Admin 与完整示例](./admin-and-examples.md)：需要后台或想看完整项目时再读。

## 按任务查找

| 我想做什么 | 去哪一页 |
| --- | --- |
| 先跑起来 | [快速开始](./getting-started.md) |
| 查 `NinjaAPI` 和 `ninja.Config` 选项 | [NinjaAPI 与实例配置](./api-and-configuration.md) |
| 处理 JSON body、表单和 multipart 上传 | [Request Body 与 Form Data](./request-body-and-forms.md) |
| 写列表接口的分页、过滤、排序 | [查询 Schema：分页、过滤与排序](./query-schemas.md) |
| 定义响应 JSON、错误响应和响应 schema | [响应、错误与 Schema](./data-and-responses.md) |
| 写 users 增删改查 | [CRUD 教程：Users 增删改查](./crud-tutorial.md) |
| 可选使用 model 派生响应 schema | [ModelSchema 高级用法](./model-schema.md) |
| 拆分 users/orders 等模块 | [快速开始](./getting-started.md#组织多个接口) / [脚手架](./scaffolding.md) |
| 查 Router、Get/Post/Put/Patch/Delete 和 operation 选项 | [Router 与 Operation](./routers-and-operations.md) |
| 创建完整项目骨架 | [项目与 CRUD 脚手架](./scaffolding.md) |
| 配置数据库、日志、环境变量 | [配置、Bootstrap 与生命周期](./configuration.md) |
| 加 JWT、API Key、CORS、CSRF | [中间件与安全](./middleware-security.md) |
| 上传文件或返回下载文件 | [文件传输：上传与下载](./file-transfer.md) |
| 修改 Swagger/OpenAPI 展示 | [NinjaAPI 与实例配置](./api-and-configuration.md#文档与-openapi-路由) / [Router 与 Operation](./routers-and-operations.md) |
| 写 API 测试 | [使用 TestClient 测试 API](./testing.md) |
| 加缓存、限流、版本管理 | [高级功能](./advanced-features.md) |
