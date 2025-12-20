package logging

import (
	"fmt"

	"go.uber.org/zap"
)

// AsynqLogger implements asynq.Logger interface using zap
type AsynqLogger struct{}

func (l *AsynqLogger) Debug(args ...any) {
	zap.L().Debug(fmt.Sprint(args...))
}

func (l *AsynqLogger) Info(args ...any) {
	zap.L().Info(fmt.Sprint(args...))
}

func (l *AsynqLogger) Warn(args ...any) {
	zap.L().Warn(fmt.Sprint(args...))
}

func (l *AsynqLogger) Error(args ...any) {
	zap.L().Error(fmt.Sprint(args...))
}

func (l *AsynqLogger) Fatal(args ...any) {
	zap.L().Fatal(fmt.Sprint(args...))
}
