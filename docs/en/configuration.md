# Configuration, Bootstrap, and Lifecycle

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/configuration.md)

When a demo becomes a real application, move port, database, JWT, CORS, and logging settings out of code. `settings` loads configuration, and `bootstrap` initializes logging and databases.

## Minimal Config File

Create `config.yaml`:

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

Load it:

```go
cfg := settings.MustLoad("config.yaml")
```

Or handle the error yourself:

```go
cfg, err := settings.Load("config.yaml")
if err != nil {
    return err
}
```

## Wire It Into The App

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

Database drivers are registered with blank imports. Import the drivers you use:

```go
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql"
_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres"
```

## Database Config

SQLite:

```yaml
database:
  driver: "sqlite"
  dsn: "app.db"
```

MySQL with a DSN:

```yaml
database:
  driver: "mysql"
  dsn: "root:password@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"
```

Or structured MySQL fields, which safely handle special characters in passwords:

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

PostgreSQL:

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

## Environment Variables

Any string setting supports `${VAR}` and `${VAR:default}`:

```yaml
database:
  dsn: "${DATASOURCE_URL:app.db}"

jwt:
  secret: "${JWT_SECRET}"
```

You can also override any key with double underscores:

```bash
export SERVER__PORT=9090
export JWT__SECRET=real-secret
export DATABASE__POSTGRES__PASSWORD=top-secret
```

Precedence, from low to high:

| Source | Example |
| --- | --- |
| YAML literal | `secret: "dev"` |
| placeholder default | `${JWT_SECRET:dev}` |
| placeholder environment variable | `JWT_SECRET=real` |
| double-underscore override | `JWT__SECRET=real` |

Do not commit real database passwords or JWT/session secrets.

## Multi-Environment Config

Split base config from environment differences:

```text
config.yaml
config.development.yaml
config.staging.yaml
config.production.yaml
```

Merge based on `app.env`:

```go
cfg := settings.MustLoadForEnv("config.yaml")
```

Or provide override files explicitly:

```go
cfg := settings.MustLoadWithOverrides("config.yaml", "config.local.yaml")
```

Later files have higher precedence. Missing override files are skipped.

## Lifecycle Hooks

Use lifecycle hooks for cache warmup, background setup, and graceful resource cleanup:

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

`Run()` handles `SIGINT` and `SIGTERM`, runs graceful shutdown, and executes shutdown hooks once. For custom server embedding, use `Serve(listener)` and `Shutdown(ctx)`.

## Parser Troubleshooting Checklist

When changing DSN, header, query/form, filter/sort, or version parameter parsing, cover:

- special characters: `@ : / ? # % + = , ;` and spaces
- empty values, duplicates, malformed values, and mixed case
- whether password fields are escaped correctly
- whether examples have tests or runnable coverage

Previous: [Project and CRUD Scaffolding](./scaffolding.md) | Next: [Middleware and Security](./middleware-security.md)
