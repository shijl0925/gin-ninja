---
name: gin-ninja-scaffold
description: 'Use when the task is specifically about gin-ninja-cli project/app scaffolding. Helps choose between startproject and startapp, pick scaffold templates, set the right CLI flags, and align generated output with repository examples.'
argument-hint: What do you want to scaffold with gin-ninja-cli?
---

# gin-ninja-scaffold

Use this skill when the user specifically wants to scaffold a new gin-ninja project or add a new app package with `gin-ninja-cli`.

## When to Use

- Create a brand new project with `gin-ninja-cli startproject`
- Add a new app inside an existing project with `gin-ninja-cli startapp`
- Choose between `minimal`, `standard`, `auth`, and `admin` scaffold templates
- Decide which scaffold flags to pass before running the CLI
- Map scaffold output back to the closest repository example

## Working Rules

1. Prefer the CLI over hand-writing project or app boilerplate.
2. Pick `startproject` for a new repository root and `startapp` for a domain package inside an existing project.
3. Choose the smallest template that satisfies the requested feature set.
4. Keep generated layout aligned with repository examples before layering custom code on top.
5. Use config presets only when repeated scaffold choices should be captured and replayed.

## Procedure

1. Identify the scaffold scope:
   - new repository/service root -> `startproject`
   - new domain/app package inside an existing project -> `startapp`
2. Read the command guide: [Startproject and startapp](./references/startproject-and-startapp.md)
3. Choose the template:
   - `minimal` -> smallest starter
   - `standard` -> fuller common project shape
   - `auth` -> auth-focused scaffold
   - `admin` -> admin-enabled scaffold
4. Fill the key flags:
   - `startproject`: `-module`, `-output`, `-config`, `-template`, optional overrides like `-app-dir`, `-with-tests`, `-with-auth`, `-with-admin`, `-with-gormx`, `-force`
   - `startapp`: `-output`, `-package`, `-model`, `-config`, `-template`, optional overrides like `-with-tests`, `-with-auth`, `-with-admin`, `-with-gormx`, `-force`
5. After scaffolding, compare the output to the closest example app before making custom edits.

## Repo Landmarks

- CLI implementation: `cmd/gin-ninja-cli/`
- Scaffold command help: `cmd/gin-ninja-cli/scaffold.go`, `cmd/gin-ninja-cli/help.go`
- Generated code templates: `cmd/gin-ninja-cli/internal/codegen/`
- Closest examples: `examples/basic`, `examples/users`, `examples/admin`, `examples/full`
- Reference: `references/startproject-and-startapp.md`
