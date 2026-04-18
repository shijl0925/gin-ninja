---
name: gin-ninja
description: '用于构建、扩展、重构或排查基于 github.com/shijl0925/gin-ninja 的 Go HTTP API。帮助选择 NinjaAPI/Router 模式、类型化请求响应结构、绑定标签、中间件、分页、缓存、版本化、流式能力、admin 特性、settings/bootstrap 接线，以及 gin-ninja-cli 脚手架命令。'
argument-hint: 你想用 gin-ninja 构建或修改什么？
---

# gin-ninja

[English](./SKILL.md) | [中文](./SKILL-zh.md)

当任务属于 `github.com/shijl0925/gin-ninja` 服务，或你想按仓库既有风格创建新服务时，使用这个 skill。

## 适用场景

- 新增或重构类型化 API 路由
- 将原始 Gin handler 改造成 `NinjaAPI` + `Router` + 类型化 operation helper
- 选择合适的请求绑定标签（`path`、`form`、`header`、`cookie`、`json`、`file`）
- 增加中间件、认证、事务、分页、过滤、排序、缓存、版本化、SSE 或 WebSocket 端点
- 保持实现与自动生成的 OpenAPI 文档一致
- 处理错误映射、Context 便捷方法、文件传输、ModelSchema、生命周期、SecuritySchemes、Admin 等框架特性

## 工作规则

1. 对需要文档化的 API 端点，优先使用框架原语，而不是临时拼装原始 Gin 接线。
2. 用独立的 Go 结构体表达请求输入和响应输出，不要手动解析。
3. 把校验和绑定行为放进 struct tag 与 route option，让代码和文档保持同步。
4. 优先复用内置中间件、helper 包和示例模式，而不是新增自定义基础设施。
5. 项目/应用脚手架需求优先交给专用子 skill：[`gin-ninja-scaffold`](../gin-ninja-scaffold/SKILL.md)。

## 使用流程

1. 先识别任务类型：
   - `startproject` / `startapp` 脚手架需求 -> [`gin-ninja-scaffold`](../gin-ninja-scaffold/SKILL.md)
   - 新增或修改接口 -> [API patterns](./references/api-patterns.md)
   - 错误处理或请求上下文使用 -> [Errors and context helpers](./references/errors-and-context.md)
   - 文件传输、ModelSchema、生命周期、安全方案、版本策略或 admin 特性 -> [Advanced features](./references/advanced-features.md)
   - 脚手架之外的示例选型 -> [Scaffolding and examples](./references/scaffolding-and-examples.md)
2. 再确定核心形态：
   - API 根对象 -> `ninja.New(ninja.Config{...})`
   - 路由分组 -> `ninja.NewRouter(...)`
   - 端点 -> `ninja.Get/Post/Put/Patch/Delete/SSE/WebSocket(...)`
3. 定义类型化输入输出结构体，并选择正确的 binding tag 与校验规则。
4. 通过 route/router option 挂摘要、标签、鉴权、事务、分页、缓存、版本化与额外文档响应。
5. 尽量复用现有 middleware、settings、bootstrap、ORM 和 response helper。
6. 通过聚焦测试与文档端点（`/docs`、`/openapi.json`）确认改动正确。

## 仓库地标

- 核心框架：`ninja.go`、`router.go`、`operation.go`、`binding.go`、`openapi.go`
- 高级特性：`cache.go`、`versioning.go`、`stream.go`、`transfer.go`
- 中间件：`middleware/`
- ORM / settings / bootstrap：`orm/`、`settings/`、`bootstrap/`
- 可运行示例：`examples/basic`、`examples/users`、`examples/features`、`examples/admin`、`examples/full`
- CLI 脚手架与迁移：`cmd/gin-ninja-cli/`
- Skill references：`references/api-patterns.md`、`references/scaffolding-and-examples.md`、`references/errors-and-context.md`、`references/advanced-features.md`
- Scaffold 子 skill：`../gin-ninja-scaffold/`
