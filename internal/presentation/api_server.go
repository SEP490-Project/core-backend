package presentation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"core-backend/config"
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/presentation/handler"
	"core-backend/internal/presentation/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type APIServer struct {
	router             *Router
	handlerRegistry    *handler.HandlerRegistry
	middlewareRegistry *middleware.MiddlewareRegistry
	serviceRegistry    *service.ServiceRegistry
	databaseRegistry   *gorm_repository.DatabaseRegistry
	server             *http.Server
}

func NewAPIServer() *APIServer {
	db := persistence.InitDB()

	databaseRegistry := gorm_repository.NewDatabaseRegistry(db)
	serviceRegistry := service.NewServiceRegistry(databaseRegistry)
	handlerRegistry := handler.NewHandlerRegistry(serviceRegistry)
	middlewareRegistry := middleware.NewMiddlewareRegistry(serviceRegistry)

	return &APIServer{
		databaseRegistry:   databaseRegistry,
		serviceRegistry:    serviceRegistry,
		handlerRegistry:    handlerRegistry,
		middlewareRegistry: middlewareRegistry,
		router:             NewRouter(handlerRegistry, middlewareRegistry),
	}
}

func (s *APIServer) Start() error {
	serverConfig := config.GetAppConfig().Server
	if serverConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if serverConfig.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		panic("Invalid server environment, valid options are 'production' or 'development'")
	}

	engine := gin.New()

	s.router.SetupRoutes(engine)
	s.router.SetupV1Routes(engine)

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

	// Create a deadline to wait for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
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
