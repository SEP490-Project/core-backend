package middleware

import (
	"context"
	"core-backend/config"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewTimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(config.GetAppConfig().Server.Timeout)*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		panicChan := make(chan any, 1)

		go func() {
			defer func() {
				if err := recover(); err != nil {
					panicChan <- err
				}
			}()

			c.Next()
			close(done)
		}()

		select {
		case p := <-panicChan:
			// Panic occurred in goroutine
			zap.L().Error("Panic in timeout middleware goroutine",
				zap.String("panic_type", fmt.Sprintf("%T", p)),
				zap.Any("panic_value", p),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.ByteString("stack", debug.Stack()),
			)

			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal Server Error",
					"data":    nil,
				})
			}
			c.Abort()

		case <-done:
			// Request completed successfully
			return

		case <-ctx.Done():
			// Timeout occurred
			zap.L().Warn("Request timeout",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)

			if !c.Writer.Written() {
				c.JSON(http.StatusGatewayTimeout, gin.H{
					"code":    504,
					"message": "Request timeout",
					"data":    nil,
				})
			}
			c.Abort()
		}
	}
}
