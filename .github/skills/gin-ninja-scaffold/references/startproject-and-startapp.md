# Startproject and startapp

## Command split

- `gin-ninja-cli startproject <name>` creates a new gin-ninja project scaffold.
- `gin-ninja-cli startapp <name>` creates a new app scaffold inside an existing project.
- If the user only wants interactive prompting, `gin-ninja-cli init` can choose either flow, but direct `startproject` / `startapp` is the clearer default when requirements are already known.

## Template guide

- `minimal` -> smallest scaffold with the least generated surface area
- `standard` -> a fuller default application shape
- `auth` -> includes auth-oriented scaffold files
- `admin` -> includes admin-oriented scaffold files

Choose the smallest template that still covers the requested features.

## Startproject

Use `startproject` when the user needs a fresh service/repository root.

Quick start:

- `gin-ninja-cli startproject mysite -module github.com/acme/mysite`

Common flags:

- `-module <module>` -> Go module path, defaulting to the project name
- `-output <path>` -> output directory, defaulting to the project name
- `-config <path>` -> load YAML/JSON scaffold presets
- `-template <minimal|standard|auth|admin>`
- `-database <sqlite|mysql|postgres|none>`
- `-with-tests`

Advanced overrides:

- `-app-dir <path>`
- `-with-auth`
- `-with-admin`
- `-with-gormx`
- `-force`

## Startapp

Use `startapp` when the project already exists and only a new domain/app package is needed.

Quick start:

- `gin-ninja-cli startapp blog`

Common flags:

- `-output <path>` -> output directory, defaulting to the app name
- `-package <name>` -> override the generated Go package name
- `-model <name>` -> override the generated model name
- `-config <path>` -> load YAML/JSON scaffold presets
- `-template <minimal|standard|auth|admin>`
- `-database <sqlite|mysql|postgres|none>`
- `-with-tests`

Advanced overrides:

- `-with-auth`
- `-with-admin`
- `-with-gormx`
- `-force`

## Decision guide

1. Need a new standalone service root -> `startproject`
2. Need a new module/domain inside an existing service -> `startapp`
3. Need repeatable answers checked into config -> add `-config`
4. Need stronger defaults but not full auth/admin -> start with `standard`
5. Need the lightest output for later manual shaping -> start with `minimal`

## Follow-up workflow

1. Run the CLI first.
2. Inspect the generated layout against the nearest example:
   - `examples/basic` for the smallest runnable shape
   - `examples/users` for app/config/auth wiring
   - `examples/admin` or `examples/full` for admin-heavy setups
3. Only then add custom routes, middleware, models, or business logic.
