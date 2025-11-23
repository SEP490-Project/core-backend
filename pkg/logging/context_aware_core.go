package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ContextAwareCore struct {
	zapcore.Core
}

func NewContextAwareCore(c zapcore.Core) zapcore.Core {
	return &ContextAwareCore{Core: c}
}

func (c *ContextAwareCore) With(fields []zapcore.Field) zapcore.Core {
	return NewContextAwareCore(c.Core.With(fields))
}

func (c *ContextAwareCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Check if the current goroutine has a request_id bound to it
	if reqID := GetRequestID(); reqID != "" {
		// Prepend or append the request_id field
		fields = append(fields, zap.String("request_id", reqID))
	}

	return c.Core.Write(entry, fields)
}

func (c *ContextAwareCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}
