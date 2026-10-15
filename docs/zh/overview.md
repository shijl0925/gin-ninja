# 概览

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/overview.md)

gin-ninja 是一个构建在 Gin 之上的 API 框架。它保留 Gin 的路由性能和生态，同时补上类型化输入输出、自动 OpenAPI 文档、常用中间件、分页过滤、配置加载、脚手架和测试工具。

## 适合什么项目

gin-ninja 适合这些场景：

- 你想继续使用 Gin，但不想在每个接口里手写 `ShouldBindQuery`、`ShouldBindJSON` 和响应 schema。
- 你希望接口代码和 Swagger/OpenAPI 文档来自同一份路由定义。
- 你在写 CRUD、后台、内部平台或服务端 API，需要请求校验、分页、过滤、认证和测试工具。
- 你希望小项目可以很轻，项目变大后也能拆成 router、controller、app package 和脚手架生成代码。

如果只是写一个极小的 Gin demo，直接用 Gin 也很好；当接口开始变多、参数和文档开始重复时，gin-ninja 的价值会更明显。

## 请求如何流转

一次请求进入 gin-ninja 后，大致经历这些步骤：

1. Gin 接收 HTTP 请求。
2. 全局或路由级中间件先运行，例如日志、CORS、JWT。
3. gin-ninja 根据输入结构体标签绑定 `path`、`query`、`header`、`cookie`、`json`、`form`、`file`。
4. `binding:"..."` 校验失败时自动返回错误响应。
5. 处理函数执行业务逻辑并返回 Go 结构体。
6. 框架把结果写成 JSON、分页响应或下载响应。
7. 同一份路由和结构体元数据用于生成 `/openapi.json` 和 `/docs`。

## 核心模块地图

| 模块 | 你会在什么时候用 |
| --- | --- |
| `ninja.New` / `NinjaAPI` | 创建应用、设置标题、版本、全局前缀、文档地址、生命周期和全局中间件 |
| `Router` | 按业务资源分组接口，例如 `/users`、`/orders` |
| `ninja.Get/Post/Put/Patch/Delete` | 注册一个类型化接口 |
| `Controller` | 把同一个资源的多个接口组织到结构体里，便于注入 DB 或 service |
| `binding.go` | 自动绑定请求输入；日常使用只需要写 struct tag |
| `pagination` | 统一 `page`、`size` 输入和分页响应 |
| `filter` / `order` | 用 tag 声明可过滤、可排序字段，避免拼接不安全 SQL |
| `middleware` | CORS、JWT、API Key、Session、CSRF、安全头、日志、恢复、i18n、上传限制 |
| `settings` | 从 YAML 和环境变量加载配置 |
| `bootstrap` | 初始化日志和数据库连接 |
| `orm` | 将 GORM DB 放入请求上下文，配合事务和 repository 使用 |
| `cache.go` | 为读接口开启缓存、ETag 和 Cache-Control |
| `versioning.go` | 为不同 API 版本生成独立路由和文档 |
| `testing` | 用 `ninjatest.TestClient` 写内存内 API 测试 |
| `cmd/gin-ninja-cli` | 创建项目、app、迁移和 CRUD 起步代码 |
| `admin` | 可选后台 API 和内置 UI shell，适合内部管理工具 |

## 目录快速认识

```text
gin-ninja/
├── ninja.go              # NinjaAPI 和 Config
├── router.go             # Router、Controller、路由注册函数
├── operation.go          # Operation 选项、响应写出、超时/限流/事务
├── binding.go            # 请求绑定和校验
├── transfer.go           # UploadedFile 和 Download
├── openapi.go            # OpenAPI/Swagger/ReDoc
├── cache.go              # 路由级缓存
├── versioning.go         # API 版本
├── middleware/           # 常用 HTTP 中间件
├── settings/             # 配置加载
├── bootstrap/            # 日志和数据库初始化
├── filter/ order/        # 过滤和排序
├── pagination/           # 分页输入和输出
├── testing/              # TestClient
├── admin/                # 可选 Admin
├── cmd/gin-ninja-cli/    # 脚手架和迁移 CLI
└── examples/             # 可运行示例
```

## 第一次学习建议

不要一开始就把所有模块都接上。推荐顺序是：

1. 只写一个 `NinjaAPI`、一个 `Router`、一个 `GET` 接口。
2. 再给接口加 `query`、`path`、`json` 输入和 DTO 响应。
3. 接着把 users 这类资源拆到单独 router 或 controller。
4. 项目开始需要数据库时，再引入 `settings`、`bootstrap`、`orm` 和迁移。
5. 上线前补齐 CORS、认证、安全头、日志、测试和必要的缓存。

下一篇：[快速开始](./getting-started.md)
