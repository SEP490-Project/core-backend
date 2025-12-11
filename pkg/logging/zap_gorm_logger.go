package logging

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ZapGormLogger implements the gorm/logger.Interface using Zap
type ZapGormLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  logger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

func NewZapGormLogger(zapLogger *zap.Logger) ZapGormLogger {
	return ZapGormLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  logger.Info, // Log everything by default (tweak as needed)
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode sets the log level
func (l ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := l
	newLogger.LogLevel = level
	return &newLogger
}

// Info prints info
func (l ZapGormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Sugar().Infof(msg, data...)
	}
}

// Warn prints warn messages
func (l ZapGormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Warn {
		l.ZapLogger.Sugar().Warnf(msg, data...)
	}
}

// Error prints error messages
func (l ZapGormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Error {
		l.ZapLogger.Sugar().Errorf(msg, data...)
	}
}

// Trace is where the SQL query logging happens
func (l ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)

	// Retrieve SQL string and rows affected
	sql, rows := fc()

	// Common fields for all DB logs
	fields := []zap.Field{
		zap.String("type", "sql"),
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("latency", elapsed),
		zap.String("latency_human", elapsed.String()),
	}

	// 1. Handle Errors
	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		l.ZapLogger.Error("Database Error", append(fields, zap.Error(err))...)
		return
	}

	// 2. Handle Slow Queries
	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn {
		l.ZapLogger.Warn("Slow SQL Query", fields...)
		return
	}

	// 3. Handle Standard Logging
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Info("SQL Query", fields...)
	}
}
