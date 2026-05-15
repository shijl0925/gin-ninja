# 配置、Bootstrap 与 ORM

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 配置管理（settings）

```go
import "github.com/shijl0925/gin-ninja/settings"

cfg := settings.MustLoad("config.yaml")
// 或
cfg := settings.MustLoadForEnv("config.yaml")
```

示例配置：

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

补充说明：

- 环境变量使用双下划线覆盖配置，例如：`SERVER__PORT=9090`
- `MustLoadWithOverrides` 支持加载基础配置后再叠加覆盖文件
- `MustLoadForEnv` 会根据 `app.env` 自动合并 `config.<env>.yaml`
- MySQL/PostgreSQL 既支持原始 DSN，也支持结构化字段配置

### 配置值中的密码与环境变量占位符

将明文密码写入 `config.yaml` 存在安全风险，尤其在容器化或云原生部署中文件可能被提交到代码仓库。gin-ninja 支持在任意字符串配置值中使用 Spring 风格的 `${VAR}` / `${VAR:默认值}` 占位符。YAML 文件解析完成后，框架会自动将每个占位符替换为对应的环境变量值；若环境变量未设置或为空，则使用 `:` 后面的默认值（不写默认值则字段变为空字符串）。

```yaml
database:
  driver: "postgres"
  # 整条 DSN 来自环境变量，未设置时回退到本地开发连接。
  dsn: "${DATASOURCE_URL:host=localhost user=postgres dbname=myapp sslmode=disable}"

  # 也可以用结构化字段，每个凭证对应独立的环境变量。
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
| 占位符内的默认值 | `password: "${DB_PASSWORD:fallback}"` |
| 占位符对应的环境变量 | `DB_PASSWORD=real-pass` |
| 双下划线环境变量覆盖 | `DATABASE__POSTGRES__PASSWORD=top` |

双下划线覆盖（Viper `AutomaticEnv`）优先级最高，在占位符展开之后生效。对于没有占位符的字段，仍可通过双下划线环境变量直接覆盖。

## Bootstrap 与 ORM

```go
import (
    "github.com/shijl0925/gin-ninja/bootstrap"
    _ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
    "github.com/shijl0925/gin-ninja/orm"
)

cfg := settings.MustLoad("config.yaml")
log := bootstrap.InitLogger(&cfg.Log)
defer func() { _ = log.Sync() }()

db := bootstrap.MustInitDB(&cfg.Database)
orm.Init(db)
```

- `bootstrap.MustInitDB` 通过驱动注册包解析数据库驱动，按需引入即可，例如：
  - `github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite`
  - `github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql`
  - `github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres`
- `orm.Middleware(db)` 可把数据库句柄注入请求上下文
- 事务场景可以在操作上使用 `ninja.WithTransaction()`

## 生命周期钩子

```go
api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources()
})
```

`Run()` 会处理 `SIGINT` / `SIGTERM` 并执行优雅关闭。

---

上一篇: [核心 API、绑定与响应](./core-api.md) | 下一篇: [中间件与安全](./middleware-security.md)
