# 概览

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## gin-ninja 是什么？

gin-ninja 适合希望保留 Gin 路由性能，同时获得更结构化 API 层的 Go 团队：

- 使用普通 Go 结构体定义处理器输入，避免手写绑定样板代码
- 从同一套路由定义自动生成 OpenAPI 和 Swagger UI
- 将横切能力放在可复用中间件和操作选项中
- 从小型 CRUD 服务扩展到版本化、有文档、面向生产的 API

典型使用场景：

- 需要严格请求/响应契约的 REST API
- 需要快速迭代且接口文档始终最新的内部平台
- 需要内置认证、安全头、请求日志和配置加载的服务
- 需要版本化 API、可缓存读端点或实时 SSE / WebSocket 路由的应用

## 架构概览

运行时，gin-ninja 在 Gin 之上增加类型化 API 层：

1. Gin 接收传入的 HTTP 请求。
2. 引擎级和路由级中间件先执行。
3. gin-ninja 将 path/query/header/cookie/body/file 输入绑定到类型化结构体。
4. 类型化处理器通过 `*ninja.Context` 执行。
5. 框架写出 JSON、下载、SSE 或 WebSocket 响应。
6. 路由元数据会复用于生成 OpenAPI 文档和 Swagger UI。

## 主要特性

- **类型安全处理器** – 使用普通 Go 结构体定义请求输入和响应输出。
- **自动参数绑定** – path 参数（`path:`）、query 参数（`query:`）、请求头（`header:`）、Cookie（`cookie:`）以及 JSON/form 请求体（`json:` / `form:`）都通过结构体标签绑定。
- **默认参数值** – `default:"..."` 适用于 query/header/cookie 字段，并会反映到 OpenAPI。
- **校验** – 基于 [go-playground/validator](https://github.com/go-playground/validator)，使用标准 `binding:` 标签。
- **文件传输抽象** – 一等支持 multipart 上传绑定和二进制下载响应。
- **自动生成 OpenAPI 3.0 文档** – 作为 `/openapi.json` 提供。
- **Swagger UI** – 开箱即用地提供 `/docs`。
- **路由分组** – 使用共享前缀、OpenAPI 标签和路由级中间件嵌套路由。
- **API Controller** – 通过 `Controller` 接口和 `api.AddController` 将资源路由组织到支持依赖注入的结构体中。
- **Gin 中间件支持** – API 和单个 Router 都支持 `UseGin()`。
- **OpenAPI 控制** – 从文档中隐藏内部端点，并为操作声明额外响应。
- **操作级控制** – 每端点超时、内存限流和标准分页响应声明。
- **ModelSchema 风格响应** – 使用 `fields` / `exclude` 控制过滤后的 JSON 输出和 OpenAPI schema。
- **路由级缓存** – 内置 `Cache(...)`、`ETag()`、`CacheControl(...)`、缓存标签和可插拔的内存/Redis 存储，适合读多端点。
- **API 版本隔离** – 版本感知路由、按版本输出 OpenAPI/Swagger，以及弃用响应头。
- **流式端点** – 一等支持 SSE 和 WebSocket 路由注册辅助函数。
- **分页** – 可复用 `PageInput` 和 `Page[T]` 类型，统一列表响应。
- **ORM 集成** – 基于 [gormx](https://github.com/shijl0925/go-toolkits/tree/main/gormx) 的 repository/service 模式薄封装。
- **内置中间件** – CORS、JWT 认证、结构化请求日志（Zap）、请求 ID、panic 恢复、i18n 语言协商、**HMAC 签名 Cookie session**、**CSRF 防护**、**安全响应头**、**上传大小/内容类型限制**。
- **生命周期钩子** – 启动和关闭钩子，支持优雅关闭。
- **配置管理** – 基于 Viper 的 YAML/env 配置管理，支持**多环境覆盖**。
- **日志** – 基于 Zap 的结构化日志，支持 console/JSON 输出、文件输出和按大小滚动。
- **标准响应信封** – `{"code": 200, "message": "success", "data": ...}`。
- **Bootstrap 辅助函数** – 一次调用完成数据库和日志初始化。
- **i18n / L10n** – 通过 `Accept-Language` 协商语言，支持中英文校验错误和通用消息翻译。
- **API 版本弃用** – 符合 RFC 的 `Deprecation` 与 `Sunset` 日期头，以及迁移链接。

---

## 包结构

```
gin-ninja/
├── ninja.go          ← NinjaAPI（核心 API 实例）
├── router.go         ← Router（路由分组）
├── operation.go      ← 类型化处理器包装
├── binding.go        ← 参数绑定（path/query/header/body）
├── context.go        ← Context（扩展 *gin.Context）
├── errors.go         ← 类型化错误类型
├── cache.go          ← 路由缓存、ETag、缓存失效辅助函数
├── openapi.go        ← OpenAPI 3.0 spec 生成 + Swagger UI
├── schema.go         ← JSON Schema 生成
├── stream.go         ← SSE 与 WebSocket 支持
├── transfer.go       ← 上传/下载抽象
├── versioning.go     ← 版本感知文档与弃用响应头
│
├── middleware/       ← 生产级 HTTP 中间件
│   ├── cors.go       ← CORS（gin-contrib/cors）
│   ├── csrf.go       ← CSRF double-submit cookie 防护
│   ├── i18n.go       ← 语言协商（Accept-Language）
│   ├── jwt.go        ← JWT 认证（golang-jwt/jwt）
│   ├── logger.go     ← 结构化请求日志（Zap）
│   ├── recovery.go   ← panic 恢复
│   ├── requestid.go  ← 注入 X-Request-ID
│   ├── secure.go     ← 安全响应头
│   ├── session.go    ← HMAC 签名 Cookie session
│   └── upload.go     ← 上传大小限制 + content-type 白名单
│
├── pkg/
│   ├── i18n/         ← 语言协商 + 校验错误翻译
│   │   └── i18n.go
│   ├── logger/       ← Zap logger bootstrap
│   └── response/     ← 标准 JSON 响应信封
│
├── settings/         ← 基于 Viper 的配置
│   └── settings.go   ← Config、Load、MustLoad、LoadWithOverrides、LoadForEnv
│
├── bootstrap/        ← 应用 bootstrap 辅助函数
│   └── bootstrap.go  ← InitLogger、InitDB、MustInitDB
│
├── filter/           ← 声明式查询过滤构建器
├── order/            ← 安全排序辅助函数
├── orm/              ← gormx 集成
│   └── orm.go        ← Init、Middleware、GetDB、WithContext
│
├── pagination/       ← 分页类型
│   └── pagination.go ← PageInput、Page[T]
│
└── examples/         ← 可运行的 basic、users、features、admin 和 full 应用
```

核心模块职责：

| 模块 | 职责 |
| --- | --- |
| `NinjaAPI` | 管理 Gin 引擎、全局中间件、生命周期钩子和 OpenAPI/Swagger 端点 |
| `Router` | 按前缀、标签、版本和路由级中间件组织端点 |
| `operation.go` | 包装类型化处理器、绑定输入、执行选项并写出类型化响应 |
| `binding.go` | 将 path/query/header/cookie/json/multipart 输入映射到结构体 |
| `middleware/` | 提供生产级认证、日志、i18n、安全、session 和上传中间件 |
| `cache.go` / `versioning.go` / `stream.go` | 添加缓存、API 版本/弃用、SSE 和 WebSocket 能力 |

---

## 安装

```bash
go get github.com/shijl0925/gin-ninja
```

## Copilot Skill

仓库现在内置了工作区 Skill：`.github/skills/gin-ninja/`。

- 可以用 `/gin-ninja` 显式调用
- 也可以让智能体在处理 gin-ninja 相关 API、中间件、脚手架和 OpenAPI 任务时自动加载

---

下一篇: [快速开始](./getting-started.md)
