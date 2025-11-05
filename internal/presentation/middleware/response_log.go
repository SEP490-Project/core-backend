package middleware

import (
	"bytes"
	"encoding/json"

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

		if c.Writer.Header().Get("Content-Type") == "application/json" {
			var pretty bytes.Buffer
			var body string
			if err := json.Indent(&pretty, []byte(body), "", "  "); err == nil {
				body = pretty.String()
			}
			zap.L().Debug("Response Body",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("response_body", w.body.String()),
			)
		}
	}
}
