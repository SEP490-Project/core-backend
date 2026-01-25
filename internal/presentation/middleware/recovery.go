package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with full context
				zap.L().Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
					zap.String("stack", string(debug.Stack())),
				)

				// Check if headers were already written
				if !c.Writer.Written() {
					// Return error response
					c.JSON(http.StatusInternalServerError, gin.H{
						"code":    500,
						"message": "Internal Server Error",
						"data":    nil,
					})
				} else {
					// If headers were already sent, we can only log
					zap.L().Warn("Cannot send error response, headers already written")
				}

				c.Abort()
			}
		}()

		c.Next()
	}
}
