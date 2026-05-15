# 概览

## 概览

gin-ninja 适合希望继续使用 Gin，但又想要更强结构化 API 开发体验的 Go 项目：

- 使用普通 Go 结构体定义请求输入与响应输出
- 自动生成 OpenAPI 3.0 文档和 Swagger UI
- 通过中间件和操作选项复用认证、日志、安全、限流、超时等横切能力
- 同时支持基础 CRUD、版本化 API、缓存接口、SSE、WebSocket 等场景

典型使用场景：

- RESTful API 服务
- 需要严格请求/响应契约的项目
- 需要自动接口文档的内部平台或开放平台
- 需要 JWT、Session、CSRF、安全头、上传限制等生产能力的应用
- 需要版本隔离、路由缓存或实时推送的服务

## 主要特性

- **类型安全处理器**：基于泛型，直接使用 Go 结构体作为输入输出
- **自动参数绑定**：支持 `path`、`query`、`form`、`header`、`cookie`、`json`、`file`
- **默认值与校验**：支持 `default:"..."` 与 `binding:"..."`
- **自动 OpenAPI / Swagger**：默认暴露 `/openapi.json` 与 `/docs`
- **路由分组**：支持嵌套路由、标签、版本、路由级中间件
- **API Controller**：通过 `Controller` 接口与 `api.AddController` 将同一资源的路由组织进结构体，支持依赖注入
- **操作级控制**：支持 `Timeout`、`RateLimit`、额外响应声明、隐藏文档等
- **ModelSchema 风格响应**：支持字段白名单/黑名单过滤
- **路由级缓存**：支持 `Cache(...)`、`ETag()`、`CacheControl(...)`、标签失效、内存/Redis 存储
- **版本管理**：支持版本路由、版本文档、弃用与迁移头部
- **流式能力**：支持 SSE 与 WebSocket
- **日志能力**：基于 Zap，支持 console / JSON 输出、文件日志与按大小滚动
- **分页、过滤、排序**：支持 `pagination`、声明式过滤与安全排序
- **文件传输**：支持 multipart 上传与下载响应
- **配置与引导**：内置 settings、bootstrap、logger、ORM 集成
- **内置中间件**：CORS、JWT、i18n、Session、CSRF、安全头、请求日志、Request ID、上传限制、Recovery
- **统一错误模型**：支持协议级 HTTP 错误与验证错误

## 架构与请求流程

运行时的核心流程如下：

1. Gin 接收 HTTP 请求。
2. 引擎级与路由级中间件先执行。
3. gin-ninja 将路径、查询、头、Cookie、JSON、multipart 参数绑定到输入结构体。
4. 处理器以 `*ninja.Context` 和强类型输入结构体执行业务逻辑。
5. 框架统一输出 JSON、下载响应、SSE 或 WebSocket。
6. 路由元数据会被复用于 OpenAPI 文档与 Swagger UI 生成。

核心组件：

- **NinjaAPI**：管理 Gin 引擎、全局中间件、生命周期和文档端点
- **Router**：组织路由前缀、标签、版本和路由级中间件
- **Operation**：包装具体端点的入参绑定、选项控制与响应输出
- **Context**：扩展 `*gin.Context`，增加请求 ID、用户 ID、语言等能力
- **Middleware**：处理鉴权、日志、安全、国际化、上传等横切逻辑

## 包结构

```text
gin-ninja/
├── ninja.go          # NinjaAPI 核心实例
├── router.go         # Router 路由组
├── operation.go      # 类型化处理器包装与操作选项
├── binding.go        # 参数绑定
├── context.go        # 扩展上下文
├── errors.go         # 错误模型与错误写出
├── cache.go          # 路由缓存 / ETag / 缓存失效
├── openapi.go        # OpenAPI 3.0 生成与 Swagger UI
├── schema.go         # JSON Schema 生成
├── stream.go         # SSE 与 WebSocket
├── transfer.go       # 上传与下载抽象
├── versioning.go     # API 版本与弃用头部
│
├── middleware/       # 生产级 HTTP 中间件
├── bootstrap/        # 日志、数据库等初始化辅助
├── filter/           # 声明式过滤
├── order/            # 安全排序
├── orm/              # gormx 集成
├── pagination/       # 分页类型
├── pkg/              # i18n / logger / response 辅助包
├── settings/         # 基于 Viper 的配置加载
└── examples/         # basic、users、features、admin 与 full 示例
```

模块职责概览：

| 模块 | 主要职责 |
| --- | --- |
| `NinjaAPI` | 管理应用入口、Gin 引擎、OpenAPI/Swagger、生命周期 |
| `Router` | 按前缀、标签、版本组织端点 |
| `operation.go` | 绑定输入、调用处理器、输出响应、应用操作级选项 |
| `binding.go` | 解析 path/query/header/cookie/json/file 输入 |
| `middleware/` | 提供 JWT、日志、安全、Session、CSRF、i18n 等通用能力 |
| `cache.go` / `versioning.go` / `stream.go` | 提供缓存、版本、SSE、WebSocket 等高级特性 |

## 安装

```bash
go get github.com/shijl0925/gin-ninja
```

## Copilot Skill

仓库现在内置了一个工作区 Skill：`.github/skills/gin-ninja/`。

- 可以直接用 `/gin-ninja` 显式调用
- 也可以让智能体在处理 gin-ninja 相关 API、中间件、脚手架和 OpenAPI 任务时自动加载
