package middleware

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func NewResponseLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		if strings.Contains(c.Writer.Header().Get("Content-Type"), "application/json") {
			zap.L().Debug("Response Body",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Any("response_body", w.body),
			)
		}
	}
}
