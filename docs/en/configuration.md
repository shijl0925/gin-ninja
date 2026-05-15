# Configuration, Bootstrap, and Lifecycle

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/README.md)

## Configuration (settings)

```go
import "github.com/shijl0925/gin-ninja/settings"

cfg := settings.MustLoad("config.yaml")
// or
cfg, err := settings.Load("config.yaml")
```

Sample `config.yaml`:

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
  # Keep origins explicit in production.
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
  output: "stdout"      # set a file path such as "logs/app.log" to enable file logging
  max_size_mb: 100      # rotate after the file reaches 100 MB
  max_age_days: 7       # keep rotated files for 7 days
  max_backups: 3        # keep up to 3 rotated files
  compress: false       # gzip old rotated files when true
```

MySQL / PostgreSQL can use the same `database` block:

```yaml
database:
  # MySQL
  driver: "mysql"
  dsn: "root:p%40ss%3Aword@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"

  # Or use structured fields so special characters in passwords are escaped safely:
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

If you still provide a raw MySQL DSN and the password contains reserved characters such as `@`, `:`, `/`, `?`, `#`, or `+`, URL-encode the password segment first. Structured `database.mysql` / `database.postgres` fields avoid that manual escaping step.

### Secrets and environment-variable placeholders

Storing plaintext passwords in `config.yaml` is a security risk, especially in
containerised or cloud deployments where the file may be committed to source control.
gin-ninja supports Spring-style `${VAR}` / `${VAR:default}` placeholders in any string
config value.  After the YAML file is parsed, every token is replaced by the value of
the named environment variable.  If the variable is unset or empty the text after the
first `:` is used as the default; omitting the default causes the field to become an
empty string.

```yaml
database:
  driver: "postgres"
  # Entire DSN from environment; fall back to a local dev connection if unset.
  dsn: "${DATASOURCE_URL:host=localhost user=postgres dbname=myapp sslmode=disable}"

  # Or use structured fields – each credential can come from its own variable.
  postgres:
    host:     "${DB_HOST:localhost}"
    user:     "${DB_USER:postgres}"
    password: "${DB_PASSWORD}"          # no default → empty string when unset

redis:
  password: "${REDIS_PASSWORD}"

jwt:
  secret: "${JWT_SECRET:change-me-in-production}"
```

Multiple placeholders in a single value are supported:

```yaml
database:
  dsn: "${DB_USER:root}:${DB_PASSWORD}@tcp(${DB_HOST:127.0.0.1}:3306)/${DB_NAME:app}"
```

**Precedence (lowest → highest)**

| Source | Example |
|---|---|
| `config.yaml` default value | `password: "fallback"` |
| `${VAR:default}` default | `password: "${DB_PASSWORD:fallback}"` |
| Env var named in placeholder | `DB_PASSWORD=real-pass` |
| Double-underscore env override | `DATABASE__POSTGRES__PASSWORD=top` |

Double-underscore overrides (Viper `AutomaticEnv`) are applied last and therefore
take precedence over placeholders.  Use them when you need to override a key that
does not already contain a placeholder.

Environment variables override file settings using double-underscore separators:
```bash
export SERVER__PORT=9090
export JWT__SECRET=my-secret
```

### Multi-environment config merging

For projects with environment-specific settings, use `LoadWithOverrides` or `LoadForEnv`.

**`LoadWithOverrides`** – loads a base file then merges one or more override files.  Later files
win.  Missing override files are silently skipped, so it is safe to commit the override path even
when the file only exists in certain environments.

```go
// Merges config.local.yaml on top of config.yaml, if it exists.
cfg := settings.MustLoadWithOverrides("config.yaml", "config.local.yaml")
```

**`LoadForEnv`** – automatically discovers and merges the environment-specific override file based
on `app.env` (or the `APP__ENV` environment variable).

```
config.yaml          ← base (always loaded)
config.production.yaml ← merged when app.env=production
config.staging.yaml  ← merged when app.env=staging
config.development.yaml ← merged when app.env=development (default)
```

```go
// Reads app.env from config.yaml, then merges config.<env>.yaml.
cfg := settings.MustLoadForEnv("config.yaml")
```

Only keys present in the override file are changed; all other keys keep their base or default values.

---

## Bootstrap

```go
import (
    "github.com/shijl0925/gin-ninja/bootstrap"
    _ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
    "github.com/shijl0925/gin-ninja/orm"
)

cfg := settings.MustLoad("config.yaml")

log := bootstrap.InitLogger(&cfg.Log)
defer func() { _ = log.Sync() }()

// Initialise database.
db := bootstrap.MustInitDB(&cfg.Database)
orm.Init(db)
```

`bootstrap.MustInitDB` resolves drivers through registration packages. Import the matching package for the driver you configure, for example:

- `github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/mysql`
- `github.com/shijl0925/gin-ninja/bootstrap/drivers/postgres`

`examples/full/config.yaml` already includes ready-to-copy MySQL and PostgreSQL DSN examples.

### Boundary-case checklist for parser changes

For any code that parses external strings (DSN, headers, query/form values, filter/sort DSL, version params), verify:

- protocol strings are treated as structured input, not generic text
- special characters are covered: `@ : / ? # % + = , ;` and spaces
- empty, malformed, repeated, and mixed-case inputs are tested
- documentation examples have matching tests
- pure parsing helpers have fuzz/property coverage to guard against panics and silent reinterpretation

---

## Lifecycle Hooks

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

`Run()` performs graceful shutdown on `SIGINT` / `SIGTERM` and executes shutdown hooks once.
`Serve(listener)` is available for custom embedding and manual shutdown orchestration.

---

Previous: [Project and CRUD Scaffolding](./scaffolding.md) | Next: [Middleware and Security](./middleware-security.md)
