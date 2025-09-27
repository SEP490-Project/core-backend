// Package presentation implements the API server and its components.
package presentation

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application"
	"core-backend/internal/infrastructure"
	"core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
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

	// Create registries
	databaseRegistry := gormrepository.NewDatabaseRegistry(db)
	infrastructureRegistry := infrastructure.NewInfrastructureRegistry(db, s3Bucket)
	serviceRegistry := application.NewApplicationRegistry(databaseRegistry, infrastructureRegistry)
	handlerRegistry := handler.NewHandlerRegistry(serviceRegistry)
	middlewareRegistry := middleware.NewMiddlewareRegistry(serviceRegistry)

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

	// Start background services
	zap.L().Info("Starting background services...")
	//s.infrastructureRegistry.StartBackgroundServices(s.ctx)

	// Start WebSocket server if enabled
	if wsConfig.Enabled {
		zap.L().Info("Starting WebSocket server...")
		s.wsServer.Start(s.ctx)
	}

	engine := gin.New()

	// Setup routes
	s.router.SetupRoutes(engine)
	s.router.SetupV1Routes(engine)

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
