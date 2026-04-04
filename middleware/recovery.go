package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/shijl0925/gin-ninja/pkg/response"
)

// Recovery returns a gin middleware that recovers from panics and logs the
// stack trace using the supplied *zap.Logger.
//
//	api.Engine().Use(middleware.Recovery(logger.Global()))
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error("panic recovered",
					zap.Any("error", r),
					zap.ByteString("stack", stack),
					zap.String("request_id", GetRequestID(c)),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					response.Fail(response.CodeError, "internal server error"))
			}
		}()
		c.Next()
	}
}
