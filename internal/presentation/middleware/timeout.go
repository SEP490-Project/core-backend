package middleware

import (
	"context"
	"core-backend/config"
	"time"

	"github.com/gin-gonic/gin"
)

func NewTimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(config.GetAppConfig().Server.Timeout)*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			// Request completed within timeout
		case <-ctx.Done():
			// Request timed out
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(504, gin.H{
					"code":    504,
					"message": "Request Timeout",
					"data":    nil,
				})
				c.Abort()
			}
		}
	}
}
