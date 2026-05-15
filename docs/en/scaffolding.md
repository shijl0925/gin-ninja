# Project and CRUD Scaffolding

## Project / App Scaffold Commands

gin-ninja also includes Django-style bootstrap commands for quickly creating a runnable project and new app packages.

The CLI now follows a progressive-help model:

- `gin-ninja-cli --help` shows command groups and recommended entry points
- `gin-ninja-cli help startproject` or `gin-ninja-cli startproject -h` shows full command details
- `gin-ninja-cli init` starts an interactive wizard for new users

The runtime framework stays in the root module, while `cmd/gin-ninja-cli` is maintained as a separate tool module so app builds do not inherit CLI/codegen package boundaries.

Install the CLI into your Go binary directory (`$GOBIN`, or `$GOPATH/bin` when `GOBIN` is unset):

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest

# or install from the cloned repository with Make
make install-cli

# or build only (binary placed at ./bin/gin-ninja-cli)
make build-cli
./bin/gin-ninja-cli --help
```

```bash
gin-ninja-cli --help
gin-ninja-cli help startproject

# small/medium CRUD services: default minimal is usually enough
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
gin-ninja-cli makemigrations
gin-ninja-cli migrate
go run .

# add another app / model package later
gin-ninja-cli startapp blog
gin-ninja-cli makemigrations -app-dir blog -name add-blog-app
gin-ninja-cli migrate

# richer templates / optional features (opt in only when you need them)
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template admin \
  -database postgres \
  -app-dir internal/app \
  -with-tests
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp accounts -template standard -with-gormx -database mysql

# interactive wizard
gin-ninja-cli init

# load a reusable scaffold preset
gin-ninja-cli startproject -config ./scaffold.yaml
gin-ninja-cli startapp -config ./scaffold.yaml
```

`startproject` creates a new directory with:

- `go.mod`
- `main.go`
- `config.yaml`
- `app/models.go`
- `app/migrations.go`
- `app/repos.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

When you opt into `-template standard`, `-template auth`, `-template admin`, or feature flags such as `-with-tests`, the scaffold also adds richer starter files, including:

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

`startapp` creates a new app package directory with the same core CRUD files, and richer templates can additionally generate:

- `migrations.go`
- `scaffold_test.go`
- `auth.go`
- `admin.go`
- `permissions.go`

In practice:

- `minimal` keeps the shortest CRUD path
- `standard` mainly adds project-level infrastructure; when `auth/admin` are not enabled it no longer forces `services.go` / `errors.go`
- `auth` / `admin` templates add the fuller service, error, and permission scaffolding

Useful scaffold flags:

- `-template minimal|standard|auth|admin`
- `-with-tests`
- `-with-auth`
- `-with-admin`
- `-database <sqlite|mysql|postgres|none>` (`startproject` defaults to `sqlite`; `startapp` defaults to `none`; selecting a driver wires the matching registration import)
- `-with-gormx` (default `false`; set it to generate gormx-based repos/services instead of native GORM code)
- `-config <path>` (load scaffold values from a YAML/JSON preset; CLI flags override preset values)
- `-app-dir <path>` (`startproject` only)
- `-force`

Example scaffold preset:

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

Standard-style project scaffolds also ship with an official [air](https://github.com/air-verse/air) preset for hot reload during development:

```bash
cd mysite
make install-air
make dev
```

The generated code is intended as a starting point and compiles as a minimal CRUD-style template; you can then customize models, validation, middleware, routing, and business logic for your own project.

### Database migrations

The CLI also provides Django-style migration commands driven by an app package that exports:

```go
func MigrationModels() []any
```

Generated scaffolds include this function automatically.

```bash
gin-ninja-cli makemigrations [-config ./config.yaml] [-app-dir app] [-name add_users]
gin-ninja-cli migrate [target|zero]
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

- `makemigrations` captures the SQL emitted by GORM `AutoMigrate` in dry-run mode and writes a timestamped SQL migration under `migrations/`; run it in development or CI because it requires the Go toolchain to inspect `MigrationModels()`
- `migrate` applies pending migrations, migrates to a target migration, or rolls everything back with `zero`
- `showmigrations` lists all migration files and whether they have been applied
- `sqlmigrate` prints the generated SQL for a migration (`-direction up|down|all`)

For production and test deployments, prefer shipping reviewed migration files and running `gin-ninja-cli migrate` against those generated SQL migrations. Automatically generated down SQL is intentionally conservative: when the CLI cannot parse a simple table, index, column, or constraint change with high confidence, it marks the migration as irreversible so you can provide a hand-written rollback. A future app-side migration generator may replace the temporary Go helper used by `makemigrations` to reduce environment sensitivity.

---

## CRUD Scaffold Generator

gin-ninja now includes a small scaffolding CLI for generating model-based CRUD boilerplate.

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./examples/full/app/models.go \
  -output ./examples/full/app/user_crud_gen.go
```

The generator:

- reads a Go model struct from the provided file
- creates request/response schemas and CRUD handlers in the same package
- generates a `Register<Model>CRUDRoutes(router)` helper for route registration
- uses `PATCH /:id` for generated partial-update handlers instead of advertising partial updates as `PUT`
- can generate list filter / sort / keyword-search inputs from model `crud:"..."` tags
- can detect same-file belongs-to / has-many / many-to-many relations and generate preload, relation input, and relation output scaffolding

Generated code is intended as a starting point. Review the scaffold and adjust validation, persistence rules, permissions, and router composition for your application.

### CRUD generator tags

Use the `crud:"..."` tag on model fields to opt into generated query inputs:

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

Supported generator directives:

- `crud:"filter"` → adds a generated list field with `filter:"column,eq"`
- `crud:"filter:like"` → adds a generated list field with `filter:"column,like"`
- `crud:"sort"` → includes the field in generated `Sort string \`order:"..."\``
- `crud:"search"` → includes the field in generated keyword search

The generated list handler wires these into `filter.BuildOptions(...)` and `order.ApplyOrder(...)` automatically.

### Generated relation support

When the generator can resolve related models from the same model file, it now scaffolds relation-aware CRUD output and loading:

- `belongs to` → generates nested relation output plus scalar relation input when needed
- `has many` / `many2many` → generates nested relation output plus `...IDs` input fields
- generated list/detail loads automatically include `Preload(...)`
- generated relation helpers keep association syncing logic out of the handler body

For example, a generated scaffold can now emit:

- nested response fields such as `Owner *ProjectOwnerOut`, `Tasks []ProjectTasksOut`
- relation inputs such as `TagsIDs []uint`
- association helpers such as `syncProjectTagsRelations(...)`

---
