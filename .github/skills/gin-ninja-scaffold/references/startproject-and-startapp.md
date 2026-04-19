# Startproject and startapp

## Command split

- `gin-ninja-cli startproject <name>` creates a new gin-ninja project scaffold.
- `gin-ninja-cli startapp <name>` creates a new app scaffold inside an existing project.
- If the user only wants interactive prompting, `gin-ninja-cli init` can choose either flow, but direct `startproject` / `startapp` is the clearer default when requirements are already known.

## Template guide

Templates fall into two primary tiers:

- `minimal` → **default and recommended** — smallest CRUD starter with the least generated surface area; start here unless you have a specific reason not to
- `full` → full-stack scenario template (auth + admin + project infrastructure); use only when you need all of it

The `standard` template is an intermediate option: it adds settings/config files and `.air.toml` hot-reload support, but omits the heavy infrastructure (`cmd/server`, `internal/server`, `bootstrap`, Docker, Make) that is reserved for `full`.

The `auth` and `admin` templates are backward-compatible aliases for `full` and continue to work.

**Rule:** always start with `minimal`; promote to `standard` or `full` only when you have a clear need.

Flag restrictions:
- `-with-admin` requires at least `standard` (rejected on `minimal`)
- `-with-auth` on `minimal` adds auth app files without changing project structure

## Startproject

Use `startproject` when the user needs a fresh service/repository root.

Quick start:

- `gin-ninja-cli startproject mysite -module github.com/acme/mysite`

Common flags:

- `-module <module>` → Go module path, defaulting to the project name
- `-output <path>` → output directory, defaulting to the project name
- `-config <path>` → load YAML/JSON scaffold presets
- `-template <minimal|standard|full>` (primary choices; `auth`/`admin` are compat aliases for `full`)
- `-with-tests`

Advanced overrides:

- `-app-dir <path>`
- `-with-auth` (requires `standard` or `full`)
- `-with-admin` (requires `standard` or `full`; also enables auth)
- `-with-gormx`
- `-force`

## Startapp

Use `startapp` when the project already exists and only a new domain/app package is needed.

Quick start:

- `gin-ninja-cli startapp blog`

Common flags:

- `-output <path>` → output directory, defaulting to the app name
- `-package <name>` → override the generated Go package name
- `-model <name>` → override the generated model name
- `-config <path>` → load YAML/JSON scaffold presets
- `-template <minimal|standard|full>` (primary choices; `auth`/`admin` are compat aliases for `full`)
- `-with-tests`

Advanced overrides:

- `-with-auth` (requires `standard` or `full`)
- `-with-admin` (requires `standard` or `full`; also enables auth)
- `-with-gormx`
- `-force`

## Decision guide

1. Need a new standalone service root → `startproject`
2. Need a new module/domain inside an existing service → `startapp`
3. Need repeatable answers checked into config → add `-config`
4. Need settings/dev tooling without heavy infra → use `standard`
5. Need the lightest output for later manual shaping → use `minimal` (default)
6. Need auth + admin + full project infrastructure → use `full`

## Follow-up workflow

1. Run the CLI first.
2. Inspect the generated layout against the nearest example:
   - `examples/basic` for the smallest runnable shape
   - `examples/users` for app/config/auth wiring
   - `examples/admin` or `examples/full` for admin-heavy setups
3. Only then add custom routes, middleware, models, or business logic.
