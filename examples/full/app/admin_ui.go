package app

import (
	"github.com/gin-gonic/gin"
	admin "github.com/shijl0925/gin-ninja/admin"
)

const fullExampleLogoURL = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Crect width='64' height='64' rx='16' fill='%230f172a'/%3E%3Cpath d='M18 39V25l14-8 14 8v14l-14 8-14-8Z' fill='%23f8fafc'/%3E%3Cpath d='M32 18v28M20 26l12 7 12-7' stroke='%230f172a' stroke-width='4' stroke-linecap='round'/%3E%3C/svg%3E"

// AdminUIConfig customizes the full example admin shell branding.
func AdminUIConfig() admin.UIConfig {
	return admin.UIConfig{
		Title:         "Gin Ninja Full Admin",
		BrandName:     "Gin Ninja Studio",
		BrandSubtitle: "Full example admin",
		LogoText:      "GN",
		LogoURL:       fullExampleLogoURL,
	}
}

// ServeAdminPrototype returns the standalone admin demo shell used by /admin, /admin/login,.
func ServeAdminPrototype(c *gin.Context) {
	admin.NewUIHandler(AdminUIConfig())(c)
}
