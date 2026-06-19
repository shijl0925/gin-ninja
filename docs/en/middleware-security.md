# Middleware and Security

[Docs Home](../README.md) | [English Index](./README.md) | [中文](../zh/README.md)

## Middleware

### Engine-level (applies to all routes)

```go
api.UseGin(
    middleware.RequestID(),          // injects X-Request-ID
    middleware.Recovery(log),        // panic recovery with Zap logging
    middleware.Logger(log),          // structured request logging
    middleware.CORSFromConfig(cfg.CORS),
    orm.Middleware(db),              // per-request DB in context
)
```

For production, use `middleware.CORSFromConfig(cfg.CORS)` or an explicit `middleware.CORSConfig`, and keep `cfg.CORS.allow_origins` explicit. Do not use `middleware.CORS(nil)` in production: it allows all origins and panics in Gin release mode.

### Router-level (applies only to that group)

```go
protected := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
protected.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))  // JWT auth for /admin/* only
```

### JWT Authentication

```go
// Generate a token (e.g. after login):
token, err := middleware.GenerateTokenWithConfig(user.ID, user.Name, cfg.JWT)

// Protect routes:
r.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))

// Read claims in a handler:
claims := middleware.GetClaims(ctx.Context)
fmt.Println(claims.UserID, claims.Username)

```

### API Key, Basic, and OAuth2 Bearer Authentication

gin-ninja also includes middleware helpers for common non-JWT authentication schemes. Each helper validates the incoming credential with your callback and stores the returned principal on the Gin context.

```go
api := ninja.New(ninja.Config{
    SecuritySchemes: map[string]ninja.SecurityScheme{
        "apiKeyAuth": ninja.APIKeyHeaderSecurityScheme("X-API-Key"),
        "basicAuth":  ninja.HTTPBasicSecurityScheme(),
        "oauth2": ninja.OAuth2SecurityScheme(ninja.OAuthFlows{
            ClientCredentials: &ninja.OAuthFlow{
                TokenURL: "/oauth/token",
                Scopes: map[string]string{"read": "read data"},
            },
        }),
    },
})

r := ninja.NewRouter("/internal",
    ninja.WithAPIKeyAuth("apiKeyAuth", middleware.APIKeyHeader("X-API-Key", func(c *gin.Context, key string) (any, bool) {
        if key == "supersecret" {
            return key, true
        }
        return nil, false
    })),
)
```

The `WithBearerAuth`, `WithBasicAuth`, `WithAPIKeyAuth`, and
`WithOAuth2AuthMiddleware` router options bind runtime Gin middleware and
OpenAPI security metadata in one place. They require a middleware argument; use
`WithSecurity(...)` or `Security(...)` only when you intentionally want
documentation-only security metadata and enforce authentication elsewhere.

For OAuth2 endpoints that require scopes, use the scope-aware helper:

```go
r := ninja.NewRouter("/internal",
    ninja.WithOAuth2AuthMiddleware(
        middleware.OAuth2BearerAuthWithScopes([]string{"read"}, func(c *gin.Context, token string) (any, []string, bool) {
            if token == "supersecret" {
                return "user", []string{"read"}, true
            }
            return nil, nil, false
        }),
        "read",
    ),
)
```

JWT-authenticated claims are also available through `middleware.GetAuthPrincipal(...)`.

If you prefer the explicit two-step form, keep the documentation-only security option and middleware next to each other:

```go
r := ninja.NewRouter("/internal", ninja.WithSecurity("apiKeyAuth"))
r.UseGin(middleware.APIKeyHeader("X-API-Key", func(c *gin.Context, key string) (any, bool) {
    if key == "supersecret" {
        return key, true
    }
    return nil, false
}))
```

Available helpers:

- `middleware.APIKeyHeader(...)`, `middleware.APIKeyCookie(...)`, `middleware.APIKeyQuery(...)`
- `middleware.HTTPBasicAuth(...)`
- `middleware.HTTPBearerAuth(...)`
- `middleware.OAuth2BearerAuthWithScopes(...)`
- `middleware.GetAuthPrincipal(...)`
- OpenAPI helpers: `APIKeyHeaderSecurityScheme`, `APIKeyCookieSecurityScheme`, `APIKeyQuerySecurityScheme`, `HTTPBasicSecurityScheme`, `OAuth2SecurityScheme`
- Router/operation auth helpers: `WithAPIKeyAuth`, `WithBasicAuth`, `WithOAuth2AuthMiddleware`, `APIKeyAuth`, `BasicAuth`
- Documentation-only security helpers: `WithSecurity`, `Security`, `WithOAuth2Auth`, `OAuth2Auth`

### I18n – Locale Negotiation and Translated Messages

Register `middleware.I18n()` to automatically negotiate the client locale from the `Accept-Language`
request header.  Supported locales are `"en"` (English) and `"zh"` (Chinese), with `"en"` as the
fallback.

```go
api.UseGin(middleware.I18n())
```

Once registered, **validation-error messages are automatically translated** into the negotiated
locale without any additional code:

```
POST /users  Accept-Language: zh-CN

{
  "code": 422,
  "message": "请求参数校验失败",
  "data": {
    "errors": [
      { "field": "email", "message": "必须是有效的电子邮件地址" }
    ]
  }
}
```

Read the active locale inside a handler:

```go
func myHandler(ctx *ninja.Context, in *MyInput) (*MyOutput, error) {
    locale := ctx.Locale()           // "en" or "zh"
    msg    := ctx.T("not_found")     // "not found" or "资源不存在"
    _ = locale
    _ = msg
    return nil, nil
}
```

Or directly from a raw `*gin.Context` (e.g. inside a custom gin middleware):

```go
locale := middleware.GetLocale(c)
```

The `pkg/i18n` package exposes helpers for translating validation tags and general messages:

```go
import "github.com/shijl0925/gin-ninja/pkg/i18n"

locale := i18n.NegotiateLocale(r.Header.Get("Accept-Language"))
msg    := i18n.TranslateValidation("required", "", locale) // "field is required" / "字段不能为空"
msg2   := i18n.T(locale, "not_found")                     // "not found" / "资源不存在"
```

Available general message keys: `bad_request`, `unauthorized`, `forbidden`, `not_found`,
`conflict`, `internal`, `timeout`, `validation`, `rate_limited`.

### Session / Cookie Authentication

`middleware.SessionMiddleware` provides HMAC-SHA256-signed, cookie-based sessions without external
dependencies.  The session data (a `map[string]string`) is serialised as JSON, signed, and stored
in a single cookie.  Tampered cookies are automatically discarded.

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret: "change-me-in-production",
    MaxAge: 86400,          // 24 h
    // Secure defaults to true in gin.ReleaseMode and false in local development.
    // Set Secure: true to force HTTPS-only cookies outside release mode.
    // Set Secure: false, SecureSet: true to opt out explicitly.
    HTTPOnly: true,
}))

// In a handler:
session := middleware.GetSession(c)
session.Set("user_id", "42")          // mutations are saved automatically
v, ok := session.Get("user_id")
session.Delete("user_id")
```

### CSRF Protection

`middleware.CSRF` implements the **double-submit cookie** pattern.  A random token is set as a
cookie on the first safe request and must be echoed back in the `X-CSRF-Token` header (or
`csrf_token` form field) for all state-changing methods (POST, PUT, PATCH, DELETE).

```go
api.UseGin(middleware.CSRF(nil))   // defaults

// Custom config:
api.UseGin(middleware.CSRF(&middleware.CSRFConfig{
    // CookieSecure defaults to true in gin.ReleaseMode and false in local development.
    CookieSecure: true, // force HTTPS-only cookies outside release mode
    // Set CookieSecure: false, CookieSecureSet: true to opt out explicitly.
    CookieSameSite: http.SameSiteStrictMode,
}))

// Embed the token in forms / single-page apps:
token := middleware.CSRFToken(c)
```

Requests with missing or mismatched tokens are rejected with HTTP 403.

### Security Response Headers

`middleware.SecureHeaders` sets industry-standard security headers in a single call:

```go
// Sensible defaults:
api.UseGin(middleware.SecureHeaders(nil))

// Strict production config (HTTPS):
api.UseGin(middleware.SecureHeadersStrict())

// Custom config:
api.UseGin(middleware.SecureHeaders(&middleware.SecurityConfig{
    ContentTypeNoSniff:    true,
    FrameOption:           "SAMEORIGIN",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    HSTSMaxAge:            31536000,       // 1 year
    HSTSIncludeSubDomains: true,
    ContentSecurityPolicy: "default-src 'self'",
    PermissionsPolicy:     "geolocation=()",
}))
```

HSTS is only emitted when the request arrives over HTTPS (or the `X-Forwarded-Proto: https`
proxy header is present).

### Upload Size Limit and Content-Type Whitelist

`middleware.UploadLimit` rejects oversized bodies (HTTP 413) and requests with disallowed
content types (HTTP 415) for POST/PUT/PATCH endpoints:

```go
api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
    MaxSize:          5 << 20,   // 5 MiB
    AllowedMIMETypes: []string{
        "application/json",
        "image/",   // prefix: matches image/jpeg, image/png, etc.
    },
}))
```

Pass `nil` to use defaults (10 MiB limit, no content-type checking).

### Security Best Practices

For production deployments, combine the built-in middleware with a few operational safeguards:

- **Use strong secrets**: keep `jwt.secret` and `SessionConfig.Secret` long, random, and environment-specific; never commit placeholder secrets such as `change-me-in-production`.
- **Use environment-aware cookies**: `Secure`/`CookieSecure` default to HTTPS-only in Gin release mode and stay off for local HTTP development; set them explicitly when your development environment also uses HTTPS.
- **Force HTTPS end-to-end**: enable `Secure` cookies for sessions/CSRF, terminate TLS at the edge, and forward the original scheme so HSTS can be emitted correctly behind proxies.
- **Prefer strict browser protections**: start with `middleware.SecureHeadersStrict()` or explicitly set CSP, Referrer-Policy, `X-Frame-Options`, and HSTS for public deployments.
- **Keep cookies scoped tightly**: use `HTTPOnly`, an appropriate `SameSite` mode, and the narrowest practical `Domain`/`Path` to reduce cross-site exposure.
- **Protect all state-changing routes**: pair cookie-based auth with `middleware.CSRF(...)`, and make sure browser clients echo the CSRF token on every POST/PUT/PATCH/DELETE request.
- **Minimize upload attack surface**: set `UploadLimit` with both a size cap and an explicit MIME allowlist instead of accepting arbitrary request bodies.
- **Harden API docs exposure**: if `/docs` or `/openapi.json` should not be public in production, gate them behind auth, network policy, or disable those routes in your deployment wrapper.
- **Rotate and expire credentials**: keep JWT lifetimes short, rotate signing secrets during incident response, and issue new session IDs after login or privilege changes.

---

## API Version Deprecation Policy

`VersionConfig` now supports richer deprecation metadata:

```go
api := ninja.New(ninja.Config{
    Versions: map[string]ninja.VersionConfig{
        "v1": {
            Deprecated:      true,
            // Optional: emit an HTTP-date in the Deprecation header (RFC 8594):
            DeprecatedSince: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
            // Optional: emit a Sunset header:
            SunsetTime:      time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
            // Or use a pre-formatted string:
            // Optional: emit a Link header pointing to migration docs:
            MigrationURL: "https://example.com/migrate-to-v2",
        },
    },
})
```

Response headers on any deprecated version endpoint:

```
Deprecation: Mon, 01 Jan 2024 00:00:00 GMT
Sunset:      Tue, 01 Jul 2025 00:00:00 GMT
Link:        <https://example.com/migrate-to-v2>; rel="deprecation"
```

When `DeprecatedSince` is zero the `Deprecation` header falls back to the literal `"true"`.

---

Previous: [Configuration, Bootstrap, and Lifecycle](./configuration.md) | Next: [Data, Binding, and Responses](./data-and-responses.md)
