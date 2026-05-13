package middleware

import "github.com/gin-gonic/gin"

func cookieSecureByDefault() bool {
	return gin.Mode() == gin.ReleaseMode
}
