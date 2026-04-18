---
name: gin-ninja
description: 'Use when building, extending, refactoring, or debugging Go HTTP APIs with github.com/shijl0925/gin-ninja. Helps choose NinjaAPI/Router patterns, typed request/response structs, binding tags, middleware, pagination, caching, versioning, streaming, admin features, settings/bootstrap wiring, and gin-ninja-cli scaffold commands.'
argument-hint: What do you want to build or change with gin-ninja?
---

# gin-ninja

Use this skill when the task belongs to a service built on `github.com/shijl0925/gin-ninja`, or when you want to create one in the repository's idiomatic style.

## When to Use

- Create a new gin-ninja service or app package
- Add or refactor typed API routes
- Convert raw Gin handlers into `NinjaAPI` + `Router` + typed operation helpers
- Choose the right request binding tags (`path`, `form`, `header`, `cookie`, `json`, `file`)
- Add middleware, auth, transactions, pagination, filtering, ordering, caching, versioning, SSE, or WebSocket endpoints
- Keep implementation and generated OpenAPI docs aligned
- Use `gin-ninja-cli` scaffolding, CRUD generation, or migration commands

## Working Rules

1. Prefer framework primitives before ad-hoc raw Gin wiring for documented API endpoints.
2. Model request input and response output with dedicated Go structs instead of manual parsing.
3. Put validation and binding behavior in struct tags and route options so docs stay in sync.
4. Reuse built-in middleware and helper packages before adding custom infrastructure.
5. Use the repository examples and scaffold commands to match the existing project style.

## Procedure

1. Identify the job:
   - new project or app scaffold -> [Scaffolding and examples](./references/scaffolding-and-examples.md)
   - new or changed endpoint -> [API patterns](./references/api-patterns.md)
2. Pick the core shape:
   - API root -> `ninja.New(ninja.Config{...})`
   - route group -> `ninja.NewRouter(...)`
   - endpoint -> `ninja.Get/Post/Put/Patch/Delete/SSE/WebSocket(...)`
3. Define typed input and output structs, then choose the correct binding tags and validation rules.
4. Apply route/router options for summaries, tags, auth, transactions, pagination, caching, versioning, and extra documented responses.
5. Reuse existing middleware, settings, bootstrap, ORM, and response helpers where they fit.
6. Validate by updating or adding focused tests and keeping docs endpoints (`/docs`, `/openapi.json`) correct.

## Repo Landmarks

- Core framework: `ninja.go`, `router.go`, `operation.go`, `binding.go`, `openapi.go`
- Advanced features: `cache.go`, `versioning.go`, `stream.go`, `transfer.go`
- Middleware: `middleware/`
- ORM/settings/bootstrap helpers: `orm/`, `settings/`, `bootstrap/`
- Runnable examples: `examples/basic`, `examples/users`, `examples/features`, `examples/admin`, `examples/full`
- CLI scaffolding and migrations: `cmd/gin-ninja-cli/`
