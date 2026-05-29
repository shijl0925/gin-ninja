# 项目与 CRUD 脚手架

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 项目 / 应用脚手架命令

gin-ninja 也提供了类似 Django 的脚手架命令，可快速创建可运行的项目骨架和新的 app 包。

CLI 现在采用渐进式帮助：

- `gin-ninja-cli --help` 只展示命令分组和推荐入口
- `gin-ninja-cli help startproject` 或 `gin-ninja-cli startproject -h` 再查看完整参数
- `gin-ninja-cli init` 可通过交互式向导完成首次创建

运行时框架继续保留在根模块，`cmd/gin-ninja-cli` 则作为独立工具模块维护，这样应用构建时不会把 CLI / codegen 视为运行时模块边界的一部分。

CLI 会安装到 Go 的可执行目录（优先使用 `$GOBIN`，未设置时使用 `$GOPATH/bin`）：

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest

# 或者通过 Make 安装到当前 Go 的可执行目录
make install-cli

# 或者只在仓库本地构建（产物位于 ./bin/gin-ninja-cli）
make build-cli
./bin/gin-ninja-cli --help
```

```bash
gin-ninja-cli --help
gin-ninja-cli help startproject

# 小中型 CRUD 项目：默认 minimal 即可
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
gin-ninja-cli makemigrations
gin-ninja-cli migrate
go run .

# 后续新增 app / 模型包
gin-ninja-cli startapp blog
gin-ninja-cli makemigrations -app-dir blog -name add-blog-app
gin-ninja-cli migrate

# 更丰富的模板 / 可选功能（只有确实需要时再开启）
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template admin \
  -database postgres \
  -app-dir internal/app \
  -with-tests
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp accounts -template standard -with-gormx -database mysql

# 交互式向导
gin-ninja-cli init

# 复用脚手架 preset 配置
gin-ninja-cli startproject -config ./scaffold.yaml
gin-ninja-cli startapp -config ./scaffold.yaml
```

`startproject` 会创建一个新目录，包含：

- `go.mod`
- `main.go`
- `config.yaml`
- `app/models.go`
- `app/migrations.go`
- `app/repos.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

当你启用 `-template standard`、`-template auth`、`-template admin`，或 `-with-tests` 等功能开关时，脚手架还会额外生成更完整的起步文件，例如：

- `.air.toml`
- `cmd/server/main.go`
- `internal/server/server.go`
- `bootstrap/db.go`
- `bootstrap/logger.go`
- `bootstrap/cache.go`
- `settings/config.local.yaml.example`
- `settings/config.prod.yaml.example`
- `.env.example`
- `Makefile`
- `Dockerfile`
- `docker-compose.yml`
- `README.md`
- `migrations/.gitkeep`
- `scripts/.gitkeep`

`startapp` 会在新的 app package 目录中生成相同的核心 CRUD 文件；更丰富的模板还会额外生成：

- `migrations.go`
- `scaffold_test.go`
- `auth.go`
- `admin.go`
- `permissions.go`

其中：

- 默认 `minimal` 只保留最短 CRUD 路径
- `standard` 主要增加项目级基础设施文件；当未启用 `auth/admin` 时，不再强制生成 `services.go` / `errors.go`
- `auth` / `admin` 模板会额外生成更完整的 service / error / 权限相关代码

常用脚手架参数：

- `-template minimal|standard|auth|admin`
- `-with-tests`
- `-with-auth`
- `-with-admin`
- `-database <sqlite|mysql|postgres|none>`（`startproject` 默认 `sqlite`；`startapp` 默认 `none`；选中驱动时会自动生成对应注册导入）
- `-with-gormx`（默认 `false`；显式开启后生成基于 gormx 的 repo/service，而不是原生 GORM 代码）
- `-config <path>`（从 YAML/JSON preset 加载脚手架参数；命令行参数优先生效）
- `-app-dir <path>`（仅 `startproject` 支持）
- `-force`

preset 配置示例：

```yaml
name: mysite
module: github.com/acme/mysite
output: ./mysite
app_dir: internal/app
database: postgres
template: admin
with_tests: true
with_gormx: false
```

标准风格项目脚手架还会内置官方 [air](https://github.com/air-verse/air) 预设，方便本地热重载开发：

```bash
cd mysite
make install-air
make dev
```

生成的代码定位为起步骨架，能够作为最小 CRUD 风格模板直接编译；后续你仍可按业务需要继续补充模型、校验、中间件、路由和业务逻辑。

### 数据库迁移命令

CLI 也支持类似 Django 的数据库迁移工作流。对应的 app package 需要导出：

```go
func MigrationModels() []any
```

脚手架生成的 app 已经默认包含该函数。

```bash
gin-ninja-cli makemigrations [-config ./config.yaml] [-app-dir app] [-name add_users]
gin-ninja-cli migrate [target|zero]
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

- `makemigrations` 会通过 GORM `AutoMigrate` 的 dry-run SQL 生成时间戳迁移文件，并写入 `migrations/`；它需要 Go 工具链来检查 `MigrationModels()`，建议只在开发或 CI 环境运行
- `migrate` 会应用未执行迁移、迁移到指定版本，或通过 `zero` 回滚全部迁移
- `showmigrations` 会列出所有迁移及其是否已执行
- `sqlmigrate` 会输出指定迁移的 SQL（可通过 `-direction up|down|all` 控制）

生产和测试部署建议随应用发布已经审查过的迁移文件，并只运行 `gin-ninja-cli migrate` 执行这些 SQL 迁移。自动生成的 Down SQL 会保持保守：当 CLI 无法高置信解析简单表、索引、列或约束变更时，会将迁移标记为不可自动回滚，便于你手写回滚脚本。后续版本可考虑由应用侧提供迁移生成入口，替代 `makemigrations` 当前使用的临时 Go helper，降低环境敏感性。

## CRUD 脚手架生成器

gin-ninja 现在内置了一个轻量级脚手架 CLI，可基于模型结构体生成 CRUD 接口代码骨架。

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./examples/full/app/models.go \
  -output ./examples/full/app/user_crud_gen.go
```

该生成器会：

- 读取指定文件中的 Go 模型结构体
- 在同一 package 下生成请求/响应结构和 CRUD handler
- 生成 `Register<Model>CRUDRoutes(router)` 路由注册辅助函数
- 对“部分更新”生成 `PATCH /:id` 路由，而不是用 `PUT` 表达部分更新语义
- 可从模型字段的 `crud:"..."` tag 自动生成列表过滤 / 排序 / 关键字搜索输入
- 可识别同一模型文件中的 belongs-to / has-many / many2many 关系，并生成 preload、relation input、relation output 骨架

生成结果定位为“起步骨架”。落地时仍建议根据业务继续补充校验、权限、事务、查询条件和路由组织方式。

### CRUD 生成器 tag 规则

可以在模型字段上声明 `crud:"..."`，控制生成器产出的查询输入：

```go
type Project struct {
    ID      uint   `json:"id"`
    Name    string `json:"name" crud:"filter,sort,search"`
    Status  string `json:"status" crud:"filter:like,sort,search"`
    OwnerID uint   `json:"owner_id" crud:"filter,sort"`
    Owner   User   `gorm:"foreignKey:OwnerID" json:"-"`
    Tasks   []Task `gorm:"foreignKey:ProjectID" json:"-"`
    Tags    []Tag  `gorm:"many2many:project_tags;" json:"-"`
}
```

目前支持的生成指令：

- `crud:"filter"`：生成 `filter:"column,eq"` 风格的列表过滤字段
- `crud:"filter:like"`：生成 `filter:"column,like"` 风格的模糊过滤字段
- `crud:"sort"`：把该字段加入生成的 `Sort string \`order:"..."\`` 白名单
- `crud:"search"`：把该字段加入生成的关键字搜索

生成出来的列表 handler 会自动接入：

- `filter.BuildOptions(...)`
- `order.ApplyOrder(...)`

### 生成的关系字段支持

当生成器能在同一个模型文件里解析到关联模型时，会自动补充 relation-aware 的 CRUD 骨架：

- `belongs to`：生成嵌套 relation output，并在需要时生成标量 relation input
- `has many` / `many2many`：生成嵌套 relation output，以及 `...IDs` 形式的 relation input
- 生成的列表 / 详情加载会自动带上 `Preload(...)`
- 自动生成关系同步 helper，减少 handler 中的关联处理样板代码

例如，生成结果现在可以包含：

- `Owner *ProjectOwnerOut`
- `Tasks []ProjectTasksOut`
- `TagsIDs []uint`
- `syncProjectTagsRelations(...)`

---

上一篇: [快速开始](./getting-started.md) | 下一篇: [配置、Bootstrap 与生命周期](./configuration.md)
