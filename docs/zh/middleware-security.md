# 中间件与安全

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 中间件

### 引擎级中间件

```go
api.UseGin(
    middleware.RequestID(),
    middleware.Recovery(log),
    middleware.Logger(log),
    middleware.CORSFromConfig(cfg.CORS),
    orm.Middleware(db),
)
```

生产环境必须使用 `middleware.CORSFromConfig(cfg.CORS)` 或显式的 `middleware.CORSConfig`，并在配置中显式设置 `allow_origins`。不要在生产环境使用 `middleware.CORS(nil)`：它会允许所有来源，并且在 Gin release mode 下会直接 panic。

### 路由级中间件

```go
protected := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
protected.UseGin(middleware.JWTAuthWithConfig(cfg.JWT))
```

### 常用内置中间件

- **JWT**：`middleware.JWTAuthWithConfig(...)`、`middleware.GenerateTokenWithConfig(...)`
- **i18n**：`middleware.I18n()`，支持 `en` / `zh`
- **Session**：HMAC-SHA256 签名 Cookie Session
- **CSRF**：双重提交 Cookie 模式
- **SecureHeaders**：统一设置安全响应头
- **UploadLimit**：限制请求体大小与 MIME 白名单
- **RequestID / Logger / Recovery / CORS**：常见基础中间件

### i18n

注册后会根据 `Accept-Language` 自动协商语言，并自动翻译校验错误。

```go
api.UseGin(middleware.I18n())

func myHandler(ctx *ninja.Context, in *MyInput) (*MyOutput, error) {
    locale := ctx.Locale()
    msg := ctx.T("not_found")
    _ = locale
    _ = msg
    return nil, nil
}
```

### Session / CSRF / 安全头

```go
api.UseGin(middleware.SessionMiddleware(&middleware.SessionConfig{
    Secret: "change-me-in-production",
    MaxAge: 86400,
    // Secure 在 gin.ReleaseMode 下默认开启，本地开发模式默认关闭。
    // 如果本地也使用 HTTPS，可显式设置 Secure: true。
    // 如需显式关闭，可设置 Secure: false, SecureSet: true。
    HTTPOnly: true,
}))

api.UseGin(middleware.CSRF(nil))
api.UseGin(middleware.SecureHeadersStrict())
```

生产建议：

- 使用强随机密钥，不要保留示例密钥
- 全链路启用 HTTPS，并正确传递 `X-Forwarded-Proto`
- Cookie 配置 `Secure`、`HTTPOnly`、合适的 `SameSite`；`Secure` 在 release 模式默认开启，本地 HTTP 开发默认关闭，也可通过 `SecureSet` / `CookieSecureSet` 显式覆盖
- 对所有修改类接口启用 CSRF 防护
- 上传接口同时配置大小上限和 MIME 白名单

---

上一篇: [配置、Bootstrap 与 ORM](./configuration.md) | 下一篇: [文件传输与 OpenAPI 控制](./files-and-openapi.md)
