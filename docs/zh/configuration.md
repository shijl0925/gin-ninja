# 配置、Bootstrap 与生命周期

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/configuration.md)

项目从 demo 走向真实应用时，通常需要把端口、数据库、JWT、CORS、日志等配置从代码里挪出来。gin-ninja 提供 `settings` 负责加载配置，`bootstrap` 负责初始化日志和数据库。

## 最小配置文件

创建 `config.yaml`：

```yaml
app:
  name: "My API"
  version: "1.0.0"
  env: "development"
  debug: true

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 60
  write_timeout: 60

database:
  driver: "sqlite"
  dsn: "app.db"

jwt:
  secret: "${JWT_SECRET:change-me-in-development}"
  expire_hours: 24
  issuer: "gin-ninja"

cors:
  allow_origins: ["http://localhost:3000", "http://localhost:5173"]
  allow_methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
  allow_headers: ["Origin", "Content-Type", "Authorization", "X-Request-ID"]
  allow_credentials: false
  max_age_secs: 43200

log:
  level: "info"
  format: "json"
  output: "stdout"
```

加载配置：

```go
cfg := settings.MustLoad("config.yaml")
```

如果你想自己处理错误：

```go
cfg, err := settings.Load("config.yaml")
if err != nil {
    return err
}
```

## 接入应用

```go
import (
    "fmt"
    "log"
    "time"

    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/bootstrap"
    _ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
    "github.com/shijl0925/gin-ninja/middleware"
    "github.com/shijl0925/gin-ninja/orm"
    "github.com/shijl0925/gin-ninja/settings"
)

func main() {
    cfg := settings.MustLoad("config.yaml")

    logger := bootstrap.InitLogger(&cfg.Log)
    defer func() { _ = logger.Sync() }()

    db := bootstrap.MustInitDB(&cfg.Database)

    api := ninja.New(ninja.Config{
        Title:        cfg.App.Name,
        Version:      cfg.App.Version,
        Settings:     cfg,
        ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
        WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
    })

    api.UseGin(
        middleware.RequestID(),
        middleware.CORSFromConfig(cfg.CORS),
        orm.Middleware(db),
    )

    log.Fatal(api.Run(fmt.Sprintf(":%d", cfg.Server.Port)))
}
```

数据库驱动通过空导入注册。按实际配置选择一个或多个：

```go
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql"
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres"
```

## 数据库配置

SQLite：

```yaml
database:
  driver: "sqlite"
  dsn: "app.db"
```

MySQL 可以写 DSN：

```yaml
database:
  driver: "mysql"
  dsn: "root:password@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"
```

也可以写结构化字段，密码里的特殊字符会安全处理：

```yaml
database:
  driver: "mysql"
  mysql:
    host: "127.0.0.1"
    port: 3306
    user: "root"
    password: "${DB_PASSWORD}"
    name: "gin_ninja"
    charset: "utf8mb4"
    parse_time: true
    loc: "Local"
```

PostgreSQL：

```yaml
database:
  driver: "postgres"
  postgres:
    host: "127.0.0.1"
    port: 5432
    user: "postgres"
    password: "${DB_PASSWORD}"
    name: "gin_ninja"
    sslmode: "disable"
    time_zone: "Asia/Shanghai"
```

## 环境变量

任意字符串配置都支持 `${VAR}` 和 `${VAR:default}`：

```yaml
database:
  dsn: "${DATASOURCE_URL:app.db}"

jwt:
  secret: "${JWT_SECRET}"
```

也可以用双下划线覆盖任意配置项：

```bash
export SERVER__PORT=9090
export JWT__SECRET=real-secret
export DATABASE__POSTGRES__PASSWORD=top-secret
```

优先级从低到高：

| 来源 | 示例 |
| --- | --- |
| YAML 字面值 | `secret: "dev"` |
| 占位符默认值 | `${JWT_SECRET:dev}` |
| 占位符环境变量 | `JWT_SECRET=real` |
| 双下划线覆盖 | `JWT__SECRET=real` |

生产环境不要把真实密码和 JWT secret 提交到仓库。

## 多环境配置

如果你有本地、测试、生产三套配置，可以把基础配置和环境差异拆开：

```text
config.yaml
config.development.yaml
config.staging.yaml
config.production.yaml
```

按 `app.env` 自动合并：

```go
cfg := settings.MustLoadForEnv("config.yaml")
```

或者手动指定覆盖文件：

```go
cfg := settings.MustLoadWithOverrides("config.yaml", "config.local.yaml")
```

后面的文件优先级更高；不存在的覆盖文件会被跳过。

## 生命周期钩子

启动时预热缓存、关闭时释放资源，可以使用生命周期钩子：

```go
api := ninja.New(ninja.Config{
    GracefulShutdownTimeout: 15 * time.Second,
})

api.OnStartup(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return warmCache(ctx)
})

api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
    return closeResources()
})

log.Fatal(api.Run(":8080"))
```

`Run()` 收到 `SIGINT` 或 `SIGTERM` 时会执行优雅关闭，并只执行一次 shutdown hooks。需要嵌入自定义 server 时，可以使用 `Serve(listener)` 和 `Shutdown(ctx)`。

## 排查配置解析问题

改动 DSN、请求头、query/form、filter/sort 或版本参数解析时，建议覆盖这些情况：

- 特殊字符：`@ : / ? # % + = , ;` 和空格
- 空值、重复值、格式错误和大小写混合
- 密码字段是否经过正确转义
- 示例配置是否有测试或可运行示例覆盖

上一篇：[项目与 CRUD 脚手架](./scaffolding.md) | 下一篇：[中间件与安全](./middleware-security.md)
