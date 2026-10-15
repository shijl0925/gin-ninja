# Middleware and Security

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/middleware-security.md)

Middleware handles concerns that are not business logic but still apply to requests: logging, CORS, authentication, sessions, CSRF, security headers, upload limits, and more. First decide the scope, then choose the middleware.

## Scope

Global middleware applies to routers mounted after it:

```go
api := ninja.New(ninja.Config{Title: "My API"})

api.UseGin(
    middleware.RequestID(),
    middleware.CORSFromConfig(cfg.CORS),
    gin.Recovery(),
)
```

Router-level middleware applies only to that group:

```go
adminRouter := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
adminRouter.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))
api.AddRouter(adminRouter)
```

Operation-level auth can be attached with options such as `ninja.BearerAuth(...)` and `ninja.APIKeyAuth(...)`.

## Recommended Baseline

Most APIs can start with:

```go
api.UseGin(
    middleware.RequestID(),
    middleware.CORSFromConfig(cfg.CORS),
    middleware.SecureHeaders(nil),
    gin.Recovery(),
)
```

With Zap logging:

```go
logger := bootstrap.InitLogger(&cfg.Log)
api.UseGin(
    middleware.Logger(logger),
    middleware.Recovery(logger),
)
```

Do not use `middleware.CORS(nil)` in production. It is a permissive development mode that allows all origins, and it panics in Gin release mode. Use `middleware.CORSFromConfig(cfg.CORS)` or an explicit `middleware.CORSConfig`.

## JWT Authentication

Generate a token:

```go
token, err := middleware.GenerateTokenWithConfig(1, "alice", cfg.JWT)
```

Protect a router:

```go
users := ninja.NewRouter("/users",
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

Read the user ID in a handler:

```go
func me(ctx *ninja.Context, _ *struct{}) (*MeOut, error) {
    userID := ctx.GetUserID()
    return &MeOut{ID: userID}, nil
}
```

If a gateway or other middleware handles auth and you only need OpenAPI metadata, use documentation-only security:

```go
users := ninja.NewRouter("/users", ninja.WithSecurity("bearerAuth"))
```

Declare the Swagger Authorize scheme:

```go
api := ninja.New(ninja.Config{
    SecuritySchemes: map[string]ninja.SecurityScheme{
        "bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
    },
})
```

## API Key, Basic, And OAuth2 Bearer

API key from a header:

```go
auth := middleware.APIKeyHeader("X-API-Key", func(c *gin.Context, key string) (any, bool) {
    return "service-a", key == cfg.ServiceAPIKey
})

router := ninja.NewRouter("/internal",
    ninja.WithAPIKeyAuth("ApiKeyAuth", auth),
)
```

OpenAPI scheme:

```go
SecuritySchemes: map[string]ninja.SecurityScheme{
    "ApiKeyAuth": ninja.APIKeyHeaderSecurityScheme("X-API-Key"),
}
```

Basic auth:

```go
basic := middleware.HTTPBasicAuth("Admin", func(c *gin.Context, username, password string) (any, bool) {
    return username, username == "admin" && password == cfg.AdminPassword
})

router.UseGin(basic)
```

OAuth2 scope check:

```go
oauth := middleware.OAuth2BearerAuthWithScopes(
    []string{"users:read"},
    validateOAuthToken,
)

ninja.Get(router, "/", listUsers,
    ninja.OAuth2AuthMiddleware(oauth, "users:read"),
)
```

After an auth middleware succeeds, it stores the authenticated principal in the Gin context. Raw Gin middleware or handlers can read it with `middleware.GetAuthPrincipal(c)`. Scope-aware OAuth2 middleware also exposes granted scopes through `middleware.GetAuthScopes(c)`:

```go
router.UseGin(func(c *gin.Context) {
    principal := middleware.GetAuthPrincipal(c)
    scopes := middleware.GetAuthScopes(c)

    _ = principal
    _ = scopes

    c.Next()
})
```

In gin-ninja handlers, JWT user IDs are usually read with `ctx.GetUserID()`. If you need the full principal stored by API key, Basic, Bearer, or OAuth2 middleware, use the underlying Gin context: `middleware.GetAuthPrincipal(ctx.Context)`.

## Session And CSRF

Cookie sessions fit browser admin tools or pages that need signed server-side state:

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret:   cfg.JWT.Secret,
    MaxAge:   86400,
    SameSite: http.SameSiteLaxMode,
}))
```

Use the session in plain Gin middleware or handlers:

```go
session := middleware.GetSession(c)
session.Set("user_id", "42")
```

CSRF is for browser flows with cookie authentication. Pure Bearer Token APIs usually do not need CSRF:

```go
api.UseGin(middleware.CSRF(&middleware.CSRFConfig{
    Secret: cfg.JWT.Secret,
}))
```

Browser forms or frontend requests should submit the CSRF token issued by the server. The exact read/submit convention is up to your frontend.

## Security Headers

Default security headers:

```go
api.UseGin(middleware.SecureHeaders(nil))
```

For production HTTPS behind a trusted reverse proxy:

```go
api.UseGin(middleware.SecureHeadersStrict())
```

Custom:

```go
api.UseGin(middleware.SecureHeaders(&middleware.SecurityConfig{
    ContentTypeNoSniff:    true,
    FrameOption:           "DENY",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    HSTSMaxAge:            31536000,
    HSTSIncludeSubDomains: true,
    TrustForwardedProto:   true,
    ContentSecurityPolicy: "default-src 'self'",
}))
```

Only enable `TrustForwardedProto` when you trust the reverse proxy that sets `X-Forwarded-Proto`.

## Upload Limits

Upload endpoints should set both size limits and MIME allowlists:

```go
api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
    MaxSize:          10 << 20,
    AllowedMIMETypes: []string{"image/png", "image/jpeg", "application/pdf"},
}))
```

Oversized requests return `413`; disallowed content types return `415`. `AllowedMIMETypes` accepts exact MIME types and prefix matches ending in `/`, so `"image/"` matches `"image/png"` and `"image/jpeg"`. `GET` requests are not affected by upload limits.

## i18n

Enable locale negotiation:

```go
api.UseGin(middleware.I18n())
```

gin-ninja uses `Accept-Language` to set the locale and translate validation messages. In handlers:

```go
locale := ctx.Locale()
message := ctx.T("validation.required")
```

## Security Checklist

- Configure explicit CORS origins in production.
- Put JWT secrets, session secrets, and database passwords in environment variables.
- Use CSRF for browser cookie authentication; pure Bearer Token APIs usually do not need it.
- Set upload size and content-type limits.
- Return DTOs from public APIs instead of models with sensitive fields.
- Protect internal and admin endpoints with auth and audit logs.

Previous: [Configuration, Bootstrap, and Lifecycle](./configuration.md) | Next: [File Transfer: Uploads and Downloads](./file-transfer.md)
