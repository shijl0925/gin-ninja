# Optional Admin and Full Example

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/admin-and-examples.md)

`admin` is optional. A normal API project does not need it on day one. Read this page when you need an internal management UI, resource lists, detail views, create/update/delete actions, export, and a simple built-in UI shell.

## Three Parts

- `Site`: registers admin resources and global permissions.
- Admin API: mounts resource operations on a gin-ninja `Router`.
- Admin UI Shell: built-in HTML/JS pages that call the Admin API.

These parts can be used separately. You can mount only the API, or both the API and UI.

## Create A Site

```go
import (
    ninja "github.com/shijl0925/gin-ninja"
    admin "github.com/shijl0925/gin-ninja/admin"
)

site := admin.NewSite(
    admin.WithPermissionChecker(func(ctx *ninja.Context, action admin.Action, res *admin.Resource) error {
        if ctx.GetUserID() == 0 {
            return ninja.UnauthorizedError()
        }
        return nil
    }),
)
```

`WithPermissionChecker` runs before list, detail, create, update, delete, bulk delete, and other actions. If you do not need a global permission check, use `admin.NewSite()`.

Use `admin.WithExcludeFromDocs()` when the admin resource API is only used by the built-in UI and should not appear in generated OpenAPI/Swagger docs. This hides the docs entries only; the mounted routes continue to work.

## Register Resources

Each GORM model maps to one resource:

```go
site.MustRegisterModel(&admin.ModelResource{
    Model:       User{},
    Icon:        "users",
    Group:       "Identity",
    Description: "Manage users and roles.",
    Order:       10,

    ListFields:   []string{"id", "name", "email", "is_admin", "createdAt"},
    DetailFields: []string{"id", "name", "email", "age", "is_admin", "createdAt"},
    CreateFields: []string{"name", "email", "password", "age", "is_admin"},
    UpdateFields: []string{"name", "email", "password", "age", "is_admin"},
    FilterFields: []string{"is_admin", "age"},
    SortFields:   []string{"id", "name", "email", "createdAt"},
    SearchFields: []string{"name", "email"},

    FieldOptions: map[string]admin.FieldOptions{
        "password": {Placeholder: "Leave blank to keep current password."},
        "is_admin": {Label: "Admin", Component: "switch"},
    },
})
```

`MustRegisterModel` panics on configuration errors, which is useful during startup. Use `RegisterModel` when you want to handle the error yourself.

Common resource fields:

| Field | Purpose |
| --- | --- |
| `Model` | GORM model value, such as `User{}` |
| `ListFields` | fields shown on list pages |
| `DetailFields` | fields shown on detail pages |
| `CreateFields` / `UpdateFields` | create/edit form fields |
| `FilterFields` / `SortFields` / `SearchFields` | list query capabilities |
| `FieldOptions` | labels, components, placeholders, help text, formats |
| `Permissions` | per-resource permission check |
| `RowPermissions` | row-level query scope, such as tenant filtering |
| `BeforeCreate` / `AfterCreate` and similar hooks | lifecycle hooks |

Row-level permission example:

```go
RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, res *admin.Resource, db *gorm.DB) *gorm.DB {
    return db.Where("owner_id = ?", ctx.GetUserID())
}),
```

## Mount The Admin API

Admin API uses a normal gin-ninja router, so you can attach JWT, versions, tags, and middleware:

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

If `api.Prefix` is `/api`, this creates `/api/v1/admin/...` endpoints.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/resources` | list resources |
| `GET` | `/resources/stats` | resource record counts |
| `GET` | `/search?q=keyword` | search across resources |
| `GET` | `/resources/{path}/meta` | field metadata |
| `GET` | `/resources/{path}` | paginated list |
| `GET` | `/resources/{path}/export` | CSV export |
| `GET` | `/resources/{path}/{id}` | detail |
| `POST` | `/resources/{path}` | create |
| `PUT` | `/resources/{path}/{id}` | update |
| `DELETE` | `/resources/{path}/{id}` | delete |
| `POST` | `/resources/{path}/bulk-delete` | bulk delete |
| `GET` | `/resources/{path}/fields/{field}/options` | relation field options |

## Mount The Built-In UI

Default UI:

```go
admin.MountUI(api.Engine(), admin.DefaultUIConfig())
```

Custom UI:

```go
admin.MountUI(api.Engine(), admin.UIConfig{
    Title:         "My App Admin",
    BrandName:     "My App",
    LogoText:      "MA",
    Locale:        "en",
    DefaultTheme:  "system",
    TokenStorage:  "session",
    APIBasePath:   "/api/v1/admin",
    AuthLoginPath: "/api/v1/auth/login",
    AuthMePath:    "/api/v1/auth/me",
    AdminPath:     "/admin",
    LoginPath:     "/admin/login",
})
```

If your login response stores the token somewhere other than `payload.token`, configure extraction expressions:

```go
admin.MountUI(api.Engine(), admin.UIConfig{
    AuthLoginPath:       "/api/v1/auth/login",
    TokenExtractExpr:    "payload.data && payload.data.accessToken",
    UserNameExtractExpr: "payload.data && payload.data.userName",
    UserIDExtractExpr:   "payload.data && payload.data.id",
})
```

These expressions support restricted `payload.foo.bar` paths and `&&` / `||` fallback combinations. They are not executed as arbitrary JavaScript. Configuration should still only come from trusted developers.

### Current User Endpoint (`AuthMePath`)

`AuthMePath` configures the endpoint that verifies the saved token and loads the signed-in user's profile. It defaults to `/api/v1/auth/me`. The built-in UI sends `GET` requests to this path with `Authorization: Bearer <token>` after a token is restored or a user signs in.

Return a successful JSON response (`2xx`) containing the profile directly, or nest it in a `user` object. All profile fields are optional, but return at least one of `name`, `user_id`, or `email` so the UI can persist and display the identity:

```json
{
  "user_id": 42,
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "is_admin": true,
  "roles": [
    { "id": 1, "name": "Administrator", "code": "admin", "status": "active" }
  ],
  "expires_at": "2026-12-31T23:59:59Z",
  "issued_at": "2026-01-01T00:00:00Z",
  "issuer": "gin-ninja"
}
```

| UI value | Accepted response fields |
| --- | --- |
| Display name | `name`, `username`, or `user_name` |
| User ID | `user_id`, `userID`, or `id` |
| Email | `email` |
| Admin indicator | `is_admin`, `isAdmin`, or `admin` |
| Roles | `roles` or `permissions`; each item may be a string or an object with `id`/`ID`, `name`/`Name`, `code`/`Code`, and `status`/`Status` |
| Token metadata | `expires_at`/`expiresAt`, `issued_at`/`issuedAt`, and `issuer` |

For example, `{ "user": { "id": 42, "username": "Ada" } }` is also valid. A `401` response clears the session and redirects to login; a `403` response shows an access-denied page. The Admin API must still enforce authorization server-side—`is_admin` and `roles` are only UI profile data.

## Runnable Examples

Examples are ordered by complexity:

| Example | Purpose |
| --- | --- |
| `examples/basic` | users CRUD API; good first code reading target |
| `examples/users` | login, registration, JWT-protected users API |
| `examples/features` | cache, ETag, rate limit, timeout, versioning, upload, download |
| `examples/admin` | focused admin API and UI |
| `examples/full` | full application with config, DB, auth, middleware, admin, versions, files |
| `examples/compact` | compact variant of the full example |
| `examples/controller` | controller organization example |

Run examples:

```bash
go run ./examples/basic
go run ./examples/features
go run ./examples/admin
go run ./examples/full
```

Common URLs after startup:

- `/docs`: Swagger UI
- `/openapi.json`: OpenAPI JSON
- `/admin/login` and `/admin`: examples with Admin UI

## When Not To Use Admin

- A public product admin UI that needs complex interactions and custom design.
- A complex permission model with many layered field, row, and action rules.
- Resources that are not GORM models or cannot map naturally to list/detail/form operations.

In those cases, reuse gin-ninja for the API and build the frontend separately.

Previous: [Advanced Features](./advanced-features.md)
