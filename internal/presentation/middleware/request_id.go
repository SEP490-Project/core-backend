package middleware

import (
	"core-backend/pkg/logging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "X-Request-ID"

func NewRequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDKey)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set(RequestIDKey, requestID)

		logging.SetRequestID(requestID)
		defer logging.CleanupRequestID()

		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	return c.GetString("request_id")
}
