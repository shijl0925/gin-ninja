# 配置、Bootstrap 与生命周期

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 配置管理（settings）

```go
import "github.com/shijl0925/gin-ninja/settings"

cfg := settings.MustLoad("config.yaml")
// 或
cfg, err := settings.Load("config.yaml")
```

示例 `config.yaml`：

```yaml
app:
  name: "My API"
  version: "1.0.0"
  env: "production"
  debug: false

server:
  host: "0.0.0.0"
  port: 8080

database:
  driver: "sqlite"
  dsn: "app.db"

cors:
  # 生产环境请保持 origins 显式配置。
  allow_origins:
    - "https://app.example.com"
  allow_methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
  allow_headers: ["Origin", "Content-Type", "Authorization", "X-Request-ID"]
  allow_credentials: false
  max_age_secs: 43200

jwt:
  secret: "change-me-in-production"
  expire_hours: 24

log:
  level: "info"
  format: "json"
  output: "stdout"      # 改成文件路径即可启用文件日志，如 "logs/app.log"
  max_size_mb: 100      # 单文件达到 100 MB 后滚动
  max_age_days: 7       # 旧日志保留 7 天
  max_backups: 3        # 最多保留 3 个滚动文件
  compress: false       # 为 true 时压缩旧日志
```

MySQL / PostgreSQL 可以使用同一个 `database` 配置块：

```yaml
database:
  # MySQL
  driver: "mysql"
  dsn: "root:p%40ss%3Aword@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"

  # 或使用结构化字段，密码中的特殊字符会被安全转义：
  # mysql:
  #   host: "127.0.0.1"
  #   port: 3306
  #   user: "root"
  #   password: "p@ss:word+plus"
  #   name: "gin_ninja"
  #   charset: "utf8mb4"
  #   parse_time: true
  #   loc: "Local"

  # PostgreSQL
  # driver: "postgres"
  # dsn: "host=127.0.0.1 user=postgres password=postgres dbname=gin_ninja port=5432 sslmode=disable TimeZone=Asia/Shanghai"
  # postgres:
  #   host: "127.0.0.1"
  #   port: 5432
  #   user: "postgres"
  #   password: "p@ss word"
  #   name: "gin_ninja"
  #   sslmode: "disable"
  #   time_zone: "Asia/Shanghai"
```

如果仍然提供原始 MySQL DSN，且密码包含 `@`、`:`、`/`、`?`、`#` 或 `+` 等保留字符，请先对密码片段进行 URL 编码。结构化的 `database.mysql` / `database.postgres` 字段可以避免手动转义。

### 配置值中的密码与环境变量占位符

将明文密码写入 `config.yaml` 存在安全风险，尤其在容器化或云部署中文件可能被提交到代码仓库。gin-ninja 支持在任意字符串配置值中使用 Spring 风格的 `${VAR}` / `${VAR:default}` 占位符。YAML 文件解析完成后，每个 token 都会替换为对应环境变量的值。如果变量未设置或为空，则使用第一个 `:` 后面的默认值；省略默认值会让字段变为空字符串。

```yaml
database:
  driver: "postgres"
  # 整条 DSN 来自环境变量，未设置时回退到本地开发连接。
  dsn: "${DATASOURCE_URL:host=localhost user=postgres dbname=myapp sslmode=disable}"

  # 或使用结构化字段，每个凭证对应独立环境变量。
  postgres:
    host:     "${DB_HOST:localhost}"
    user:     "${DB_USER:postgres}"
    password: "${DB_PASSWORD}"          # 无默认值 → 未设置时为空字符串

redis:
  password: "${REDIS_PASSWORD}"

jwt:
  secret: "${JWT_SECRET:change-me-in-production}"
```

单个值中可同时使用多个占位符：

```yaml
database:
  dsn: "${DB_USER:root}:${DB_PASSWORD}@tcp(${DB_HOST:127.0.0.1}:3306)/${DB_NAME:app}"
```

**优先级（由低到高）**

| 来源 | 示例 |
|---|---|
| `config.yaml` 中的字面值 | `password: "fallback"` |
| `${VAR:default}` 中的默认值 | `password: "${DB_PASSWORD:fallback}"` |
| 占位符对应的环境变量 | `DB_PASSWORD=real-pass` |
| 双下划线环境变量覆盖 | `DATABASE__POSTGRES__PASSWORD=top` |

双下划线覆盖（Viper `AutomaticEnv`）最后应用，因此优先级高于占位符。对于没有占位符的字段，也可以用它直接覆盖。

环境变量使用双下划线分隔符覆盖文件配置：

```bash
export SERVER__PORT=9090
export JWT__SECRET=my-secret
```

### 多环境配置合并

对于有环境差异配置的项目，使用 `LoadWithOverrides` 或 `LoadForEnv`。

**`LoadWithOverrides`** – 先加载基础文件，再合并一个或多个覆盖文件。后面的文件优先生效。缺失的覆盖文件会被静默跳过，因此即使某些环境才有该文件，也可以安全地提交覆盖路径。

```go
// 如果存在 config.local.yaml，则合并到 config.yaml 之上。
cfg := settings.MustLoadWithOverrides("config.yaml", "config.local.yaml")
```

**`LoadForEnv`** – 根据 `app.env`（或 `APP__ENV` 环境变量）自动发现并合并对应环境的覆盖文件。

```
config.yaml             ← 基础配置（总是加载）
config.production.yaml  ← app.env=production 时合并
config.staging.yaml     ← app.env=staging 时合并
config.development.yaml ← app.env=development（默认）时合并
```

```go
// 从 config.yaml 读取 app.env，然后合并 config.<env>.yaml。
cfg := settings.MustLoadForEnv("config.yaml")
```

只有覆盖文件中出现的键会被修改；其他键保持基础配置或默认值。

---

## Bootstrap

```go
import (
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/bootstrap"
    _ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
    "github.com/shijl0925/gin-ninja/orm"
    "github.com/shijl0925/gin-ninja/settings"
)

cfg := settings.MustLoad("config.yaml")

log := bootstrap.InitLogger(&cfg.Log)
defer func() { _ = log.Sync() }()

// 初始化数据库。
db := bootstrap.MustInitDB(&cfg.Database)

api := ninja.New(ninja.Config{Settings: cfg})
api.UseGin(orm.Middleware(db))
```

`bootstrap.MustInitDB` 通过注册包解析数据库驱动。请按配置的驱动导入对应包，例如：

- `github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres`

`examples/full/config.yaml` 已包含可直接复制的 MySQL 和 PostgreSQL DSN 示例。

### 解析器变更的边界用例清单

对于任何解析外部字符串的代码（DSN、请求头、query/form 值、filter/sort DSL、版本参数），请验证：

- 协议字符串应作为结构化输入处理，而不是普通文本
- 覆盖特殊字符：`@ : / ? # % + = , ;` 和空格
- 覆盖空值、格式错误、重复输入和大小写混合输入
- 文档示例有对应测试
- 纯解析辅助函数有 fuzz/property 覆盖，避免 panic 和静默重新解释

---

## 生命周期钩子

```go
api := ninja.New(ninja.Config{
    GracefulShutdownTimeout: 15 * time.Second,
    ReadTimeout:             15 * time.Second,
    WriteTimeout:            30 * time.Second,
    IdleTimeout:             60 * time.Second,
})

api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources()
})

log.Fatal(api.Run(":8080"))
```

`Run()` 会在 `SIGINT` / `SIGTERM` 时执行优雅关闭，并且只执行一次 shutdown hooks。
`Serve(listener)` 可用于自定义嵌入和手动关闭编排。

---

上一篇: [项目与 CRUD 脚手架](./scaffolding.md) | 下一篇: [中间件与安全](./middleware-security.md)
