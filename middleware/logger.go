package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a gin middleware that logs every request using the supplied
// *zap.Logger.
//
//	api.Engine().Use(middleware.Logger(logger.Global()))
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()
		requestID := GetRequestID(c)
		errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}
		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}
		if requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}
		if errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
		}

		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}
