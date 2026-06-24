package admin

import (
	"embed"
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UIConfig configures the default admin UI shell routes and API endpoint paths.
type UIConfig struct {
	Title         string
	BrandName     string
	LogoText      string
	Locale        string
	DefaultTheme  string
	TokenStorage  string
	APIBasePath   string
	AuthLoginPath string
	AuthMePath    string
	AdminPath     string
	LoginPath     string

	// TokenExtractExpr is a restricted JavaScript-like payload path expression
	// used by the UI to read the token from a login response. It is serialized as
	// data and evaluated by a small allowlisted extractor, not injected as code.
	// Default: "payload.token"
	// Example for {"data":{"accessToken":"…"}}: "payload.data && payload.data.accessToken"
	TokenExtractExpr string

	// UserNameExtractExpr reads the display name from a login response.
	// It uses the same restricted payload path syntax as TokenExtractExpr.
	// Default: "payload.name"
	UserNameExtractExpr string

	// UserIDExtractExpr reads the user ID from a login response.
	// It uses the same restricted payload path syntax as TokenExtractExpr.
	// Default: "payload.user_id || payload.userID"
	UserIDExtractExpr string
}

// DefaultUIConfig returns the built-in admin UI route and API defaults.
func DefaultUIConfig() UIConfig {
	return UIConfig{
		Title:         "Gin Ninja Admin",
		BrandName:     "Gin Ninja",
		LogoText:      "G",
		Locale:        "en",
		DefaultTheme:  "light",
		TokenStorage:  "local",
		APIBasePath:   "/api/v1/admin",
		AuthLoginPath: "/api/v1/auth/login",
		AuthMePath:    "/api/v1/auth/me",
		AdminPath:     "/admin",
		LoginPath:     "/admin/login",
	}
}

func normalizeUIConfig(cfg UIConfig) UIConfig {
	defaults := DefaultUIConfig()
	if strings.TrimSpace(cfg.Title) == "" {
		cfg.Title = defaults.Title
	}
	if strings.TrimSpace(cfg.BrandName) == "" {
		cfg.BrandName = defaults.BrandName
	}
	if strings.TrimSpace(cfg.LogoText) == "" {
		cfg.LogoText = defaults.LogoText
	}
	if strings.TrimSpace(cfg.Locale) == "" {
		cfg.Locale = defaults.Locale
	}
	switch strings.ToLower(strings.TrimSpace(cfg.DefaultTheme)) {
	case "dark", "system":
		cfg.DefaultTheme = strings.ToLower(strings.TrimSpace(cfg.DefaultTheme))
	default:
		cfg.DefaultTheme = defaults.DefaultTheme
	}
	switch strings.ToLower(strings.TrimSpace(cfg.TokenStorage)) {
	case "session":
		cfg.TokenStorage = "session"
	default:
		cfg.TokenStorage = defaults.TokenStorage
	}
	if strings.TrimSpace(cfg.APIBasePath) == "" {
		cfg.APIBasePath = defaults.APIBasePath
	}
	if strings.TrimSpace(cfg.AuthLoginPath) == "" {
		cfg.AuthLoginPath = defaults.AuthLoginPath
	}
	if strings.TrimSpace(cfg.AuthMePath) == "" {
		cfg.AuthMePath = defaults.AuthMePath
	}
	if strings.TrimSpace(cfg.AdminPath) == "" {
		cfg.AdminPath = defaults.AdminPath
	}
	if strings.TrimSpace(cfg.LoginPath) == "" {
		cfg.LoginPath = defaults.LoginPath
	}
	if strings.TrimSpace(cfg.TokenExtractExpr) == "" {
		cfg.TokenExtractExpr = "payload.token"
	}
	if strings.TrimSpace(cfg.UserNameExtractExpr) == "" {
		cfg.UserNameExtractExpr = "payload.name"
	}
	if strings.TrimSpace(cfg.UserIDExtractExpr) == "" {
		cfg.UserIDExtractExpr = "payload.user_id || payload.userID"
	}
	return cfg
}

func renderUIHTML(cfg UIConfig) string {
	cfg = normalizeUIConfig(cfg)
	template := strings.NewReplacer(
		"__GIN_NINJA_ADMIN_INLINE_CSS__", adminCSS,
		"__GIN_NINJA_ADMIN_INLINE_JS__", adminJS,
	).Replace(adminHTMLTemplate)
	replacer := strings.NewReplacer(
		"__GIN_NINJA_ADMIN_TITLE__", html.EscapeString(cfg.Title),
		"__GIN_NINJA_ADMIN_BRAND_NAME__", html.EscapeString(cfg.BrandName),
		"__GIN_NINJA_ADMIN_LOGO_TEXT__", html.EscapeString(shortLogoText(cfg.LogoText)),
		"__GIN_NINJA_ADMIN_TITLE_JS__", jsonString(cfg.Title),
		"__GIN_NINJA_ADMIN_BRAND_NAME_JS__", jsonString(cfg.BrandName),
		"__GIN_NINJA_ADMIN_LOGO_TEXT_JS__", jsonString(shortLogoText(cfg.LogoText)),
		"__GIN_NINJA_ADMIN_LOCALE_ATTR__", html.EscapeString(cfg.Locale),
		"__GIN_NINJA_ADMIN_LOCALE__", jsonString(cfg.Locale),
		"__GIN_NINJA_ADMIN_DEFAULT_THEME__", jsonString(cfg.DefaultTheme),
		"__GIN_NINJA_ADMIN_TOKEN_STORAGE__", jsonString(cfg.TokenStorage),
		"__GIN_NINJA_ADMIN_API_BASE__", jsonString(cfg.APIBasePath),
		"__GIN_NINJA_ADMIN_AUTH_LOGIN_HINT__", html.EscapeString(cfg.AuthLoginPath),
		"__GIN_NINJA_ADMIN_PAGE_PATH_ATTR__", html.EscapeString(cfg.AdminPath),
		"__GIN_NINJA_ADMIN_AUTH_LOGIN_PATH__", jsonString(cfg.AuthLoginPath),
		"__GIN_NINJA_ADMIN_AUTH_ME_PATH__", jsonString(cfg.AuthMePath),
		"__GIN_NINJA_ADMIN_PAGE_PATH__", jsonString(cfg.AdminPath),
		"__GIN_NINJA_ADMIN_LOGIN_PATH__", jsonString(cfg.LoginPath),
		"__GIN_NINJA_ADMIN_TOKEN_EXTRACT_EXPR__", jsonString(cfg.TokenExtractExpr),
		"__GIN_NINJA_ADMIN_USER_NAME_EXTRACT_EXPR__", jsonString(cfg.UserNameExtractExpr),
		"__GIN_NINJA_ADMIN_USER_ID_EXTRACT_EXPR__", jsonString(cfg.UserIDExtractExpr),
	)
	return replacer.Replace(template)
}

func shortLogoText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "G"
	}
	if len([]rune(value)) > 3 {
		return string([]rune(value)[:3])
	}
	return value
}

func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

// NewUIHandler returns the built-in admin UI shell handler.
func NewUIHandler(cfg UIConfig) gin.HandlerFunc {
	html := []byte(renderUIHTML(cfg))
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", html)
	}
}

// ServeDefaultUI serves the built-in admin UI shell with default paths.
func ServeDefaultUI(c *gin.Context) {
	NewUIHandler(DefaultUIConfig())(c)
}

// MountUI mounts the built-in admin UI shell on the configured page routes.
func MountUI(routes gin.IRoutes, cfg UIConfig) {
	cfg = normalizeUIConfig(cfg)
	handler := NewUIHandler(cfg)
	registered := map[string]struct{}{}
	for _, path := range []string{cfg.LoginPath, cfg.AdminPath} {
		if _, exists := registered[path]; exists || strings.TrimSpace(path) == "" {
			continue
		}
		routes.GET(path, handler)
		registered[path] = struct{}{}
	}
}

//go:embed assets/admin.html assets/admin.css assets/admin.js
var adminAssetFS embed.FS

var (
	adminHTMLTemplate = mustReadAdminAsset("assets/admin.html")
	adminCSS          = mustReadAdminAsset("assets/admin.css")
	adminJS           = mustReadAdminAsset("assets/admin.js")
)

func mustReadAdminAsset(name string) string {
	data, err := adminAssetFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(data)
}
