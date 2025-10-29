// Package presentation implements the API server and its components.
package presentation

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/presentation/consumer"
	"core-backend/internal/presentation/handler"
	"core-backend/internal/presentation/middleware"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type APIServer struct {
	router                 *Router
	handlerRegistry        *handler.HandlerRegistry
	middlewareRegistry     *middleware.MiddlewareRegistry
	serviceRegistry        *application.ApplicationRegistry
	consumerRegistry       *consumer.ConsumerRegistry
	databaseRegistry       *gormrepository.DatabaseRegistry
	infrastructureRegistry *infrastructure.InfrastructureRegistry
	wsServer               *WebSocketServer
	server                 *http.Server
	ctx                    context.Context
	cancel                 context.CancelFunc
}

func NewAPIServer() *APIServer {
	db := persistence.InitDB()
	s3Bucket := persistence.InitS3()
	s3StreamBucket := persistence.InitS3StreamingBucket()

	// Create registries in order
	databaseRegistry := gormrepository.NewDatabaseRegistry(db)
	infrastructureRegistry := infrastructure.NewInfrastructureRegistry(config.GetAppConfig(), db, s3Bucket, s3StreamBucket)
	serviceRegistry := application.NewApplicationRegistry(config.GetAppConfig(), databaseRegistry, infrastructureRegistry)
	handlerRegistry := handler.NewHandlerRegistry(serviceRegistry)
	middlewareRegistry := middleware.NewMiddlewareRegistry(serviceRegistry)
	consumerRegistry := consumer.NewConsumerRegistry(serviceRegistry, infrastructureRegistry, databaseRegistry)

	// Create WebSocket server
	wsServer := NewWebSocketServer()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	return &APIServer{
		databaseRegistry:       databaseRegistry,
		infrastructureRegistry: infrastructureRegistry,
		serviceRegistry:        serviceRegistry,
		handlerRegistry:        handlerRegistry,
		middlewareRegistry:     middlewareRegistry,
		consumerRegistry:       consumerRegistry,
		wsServer:               wsServer,
		router:                 NewRouter(handlerRegistry, middlewareRegistry),
		ctx:                    ctx,
		cancel:                 cancel,
	}
}

func (s *APIServer) Start() error {
	serverConfig := config.GetAppConfig().Server
	wsConfig := config.GetAppConfig().WebSocket

	switch serverConfig.Environment {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "development":
		gin.SetMode(gin.DebugMode)
	default:
		panic("Invalid server environment, valid options are 'production' or 'development'")
	}

	// Register RabbitMQ consumer handlers
	if err := s.registerRabbitMQConsumers(); err != nil {
		zap.L().Error("Failed to register RabbitMQ consumers", zap.Error(err))
		// Don't fail startup - RabbitMQ is optional
	}

	// Start background schedulers (location sync, etc.)
	s.infrastructureRegistry.StartSchedulers(s.ctx)

	// Start WebSocket server if enabled
	if wsConfig.Enabled {
		zap.L().Info("Starting WebSocket server...")
		s.wsServer.Start(s.ctx)
	}

	engine := gin.New()

	// Setup routes
	s.router.SetupRoutes(engine)

	// Setup WebSocket routes if enabled
	if wsConfig.Enabled {
		s.router.SetupWebSocketRoutes(engine, s.wsServer)
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", serverConfig.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  time.Duration(serverConfig.Timeout) * time.Second,
		WriteTimeout: time.Duration(serverConfig.Timeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for interrupt signal to terminate server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		zap.L().Info("Starting API server",
			zap.String("address", addr),
			zap.String("environment", serverConfig.Environment),
		)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Block until we receive our signal
	<-quit
	zap.L().Info("Shutting down API server...")

	// Cancel context to stop background services
	s.cancel()

	// Stop infrastructure services
	s.infrastructureRegistry.StopServices()

	// Create a deadline to wait for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown of HTTP server
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	zap.L().Info("API server stopped gracefully")
	return nil
}

func (s *APIServer) Stop() error {
	if s.server == nil {
		return nil
	}

	// Cancel context to stop background services
	s.cancel()

	// Stop infrastructure services
	s.infrastructureRegistry.StopServices()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// registerRabbitMQConsumers registers all consumer handlers with RabbitMQ
func (s *APIServer) registerRabbitMQConsumers() error {
	zap.L().Info("Registering RabbitMQ consumer handlers")

	// Map consumer names (from rabbitmq-config.yaml) to handler functions
	handlers := map[string]func(context.Context, []byte) error{
		"contract-create-consumer":         s.consumerRegistry.ContractCreateConsumer.Handle,
		"contract-create-payment-consumer": s.consumerRegistry.ContractCreatePaymentConsumer.Handle,
		"excel-import-products-consumer":   s.consumerRegistry.ExcelImportProductsConsumer.Handle,
		"notification-email-consumer":      s.consumerRegistry.NotificationEmailConsumer.Handle,
		"notification-push-consumer":       s.consumerRegistry.NotificationPushConsumer.Handle,
		"video-upload-consumer":            s.consumerRegistry.VideoUploadConsumer.Handle,
	}

	// Register handlers with RabbitMQ
	return s.infrastructureRegistry.RegisterRabbitMQConsumers(s.ctx, handlers)
}

func (s *APIServer) GetServer() *http.Server {
	return s.server
}

// GetAddr returns the server address
func (s *APIServer) GetAddr() string {
	if s.server != nil {
		return s.server.Addr
	}
	return fmt.Sprintf(":%d", config.GetAppConfig().Server.Port)
}

// IsRunning checks if the server is running
func (s *APIServer) IsRunning() bool {
	return s.server != nil
}
