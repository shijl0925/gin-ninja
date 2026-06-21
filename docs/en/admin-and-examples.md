# Admin and Full Example

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/README.md)

## Admin Package

The `admin` sub-package provides a metadata-driven back-office API layer plus a built-in single-page admin UI shell that talks to that API.  All three pieces — Site, API routes, and UI pages — are wired up independently so you can use any subset.

### 1. Create a Site

```go
import admin "github.com/shijl0925/gin-ninja/admin"

site := admin.NewSite(
    // optional: enforce auth on every action
    admin.WithPermissionChecker(func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        if ctx.GetUserID() == 0 {
            return ninja.UnauthorizedError()
        }
        return nil
    }),
)
```

`NewSite` accepts zero or more `Option` values.  The only built-in option is `WithPermissionChecker`, which runs before every list / detail / create / update / delete action.

### 2. Register Models with `MustRegisterModel`

Each GORM model gets one `ModelResource` descriptor that controls which fields appear in which views and what operations are allowed.

```go
site.MustRegisterModel(&admin.ModelResource{
    // Model is the GORM model struct (value, not pointer).
    Model: User{},

    // Optional: the built-in UI uses these hints for grouping, ordering, and copy.
    Icon:        "users",
    Group:       "Identity",
    Description: "Manage back-office users, admin flags, and role relations.",
    Order:       10,

    // Preloads lists GORM association names to Preload on every query.
    Preloads: []string{"Roles"},

    // Field lists control which fields appear in each view.
    ListFields:   []string{"id", "name", "email", "is_admin", "createdAt"},
    DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt"},
    CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
    FilterFields: []string{"is_admin", "age", "createdAt"},
    SortFields:   []string{"id", "name", "email", "age", "createdAt"},
    SearchFields: []string{"name", "email"},

    // Optional per-field display/component overrides.
    FieldOptions: map[string]admin.FieldOptions{
        "is_admin": {Label: "Admin?", Component: "switch"},
        "age": {Format: "integer"},
        "createdAt": {Format: "relative"},
        "role_ids": {Help: "Search and select one or more roles.", Width: "full"},
    },

    // Optional permission hook called for every action on this resource.
    Permissions: func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        return nil
    },

    // Optional row-level query scope (e.g. multi-tenant filtering).
    RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, res *admin.Resource, db *gorm.DB) *gorm.DB {
        return db.Where("owner_id = ?", ctx.GetUserID())
    }),

    // Optional lifecycle hooks.
    BeforeCreate: func(ctx *ninja.Context, data map[string]any) error { return nil },
    AfterCreate:  func(ctx *ninja.Context, record any) error { return nil },
})
```

`MustRegisterModel` panics on configuration errors (e.g. duplicate resource name).  Use `RegisterModel` instead if you want to handle the error yourself.

Relation fields pointing to another registered model are resolved automatically: the framework infers `value_field`, `label_field`, and `search_fields` from the target resource.

Use `FieldOptions` when a field needs an explicit component, label, placeholder, help copy, width, or display format. The built-in UI currently understands display formats such as `title`, `uppercase`, `lowercase`, `mono`, `number`, `integer`, `percent`, `currency:USD`, `date`, `datetime`, and `relative`.

### 3. Mount the Admin API Routes

`site.Mount` registers REST endpoints for every resource under the given `*ninja.Router`.  The router is a standard gin-ninja router, so you can attach JWT middleware or any other gin middleware to it.

```go
adminRouter := ninja.NewRouter(
    "/admin",
    ninja.WithTags("Admin"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
    ninja.WithVersion("v1"),
)

site.Mount(adminRouter)
api.AddRouter(adminRouter)
```

This registers the following endpoints under `/api/v1/admin` (given `Prefix: "/api"` on `NinjaAPI`):

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/resources` | List all registered resources |
| `GET` | `/resources/stats` | Count summary for currently visible resources |
| `GET` | `/search?q=term` | Aggregated search across currently visible searchable resources |
| `GET` | `/resources/{path}/meta` | Resource field metadata |
| `GET` | `/resources/{path}` | Paginated record list (search / filter / sort) |
| `GET` | `/resources/{path}/export` | CSV export for the current search / filter / sort query; optional `fields=id,name` limits exported list fields |
| `GET` | `/resources/{path}/{id}` | Single record detail |
| `POST` | `/resources/{path}` | Create record |
| `PUT` | `/resources/{path}/{id}` | Update record |
| `DELETE` | `/resources/{path}/{id}` | Delete record |
| `POST` | `/resources/{path}/bulk-delete` | Bulk delete records |
| `GET` | `/resources/{path}/fields/{field}/options` | Relation selector options |

### 4. Mount the Built-in Admin UI Shell

`admin.MountUI` registers the standalone login page and admin workspace as plain HTML routes on any `gin.IRoutes` (including `api.Engine()` for top-level paths outside the API prefix).

```go
// Use all defaults: /admin/login, /admin
admin.MountUI(api.Engine(), admin.DefaultUIConfig())

// Or customise paths and title:
admin.MountUI(api.Engine(), admin.UIConfig{
    Title:         "My App Admin",
    BrandName:     "My App",
    LogoText:      "MA",
    Locale:        "en",
    DefaultTheme:  "system",
    TokenStorage:  "session",
    APIBasePath:   "/api/v1/admin",
    AuthLoginPath: "/api/v1/auth/login",
    AdminPath:     "/admin",
    LoginPath:     "/admin/login",
})
```

`UIConfig` fields and their defaults:

| Field | Default | Description |
|-------|---------|-------------|
| `Title` | `"Gin Ninja Admin"` | Browser tab title |
| `BrandName` | `"Gin Ninja"` | Brand name shown in the admin shell |
| `LogoText` | `"G"` | Text logo, capped to 3 characters |
| `Locale` | `"en"` | HTML language and localised formatting hint |
| `DefaultTheme` | `"light"` | Initial theme: `light`, `dark`, or `system` |
| `TokenStorage` | `"local"` | Token and session identity storage policy: `local` or `session` |
| `APIBasePath` | `"/api/v1/admin"` | Admin API root path (for resource navigation) |
| `AuthLoginPath` | `"/api/v1/auth/login"` | Login endpoint called by the sign-in form |
| `AdminPath` | `"/admin"` | Admin workspace page path |
| `LoginPath` | `"/admin/login"` | Standalone login page path |
| `TokenExtractExpr` | `"payload.token"` | JS expression to extract the token from the login response |
| `UserNameExtractExpr` | `"payload.name"` | JS expression to extract the display name |
| `UserIDExtractExpr` | `"payload.user_id \|\| payload.userID"` | JS expression to extract the user ID |

#### Customising the token extraction expression

By default the UI reads `payload.token` from the login response.  If your auth endpoint returns the token under a different key (e.g. `{"data": {"accessToken": "..."}}`) set `TokenExtractExpr`:

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:    "/api/v1/user/login",
    // For {"data": {"accessToken": "..."}}
    TokenExtractExpr: "payload.data && payload.data.accessToken",
})
```

The expression is passed as string data to a restricted path extractor.  It currently supports `payload.foo.bar` paths plus `&&` / `||` fallback combinations; it is not executed as arbitrary JavaScript.  Similarly, `UserNameExtractExpr` and `UserIDExtractExpr` customise where the display name and user ID are read from:

```go
admin.MountUI(router, admin.UIConfig{
    AuthLoginPath:       "/api/v1/user/login",
    TokenExtractExpr:    "payload.data && payload.data.accessToken",
    UserNameExtractExpr: "payload.data && payload.data.userName",
    UserIDExtractExpr:   "payload.data && payload.data.id",
})
```

When a login response includes a display name or user ID, the admin shell stores that session identity with the token so refreshed pages keep the same sidebar and topbar user label. Manual token changes clear the saved identity to avoid showing stale user information.

> **Security note:** these expressions should still come only from trusted, developer-controlled configuration — never from user-supplied input. Expressions outside the restricted path syntax fail to resolve.

---

## Full Example

Split examples are available by feature:

- [examples/users](../../examples/users/) — auth register/login plus JWT-protected users CRUD and the cached v2 users API
- [examples/features](../../examples/features/) — request metadata, cache / ETag, rate limit, timeout, versioned routing, SSE, WebSocket, upload, and download demos
- [examples/admin](../../examples/admin/) — JWT-protected admin resource APIs plus the standalone admin pages
- [examples/full](../../examples/full/) — the combined application with every feature above in one app
- [examples/compact](../../examples/compact/) — a compact counterpart to `examples/full` that shows the same feature set with fewer local files

The combined [examples/full](../../examples/full/) application includes:
- Settings from `config.yaml`
- Bootstrap (DB + logger initialisation)
- JWT-protected user CRUD endpoints
- Auth register/login endpoints
- Structured Zap logging
- Route-level cache / ETag / Cache-Control demos
- Versioned API routing and per-version docs demos
- SSE / WebSocket demos
- Multipart single-file and multi-file upload demos
- Binary download and reader-backed download demos

### Admin console in `examples/full`

The full example also includes a metadata-driven admin experience built on top of the JWT-protected admin resource APIs.

It includes:

- a standalone login page at `/admin/login`
- a standalone admin workspace at `/admin`
- resource navigation backed by `/api/v1/admin/resources`
- dashboard resource counts loaded in one request from `/api/v1/admin/resources/stats`
- topbar global aggregate search backed by `/api/v1/admin/search`
- sidebar navigation grouped and ordered by resource `Group` / `Order`
- resource-level action chips that show the currently allowed list / detail / write operations
- record listing with search, metadata-driven filters, sort, page size, pagination ranges, and refresh recency
- actionable empty states for clearing list state or creating the first record
- active list-state chips for the current search, sort, filters, and page size
- per-resource list state remembered locally when switching between resources
- per-resource saved views for reusing named search / filter / sort states
- modal focus management for faster keyboard-first create, edit, and detail flows
- copyable current-view links for sharing or bookmarking list state
- CSV export for the current search / filter / sort query and currently visible table columns
- delete and bulk-delete reloads step back from emptied pages automatically
- table density switching and visible-column controls with visible counts, show-all, and reset actions
- search, sort, pagination, and filters sync to the URL query so refreshes and shared links restore the list state
- collapsible filters with remembered collapsed state
- detail, create, update, delete, and bulk delete flows
- detail modal quick actions for editing the selected record, copying individual fields, and copying its JSON payload
- create / update forms with inline validation, pending submit states, and unsaved-change guards
- failed form submissions focus the first highlighted field
- bulk selection summary with one-click ID copying and selection clearing
- clickable, keyboard-openable table rows for faster record inspection
- relation-backed field selectors with option search previews
- a more compact “Admin Workspace” header for a denser back-office layout

Suggested manual flow:

1. Start the full example:
   ```bash
   cd examples/full
   go run .
   ```
2. Open `http://localhost:8080/admin/login`
3. Sign in with the demo credentials shown on the page
4. After redirecting to `/admin`, pick a resource from the left sidebar
5. Use the workspace to:
   - search and filter the current resource
   - change sort order and page size
   - page through result sets
   - inspect record details
   - create, edit, delete, or bulk delete records
   - preview relation options while filling relation-backed fields

Useful routes:

- `/admin/login` — standalone login shell
- `/admin` — standalone admin workspace
- `/api/v1/admin/resources` — admin metadata and CRUD API root

```bash
cd examples/full
go run .
# Open http://localhost:8080/docs
```

---

Previous: [Advanced Features](./advanced-features.md)
