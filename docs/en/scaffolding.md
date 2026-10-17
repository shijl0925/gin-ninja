# Project and CRUD Scaffolding

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/scaffolding.md)

For a small demo, a handwritten `main.go` is enough. When you need a real project layout, app packages, migrations, or CRUD starter code, use `gin-ninja-cli`.

## Install The CLI

```bash
go install github.com/shijl0925/gin-ninja/cmd/gin-ninja-cli@latest
```

When developing from this repository:

```bash
make install-cli
make build-cli
./bin/gin-ninja-cli --help
```

Help is progressive:

```bash
gin-ninja-cli --help
gin-ninja-cli help startproject
gin-ninja-cli startproject -h
```

## Create The First Project

Common starting point:

```bash
gin-ninja-cli startproject mysite -module github.com/acme/mysite
cd mysite
go run .
```

The default template creates a compile-ready minimal project, usually including:

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

For small and medium CRUD services, start with the default `minimal` template. Choose `standard` once you want more infrastructure files:

```bash
gin-ninja-cli startproject mysite \
  -module github.com/acme/mysite \
  -template standard \
  -database postgres \
  -with-tests
```

The `standard` template adds local development and deployment files such as `Makefile`, `Dockerfile`, `.air.toml`, `settings/config.*.yaml.example`, and `bootstrap/`.

## Add An App

As the project grows, split users, orders, billing, and similar domains into app packages:

```bash
gin-ninja-cli startapp blog
```

Choose templates or database style:

```bash
gin-ninja-cli startapp accounts -template auth -with-tests
gin-ninja-cli startapp inventory -template standard -database mysql
```

Templates:

| Template | Use when |
| --- | --- |
| `minimal` | shortest CRUD path, recommended default |
| `standard` | fuller project structure and local development files |
| `auth` | login, JWT, and permission starter code |
| `admin` | explicit admin resource/API integration |

Common flags:

| Flag | Purpose |
| --- | --- |
| `-database sqlite|mysql|postgres|none` | generate matching database config and driver imports |
| `-with-tests` | generate tests |
| `-with-auth` | generate auth starter code |
| `-with-admin` | generate admin files |
| `-app-dir <path>` | choose app directory for `startproject` |
| `-force` | allow a non-empty output directory |

## Migration Workflow

Generated apps include:

```go
func MigrationModels() []any
```

Then use Django-like migration commands:

```bash
gin-ninja-cli makemigrations
gin-ninja-cli migrate
gin-ninja-cli showmigrations
gin-ninja-cli sqlmigrate 20260417120000_add_users
```

Common usage:

```bash
gin-ninja-cli makemigrations -app-dir app -name add-users
gin-ninja-cli migrate
gin-ninja-cli migrate zero
```

Notes:

- `makemigrations` reads `MigrationModels()` and uses GORM dry-run SQL to generate migration files.
- Commit migration files to the repository and run reviewed SQL in production.
- Generated Down SQL is conservative. When rollback cannot be inferred safely, write it manually.

## Generate CRUD Starter Code

If you already have a model, generate CRUD handlers, schemas, and route registration code:

```bash
gin-ninja-cli generate crud \
  -model User \
  -model-file ./app/models.go \
  -output ./app/user_crud_gen.go
```

The generator creates:

- request input structs such as `ListUsersInput` and `CreateUserInput`
- explicit response DTOs such as `UserOut`
- List/Get/Create/Update/Delete handlers
- `RegisterUserCRUDRoutes(router)`

Generated code is a starter skeleton, not final business code. You will usually add permissions, transactions, complex validation, uniqueness checks, events, and domain logic.

## Control Filtering And Sorting With Tags

Add `crud:"..."` to model fields to influence generated list queries:

```go
type Project struct {
    ID      uint   `json:"id"`
    Name    string `json:"name" crud:"filter,sort,search"`
    Status  string `json:"status" crud:"filter:like,sort"`
    OwnerID uint   `json:"owner_id" crud:"filter,sort"`
}
```

Common directives:

| Tag | Effect |
| --- | --- |
| `crud:"filter"` | generate equality filtering |
| `crud:"filter:like"` | generate partial-match filtering |
| `crud:"sort"` | add the field to the sort allowlist |
| `crud:"search"` | add the field to keyword search |

When the generator can identify relation fields in the same model file, it also emits relation output, relation input, `Preload(...)`, and relation synchronization helpers.

## Use A Preset

Put repeated options in YAML/JSON:

```yaml
name: mysite
module: github.com/acme/mysite
output: ./mysite
app_dir: internal/app
database: postgres
template: standard
with_tests: true
```

Run:

```bash
gin-ninja-cli startproject -config ./scaffold.yaml
gin-ninja-cli startapp -config ./scaffold.yaml
```

Command-line flags take precedence over the config file.

Previous: [Routers and Operations](./routers-and-operations.md) | Next: [Configuration, Bootstrap, and Lifecycle](./configuration.md)
