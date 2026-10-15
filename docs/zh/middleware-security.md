# 中间件与安全

[文档首页](../README.md) | [中文索引](./README.md) | [English](../en/middleware-security.md)

中间件用于处理和业务无关但每个请求都需要的事情：日志、CORS、认证、Session、CSRF、安全头、上传限制等。先决定作用范围，再选择具体中间件。

## 作用范围

全局中间件作用于之后挂载的所有路由：

```go
api := ninja.New(ninja.Config{Title: "My API"})

api.UseGin(
    middleware.RequestID(),
    middleware.CORSFromConfig(cfg.CORS),
    gin.Recovery(),
)
```

路由级中间件只作用于当前分组：

```go
adminRouter := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
adminRouter.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))
api.AddRouter(adminRouter)
```

操作级认证可以通过 `ninja.BearerAuth(...)`、`ninja.APIKeyAuth(...)` 等选项挂到单个接口。

## 推荐基础组合

多数 API 可以从这组开始：

```go
api.UseGin(
    middleware.RequestID(),
    middleware.CORSFromConfig(cfg.CORS),
    middleware.SecureHeaders(nil),
    gin.Recovery(),
)
```

如果使用 Zap 日志：

```go
logger := bootstrap.InitLogger(&cfg.Log)
api.UseGin(
    middleware.Logger(logger),
    middleware.Recovery(logger),
)
```

生产环境不要使用 `middleware.CORS(nil)`。它是开发期的宽松模式，允许所有来源；在 Gin release mode 下会直接 panic。生产环境使用 `middleware.CORSFromConfig(cfg.CORS)` 或显式 `middleware.CORSConfig`。

## JWT 认证

生成 token：

```go
token, err := middleware.GenerateTokenWithConfig(1, "alice", cfg.JWT)
```

保护一组路由：

```go
users := ninja.NewRouter("/users",
    ninja.WithTags("Users"),
    ninja.WithBearerAuth(middleware.JWTAuthWithConfig(cfg.JWT)),
)
```

在 handler 中读取用户 ID：

```go
func me(ctx *ninja.Context, _ *struct{}) (*MeOut, error) {
    userID := ctx.GetUserID()
    return &MeOut{ID: userID}, nil
}
```

如果只想在 OpenAPI 中标记安全要求，而认证由网关或其他中间件处理，可以用文档型选项：

```go
users := ninja.NewRouter("/users", ninja.WithSecurity("bearerAuth"))
```

同时在 API 配置中声明 Swagger 的 Authorize 按钮：

```go
api := ninja.New(ninja.Config{
    SecuritySchemes: map[string]ninja.SecurityScheme{
        "bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
    },
})
```

## API Key、Basic 和 OAuth2 Bearer

API Key 从请求头读取：

```go
auth := middleware.APIKeyHeader("X-API-Key", func(c *gin.Context, key string) (any, bool) {
    return "service-a", key == cfg.ServiceAPIKey
})

router := ninja.NewRouter("/internal",
    ninja.WithAPIKeyAuth("ApiKeyAuth", auth),
)
```

OpenAPI scheme：

```go
SecuritySchemes: map[string]ninja.SecurityScheme{
    "ApiKeyAuth": ninja.APIKeyHeaderSecurityScheme("X-API-Key"),
}
```

Basic：

```go
basic := middleware.HTTPBasicAuth("Admin", func(c *gin.Context, username, password string) (any, bool) {
    return username, username == "admin" && password == cfg.AdminPassword
})

router.UseGin(basic)
```

OAuth2 scope 校验：

```go
oauth := middleware.OAuth2BearerAuthWithScopes(
    []string{"users:read"},
    validateOAuthToken,
)

ninja.Get(router, "/", listUsers,
    ninja.OAuth2AuthMiddleware(oauth, "users:read"),
)
```

认证中间件验证成功后，会把认证主体写入 Gin context。原生 Gin middleware 或 handler 可以用 `middleware.GetAuthPrincipal(c)` 读取。OAuth2 scope-aware 中间件还可以用 `middleware.GetAuthScopes(c)` 读取已授予 scope：

```go
router.UseGin(func(c *gin.Context) {
    principal := middleware.GetAuthPrincipal(c)
    scopes := middleware.GetAuthScopes(c)

    _ = principal
    _ = scopes

    c.Next()
})
```

在 gin-ninja handler 中，JWT 用户 ID 更常用 `ctx.GetUserID()`。如果需要读取 API Key、Basic、Bearer、OAuth2 中间件保存的完整 principal，可以通过底层 Gin context 使用 `middleware.GetAuthPrincipal(ctx.Context)`。

## Session 与 CSRF

Cookie Session 适合浏览器后台、管理工具或需要服务端签名状态的页面：

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret:   cfg.JWT.Secret,
    MaxAge:   86400,
    SameSite: http.SameSiteLaxMode,
}))
```

在普通 Gin 中间件或 handler 里使用：

```go
session := middleware.GetSession(c)
session.Set("user_id", "42")
```

CSRF 适合 Cookie 鉴权的浏览器场景；如果你只做 Bearer Token API，通常不需要 CSRF：

```go
api.UseGin(middleware.CSRF(&middleware.CSRFConfig{
    Secret: cfg.JWT.Secret,
}))
```

浏览器表单或前端请求需要带上服务端发出的 CSRF token。具体 token 读取和提交方式取决于你的前端约定。

## 安全响应头

默认安全头：

```go
api.UseGin(middleware.SecureHeaders(nil))
```

生产 HTTPS 且位于可信反向代理后，可以使用严格模式：

```go
api.UseGin(middleware.SecureHeadersStrict())
```

自定义：

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

只有在你信任前置代理会正确设置 `X-Forwarded-Proto` 时，才开启 `TrustForwardedProto`。

## 上传限制

上传接口应设置大小限制和 MIME 白名单：

```go
api.UseGin(middleware.UploadLimit(&middleware.UploadConfig{
    MaxSize:          10 << 20,
    AllowedMIMETypes: []string{"image/png", "image/jpeg", "application/pdf"},
}))
```

超出大小返回 `413`，类型不允许返回 `415`。`AllowedMIMETypes` 支持精确 MIME，也支持以 `/` 结尾的前缀匹配，例如 `"image/"` 可以匹配 `"image/png"`、`"image/jpeg"`。`GET` 请求不受上传限制影响。

## i18n

启用语言协商：

```go
api.UseGin(middleware.I18n())
```

框架会根据 `Accept-Language` 设置 locale，并用于校验错误等消息。handler 里可以用：

```go
locale := ctx.Locale()
message := ctx.T("validation.required")
```

## 安全检查清单

- CORS 生产环境显式配置域名，不使用 allow-all。
- JWT secret、Session secret、数据库密码走环境变量。
- 浏览器 Cookie 鉴权搭配 CSRF；纯 Bearer Token API 通常不需要 CSRF。
- 上传接口设置大小和类型限制。
- 公开接口只返回 DTO，不直接返回包含敏感字段的 model。
- 对内部接口、admin 接口加认证和审计日志。

上一篇：[配置、Bootstrap 与生命周期](./configuration.md) | 下一篇：[文件传输：上传与下载](./file-transfer.md)
