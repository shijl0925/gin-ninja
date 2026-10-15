# 项目与 CRUD 脚手架

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/scaffolding.md)

小 demo 可以手写 `main.go`。当你要创建真实项目、拆 app、管理迁移或快速生成 CRUD 起步代码时，使用 `gin-ninja-cli`。

## 安装 CLI

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest
```

在当前仓库开发时也可以：

```bash
make install-cli
make build-cli
./bin/gin-ninja-cli --help
```

帮助命令是渐进式的：

```bash
gin-ninja-cli --help
gin-ninja-cli help startproject
gin-ninja-cli startproject -h
```

## 创建第一个项目

最常用的起步方式：

```bash
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
go run .
```

默认模板会创建一个可编译的最小项目，通常包含：

```text
go.mod
main.go
config.yaml
app/models.go
app/migrations.go
app/repos.go
app/schemas.go
app/apis.go
app/routers.go
```

如果你正在做中小型 CRUD 服务，先用默认 `minimal` 模板即可。等需要更多工程文件时，再选择 `standard`：

```bash
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template standard \
  -database postgres \
  -with-tests
```

`standard` 模板会额外生成本地开发和部署相关文件，例如 `Makefile`、`Dockerfile`、`.air.toml`、`settings/config.*.yaml.example`、`bootstrap/` 等。

## 新增 app

项目变大后，可以把 users、orders、billing 等业务拆成 app package：

```bash
gin-ninja-cli startapp blog
```

指定模板或数据库风格：

```bash
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp inventory -template standard -database mysql
```

常用模板：

| 模板 | 适合场景 |
| --- | --- |
| `minimal` | 最短 CRUD 路径，默认推荐 |
| `standard` | 需要更完整项目结构和本地开发文件 |
| `auth` | 需要登录、JWT、权限相关起步代码 |
| `admin` | 显式开启 admin 资源/API 集成 |

常用开关：

| 参数 | 作用 |
| --- | --- |
| `-database sqlite|mysql|postgres|none` | 生成对应数据库配置和驱动导入 |
| `-with-tests` | 生成测试文件 |
| `-with-auth` | 生成认证相关起步代码 |
| `-with-admin` | 生成 admin 相关文件 |
| `-with-gormx` | 使用 gormx 风格 repo/service |
| `-app-dir <path>` | `startproject` 时指定 app 目录 |
| `-force` | 允许覆盖非空输出目录 |

## 迁移工作流

脚手架生成的 app 会包含：

```go
func MigrationModels() []any
```

然后可以使用类似 Django 的迁移命令：

```bash
gin-ninja-cli makemigrations
gin-ninja-cli migrate
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

常见用法：

```bash
gin-ninja-cli makemigrations -app-dir app -name add-users
gin-ninja-cli migrate
gin-ninja-cli migrate zero
```

注意事项：

- `makemigrations` 会读取 `MigrationModels()`，通过 GORM dry-run SQL 生成迁移文件。
- 迁移文件建议提交到仓库，生产环境只执行审查过的 SQL。
- 自动生成的 Down SQL 会保持保守；无法安全推断的回滚需要你手写。

## 生成 CRUD 起步代码

如果已有 model，可以让 CLI 生成一份 CRUD handler、schema 和路由注册代码：

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./app/models.go \
  -output ./app/user_crud_gen.go
```

生成器会创建：

- 请求输入结构体，例如 `ListUsersInput`、`CreateUserInput`
- 显式响应 DTO，例如 `UserOut`
- List/Get/Create/Update/Delete handler
- `RegisterUserCRUDRoutes(router)` 路由注册函数

生成结果是起步骨架，不是最终业务代码。落地时通常还要补权限、事务、复杂校验、唯一性检查、事件发布等业务逻辑。

## 用 tag 控制过滤和排序

在 model 字段上写 `crud:"..."`，生成器会把它转成列表查询能力：

```go
type Project struct {
    ID      uint   `json:"id"`
    Name    string `json:"name" crud:"filter,sort,search"`
    Status  string `json:"status" crud:"filter:like,sort"`
    OwnerID uint   `json:"owner_id" crud:"filter,sort"`
}
```

常用指令：

| tag | 生成效果 |
| --- | --- |
| `crud:"filter"` | 生成等值过滤 |
| `crud:"filter:like"` | 生成模糊过滤 |
| `crud:"sort"` | 加入排序白名单 |
| `crud:"search"` | 加入关键字搜索字段 |

如果生成器能在同一个模型文件里识别关系字段，还会生成 relation output、relation input、`Preload(...)` 和关系同步 helper 的起步代码。

## 使用 preset 配置

多次创建项目时，可以把参数写进 YAML/JSON：

```yaml
name: mysite
module: github.com/acme/mysite
output: ./mysite
app_dir: internal/app
database: postgres
template: standard
with_tests: true
with_gormx: false
```

执行：

```bash
gin-ninja-cli startproject -config ./scaffold.yaml
gin-ninja-cli startapp -config ./scaffold.yaml
```

命令行参数优先级高于配置文件。

上一篇：[Router 与 Operation](./routers-and-operations.md) | 下一篇：[配置、Bootstrap 与生命周期](./configuration.md)
