package app

import (
	"github.com/gin-gonic/gin"
	admin "github.com/shijl0925/gin-ninja/admin"
)

// ServeAdminPrototype returns the standalone admin demo shell used by /admin, /admin/login, and /admin-prototype.
func ServeAdminPrototype(c *gin.Context) {
	admin.ServeDefaultUI(c)
}
