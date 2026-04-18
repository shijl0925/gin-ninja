# Advanced features

## File upload and download

- Bind uploaded multipart files with `file:"..."` onto:
  - `*ninja.UploadedFile`
  - `[]*ninja.UploadedFile` for repeated fields
- `UploadedFile` wraps `multipart.FileHeader` and exposes:
  - filename and size metadata
  - `Open()`
  - `Bytes()`
- Use `ninja.NewDownload(filename, contentType, data)` for byte-slice downloads.
- Use `ninja.NewDownloadReader(filename, contentType, size, reader)` for streamed or reader-backed downloads.
- `*ninja.Download` also supports:
  - `Inline = true` to switch away from attachment downloads
  - `Headers` for extra response headers

## Model schema shaping

- Use `ninja.ModelSchema[T]` when API output should expose only part of a model.
- Common patterns:
  - embed `ninja.ModelSchema[Model]` inside a response struct with `fields:"..."` and `exclude:"..."`
  - call `ninja.BindModelSchema[Out](model)` when a typed response schema already embeds `ModelSchema`
  - call `ninja.NewModelSchema(model, ninja.Fields(...), ninja.Exclude(...))` for ad-hoc field filtering without defining a new output type
- Prefer this when persistence models contain sensitive fields or internal-only relationships.

## Lifecycle hooks and serving

- `api.OnStartup(hook)` runs before the server begins accepting traffic.
- `api.OnShutdown(hook)` runs during graceful shutdown, in reverse registration order.
- Use startup hooks for warm-up and dependency checks.
- Use shutdown hooks for closing DB handles, caches, and background resources.
- `api.Run(addr)` is the default path when the app owns its own listener and OS signal handling.
- `api.Serve(listener)` is the better fit when embedding gin-ninja into a custom server/bootstrap flow that manages listeners or shutdown externally.

## OpenAPI security schemes

- Configure reusable docs/auth schemes on `ninja.Config.SecuritySchemes`.
- `ninja.HTTPBearerSecurityScheme("JWT")` is the common default for bearer auth.
- Pair config-level schemes with operation/router security options such as `BearerAuth`, `Security`, `WithBearerAuth`, or `WithSecurity`.
- This is what powers Swagger UI's `Authorize` button and keeps security requirements documented in OpenAPI.

## API version deprecation metadata

- `VersionConfig` supports richer deprecation signaling beyond just a prefix:
  - `Deprecated`
  - `DeprecatedSince`
  - `Sunset`
  - `SunsetTime`
  - `MigrationURL`
- Use these when keeping older versions alive while steering clients to a replacement.
- The framework can emit:
  - `Deprecation`
  - `Sunset`
  - `Link: <...>; rel="deprecation"`
- Use router-level `WithVersion(...)` to attach operations to the intended version group.

## Admin package overview

- The `admin/` package exposes a metadata-driven admin API over GORM models.
- `admin.Resource` is the core declaration point for one managed model/resource.
- Common resource configuration includes:
  - list/detail/create/update field sets
  - filter, sort, and search fields
  - `FieldOptions` for labels, enums, relations, visibility, and per-view behavior
- Permission and query controls:
  - `PermissionChecker`
  - `QueryScope`
  - `RowPermissionChecker`
  - `FieldPermissionChecker`
- Lifecycle hooks around mutations:
  - `BeforeCreateHook` / `AfterCreateHook`
  - `BeforeUpdateHook` / `AfterUpdateHook`
  - `BeforeDeleteHook` / `AfterDeleteHook`
- Reach for `examples/admin` or `examples/full` when the task touches admin resources.

## Good defaults

1. Use typed file fields and `*ninja.Download` instead of manual multipart or binary response plumbing.
2. Use `ModelSchema` whenever API shape should diverge from the ORM model.
3. Put resource cleanup in `OnShutdown`, not in ad-hoc signal handlers.
4. Define `SecuritySchemes` once at API construction and reuse named requirements on routers and operations.
5. Treat version deprecation metadata as part of the public contract, not just internal notes.
