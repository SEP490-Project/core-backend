package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		logFields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.Duration("latency", latency),
			zap.Int("response_size", c.Writer.Size()),
		}

		if requestID := c.GetString("request_id"); requestID != "" {
			logFields = append(logFields, zap.String("request_id", requestID))
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				zap.L().Error(e, logFields...)
			}
		} else {
			switch {
			case statusCode >= 500:
				zap.L().Error("HTTP request completed with server error", logFields...)
			case statusCode >= 400:
				zap.L().Warn("HTTP request completed with client error", logFields...)
			default:
				zap.L().Info("HTTP request completed successfully", logFields...)
			}
		}
	}
}
