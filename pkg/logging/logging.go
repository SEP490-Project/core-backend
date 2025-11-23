// Package logging provides a simple logging interface.
package logging

import (
	"context"
	"core-backend/config"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var loggerProvider *sdklog.LoggerProvider

func InitLogger() error {
	cfg := config.GetAppConfig()
	logLevel := convertLogLevel(cfg.Log.Level)

	consoleCore := createConsoleCore(logLevel)

	cores := []zapcore.Core{consoleCore}

	if cfg.Otel.Enabled {
		otelCore, err := createOtelCore()
		if err != nil {
			zap.L().Warn("Failed to create Otel Zap Core for logging: %s", zap.Error(err))
		} else {
			cores = append(cores, otelCore)
			zap.L().Info("OpenTelemetry logging enabled.")
		}
	}

	teeCore := zapcore.NewTee(cores...)
	contextAwareCore := NewContextAwareCore(teeCore)

	logLeveler := zap.New(
		contextAwareCore,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	zap.ReplaceGlobals(logLeveler)

	zap.L().Info("Logger initialized.", zap.String("level", cfg.Log.Level))

	return nil
}

func ShutdownLogger() {
	if loggerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := loggerProvider.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry logger provider: %v", err)
		}
	}
}

func convertLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func createConsoleCore(logLevel zapcore.Level) zapcore.Core {
	var encoder zapcore.Encoder
	var environment = config.GetAppConfig().Server.Environment
	if strings.ToLower(environment) == "development" {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	return zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), logLevel)
}

func createOtelCore() (zapcore.Core, error) {
	ctx := context.Background()
	cfg := config.GetAppConfig()

	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(cfg.Server.ServiceName)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create Otel resource: %w", err)
	}

	exporterOpts := []otlpgrpc.Option{
		otlpgrpc.WithEndpoint(cfg.Otel.Endpoint),
	}
	if cfg.Otel.Insecure {
		exporterOpts = append(exporterOpts, otlpgrpc.WithInsecure())
	}

	logExporter, err := otlpgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Otel Log Exporter: %w", err)
	}

	loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)

	otelZapCore := otelzap.NewCore(
		cfg.Otel.ServiceName, otelzap.WithVersion("v1.0.0"),
	)

	return otelZapCore, nil
}
