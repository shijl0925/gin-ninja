# 中间件与安全

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 中间件

### 引擎级中间件（作用于所有路由）

```go
api.UseGin(
    middleware.RequestID(),          // 注入 X-Request-ID
    middleware.Recovery(log),        // panic 恢复并使用 Zap 记录日志
    middleware.Logger(log),          // 结构化请求日志
    middleware.CORSFromConfig(cfg.CORS),
    orm.Middleware(db),              // 将 DB 注入每个请求的 context
)
```

生产环境请使用 `middleware.CORSFromConfig(cfg.CORS)` 或显式的 `middleware.CORSConfig`，并保持 `cfg.CORS.allow_origins` 显式配置。不要在生产环境使用 `middleware.CORS(nil)`：它会允许所有来源，并且在 Gin release mode 下会 panic。

### 路由级中间件（只作用于该分组）

```go
protected := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
protected.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))  // 仅保护 /admin/*
```

### JWT 认证

```go
// 生成 token（例如登录后）：
token, err := middleware.GenerateTokenWithConfig(user.ID, user.Name, cfg.JWT)

// 保护路由：
r.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))

// 在处理器中读取 claims：
claims := middleware.GetClaims(ctx.Context)
fmt.Println(claims.UserID, claims.Username)

```

### API Key、HTTP Basic 与 OAuth2 Bearer 认证

gin-ninja 还包含常见非 JWT 认证方案的中间件辅助函数。每个辅助函数都会通过你提供的回调校验传入凭据，并把回调返回的 principal 存入 Gin context。

```go
api := ninja.New(ninja.Config{
    SecuritySchemes: map[string]ninja.SecurityScheme{
        "apiKeyAuth": ninja.APIKeyHeaderSecurityScheme("X-API-Key"),
        "basicAuth":  ninja.HTTPBasicSecurityScheme(),
        "oauth2": ninja.OAuth2SecurityScheme(ninja.OAuthFlows{
            ClientCredentials: &ninja.OAuthFlow{
                TokenURL: "/oauth/token",
                Scopes: map[string]string{"read": "读取数据"},
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

`WithBearerAuth`、`WithBasicAuth`、`WithAPIKeyAuth` 和
`WithOAuth2AuthMiddleware` 路由选项会把运行时 Gin 中间件和 OpenAPI
security 元数据绑定在一起。它们要求传入中间件；只有在确实想要仅文档用
security 元数据，并在其他地方执行认证时，才使用 `WithSecurity(...)` 或 `Security(...)`。

对于需要 OAuth2 scope 的端点，请使用 scope 感知的辅助函数：

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

JWT 认证后的 claims 也可以通过 `middleware.GetAuthPrincipal(...)` 获取。

如果更偏好显式的两步写法，请把仅文档用 security 选项和中间件相邻放置：

```go
r := ninja.NewRouter("/internal", ninja.WithSecurity("apiKeyAuth"))
r.UseGin(middleware.APIKeyHeader("X-API-Key", func(c *gin.Context, key string) (any, bool) {
    if key == "supersecret" {
        return key, true
    }
    return nil, false
}))
```

可用辅助函数：

- `middleware.APIKeyHeader(...)`、`middleware.APIKeyCookie(...)`、`middleware.APIKeyQuery(...)`
- `middleware.HTTPBasicAuth(...)`
- `middleware.HTTPBearerAuth(...)`
- `middleware.OAuth2BearerAuthWithScopes(...)`
- `middleware.GetAuthPrincipal(...)`
- OpenAPI 辅助函数：`APIKeyHeaderSecurityScheme`、`APIKeyCookieSecurityScheme`、`APIKeyQuerySecurityScheme`、`HTTPBasicSecurityScheme`、`OAuth2SecurityScheme`
- 路由/操作认证辅助函数：`WithAPIKeyAuth`、`WithBasicAuth`、`WithOAuth2AuthMiddleware`、`APIKeyAuth`、`BasicAuth`
- 仅文档用 security 辅助函数：`WithSecurity`、`Security`、`WithOAuth2Auth`、`OAuth2Auth`

### i18n – 语言协商与翻译消息

注册 `middleware.I18n()` 后，会从 `Accept-Language` 请求头自动协商客户端语言。支持的 locale 为 `"en"`（英文）和 `"zh"`（中文），默认回退到 `"en"`。

```go
api.UseGin(middleware.I18n())
```

注册后，**校验错误消息会自动翻译**为协商后的语言，不需要额外代码：

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

在处理器中读取当前 locale：

```go
func myHandler(ctx *ninja.Context, in *MyInput) (*MyOutput, error) {
    locale := ctx.Locale()           // "en" 或 "zh"
    msg    := ctx.T("not_found")     // "not found" 或 "资源不存在"
    _ = locale
    _ = msg
    return nil, nil
}
```

也可以直接从原始 `*gin.Context` 中读取（例如在自定义 Gin 中间件中）：

```go
locale := middleware.GetLocale(c)
```

`pkg/i18n` 包暴露了翻译校验标签和通用消息的辅助函数：

```go
import "github.com/shijl0925/gin-ninja/pkg/i18n"

locale := i18n.NegotiateLocale(r.Header.Get("Accept-Language"))
msg    := i18n.TranslateValidation("required", "", locale) // "field is required" / "字段不能为空"
msg2   := i18n.T(locale, "not_found")                       // "not found" / "资源不存在"
```

可用通用消息 key：`bad_request`、`unauthorized`、`forbidden`、`not_found`、`conflict`、`internal`、`timeout`、`validation`、`rate_limited`。

### Session / Cookie 认证

`middleware.SessionMiddleware` 提供基于 Cookie 的 HMAC-SHA256 签名 session，不依赖外部存储。Session 数据（`map[string]string`）会序列化为 JSON、签名并存储在单个 Cookie 中。被篡改的 Cookie 会自动丢弃。

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret: "change-me-in-production",
    MaxAge: 86400,          // 24 h
    // Secure 在 gin.ReleaseMode 下默认为 true，本地开发默认为 false。
    // 设置 Secure: true 可在非 release mode 下强制 HTTPS-only Cookie。
    // 设置 Secure: false, SecureSet: true 可显式关闭。
    HTTPOnly: true,
}))

// 在处理器中：
session := middleware.GetSession(c)
session.Set("user_id", "42")          // 变更会自动保存
v, ok := session.Get("user_id")
session.Delete("user_id")

// 生成新的 session ID（用于服务端 session 存储）：
id := middleware.NewSessionID()
```

### CSRF 防护

`middleware.CSRF` 实现了 **double-submit cookie** 模式。第一次安全请求会设置随机 token Cookie；所有修改状态的方法（POST、PUT、PATCH、DELETE）都必须在 `X-CSRF-Token` 请求头（或 `csrf_token` 表单字段）中回传该 token。

```go
api.UseGin(middleware.CSRF(nil))   // 默认配置

// 自定义配置：
api.UseGin(middleware.CSRF(&middleware.CSRFConfig{
    // CookieSecure 在 gin.ReleaseMode 下默认为 true，本地开发默认为 false。
    CookieSecure: true, // 在非 release mode 下强制 HTTPS-only Cookie
    // 设置 CookieSecure: false, CookieSecureSet: true 可显式关闭。
    CookieSameSite: http.SameSiteStrictMode,
}))

// 在表单 / 单页应用中嵌入 token：
token := middleware.CSRFToken(c)
```

缺失或不匹配 token 的请求会返回 HTTP 403。

### 安全响应头

`middleware.SecureHeaders` 一次性设置业界标准安全响应头：

```go
// 合理默认值：
api.UseGin(middleware.SecureHeaders(nil))

// 严格生产配置（HTTPS）：
api.UseGin(middleware.SecureHeadersStrict())

// 自定义配置：
api.UseGin(middleware.SecureHeaders(&middleware.SecurityConfig{
    ContentTypeNoSniff:    true,
    FrameOption:           "SAMEORIGIN",
    XSSProtection:         true,
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    HSTSMaxAge:            31536000,       // 1 year
    HSTSIncludeSubDomains: true,
    ContentSecurityPolicy: "default-src 'self'",
    PermissionsPolicy:     "geolocation=()",
}))
```

只有请求通过 HTTPS 到达（或存在 `X-Forwarded-Proto: https` 代理头）时才会输出 HSTS。

### 上传大小限制与 Content-Type 白名单

`middleware.UploadLimit` 会拒绝 POST/PUT/PATCH 端点中过大的请求体（HTTP 413）和不允许的内容类型（HTTP 415）：

```go
api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
    MaxSize:          5 << 20,   // 5 MiB
    AllowedMIMETypes: []string{
        "application/json",
        "image/",   // 前缀：匹配 image/jpeg、image/png 等
    },
}))
```

传入 `nil` 可使用默认值（10 MiB 限制，不检查 content-type）。

### 安全最佳实践

生产部署时，请将内置中间件与一些运维防护结合使用：

- **使用强密钥**：保持 `jwt.secret` 和 `SessionConfig.Secret` 足够长、随机且按环境区分；不要提交 `change-me-in-production` 等占位密钥。
- **使用环境感知 Cookie**：`Secure` / `CookieSecure` 在 Gin release mode 下默认 HTTPS-only，本地 HTTP 开发默认关闭；如果开发环境也使用 HTTPS，请显式设置。
- **全链路强制 HTTPS**：为 session/CSRF 启用 Secure Cookie，在边缘终止 TLS，并转发原始 scheme，以便代理后正确输出 HSTS。
- **优先使用严格浏览器防护**：公开部署从 `middleware.SecureHeadersStrict()` 开始，或显式设置 CSP、Referrer-Policy、`X-Frame-Options` 和 HSTS。
- **缩小 Cookie 作用域**：使用 `HTTPOnly`、合适的 `SameSite` 模式，以及尽可能窄的 `Domain` / `Path`，减少跨站暴露。
- **保护所有修改状态的路由**：Cookie 认证应配合 `middleware.CSRF(...)`，并确保浏览器客户端在每个 POST/PUT/PATCH/DELETE 请求中回传 CSRF token。
- **最小化上传攻击面**：为 `UploadLimit` 同时设置大小上限和明确 MIME 白名单，而不是接受任意请求体。
- **加固 API 文档暴露**：如果生产环境不应公开 `/docs` 或 `/openapi.json`，请通过认证、网络策略保护，或在部署封装中禁用这些路由。
- **轮换并过期凭证**：保持 JWT 生命周期较短，事故响应时轮换签名密钥，并在登录或权限变更后签发新的 session ID。

---

## API 版本弃用策略

`VersionConfig` 现在支持更丰富的弃用元数据：

```go
api := ninja.New(ninja.Config{
    Versions: map[string]ninja.VersionConfig{
        "v1": {
            Deprecated:      true,
            // 可选：在 Deprecation 头中输出 HTTP-date（RFC 8594）：
            DeprecatedSince: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
            // 可选：输出 Sunset 头：
            SunsetTime:      time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
            // 或使用预格式化字符串：
            // Sunset: "Tue, 01 Jul 2025 00:00:00 GMT",
            // 可选：输出指向迁移文档的 Link 头：
            MigrationURL: "https://example.com/migrate-to-v2",
        },
    },
})
```

任何弃用版本端点的响应头：

```
Deprecation: Mon, 01 Jan 2024 00:00:00 GMT
Sunset:      Tue, 01 Jul 2025 00:00:00 GMT
Link:        <https://example.com/migrate-to-v2>; rel="deprecation"
```

当 `DeprecatedSince` 为零值时，`Deprecation` 头会回退为字面量 `"true"`。

---

上一篇: [配置、Bootstrap 与生命周期](./configuration.md) | 下一篇: [数据、绑定与响应](./data-and-responses.md)
