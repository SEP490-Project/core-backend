// Package main provides the main application entry point.
//
//	@title						SEP490 Core Backend API
//	@version					1.0
//	@description				This is the core backend service for SEP490 project with authentication, user management, and business features.
//	@termsOfService				http://swagger.io/terms/
//
//	@contact.name				API Support
//	@contact.url				http://www.swagger.io/support
//	@contact.email				support@swagger.io
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:8080
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
package main

import (
	"core-backend/config"
	"core-backend/internal/presentation"
	"core-backend/pkg/logging"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	if err := config.LoadConfig("./config"); err != nil {
		panic(err)
	}
	appConfig := config.GetAppConfig()
	loc, err := time.LoadLocation(appConfig.Server.Timezone)
	if err != nil {
		panic("Failed to load timezone: " + err.Error())
	} else {
		time.Local = loc
	}

	// Initialize logger
	err = logging.InitLogger()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	defer func() {
		zap.L().Sync()
		zap.S().Sync()
		logging.ShutdownLogger()
	}()

	zap.L().Info("Starting server...",
		zap.String("env", appConfig.Server.Environment),
		zap.Int("port", appConfig.Server.Port),
	)

	// Create and start API server
	server := presentation.NewAPIServer()

	// Start the server (this will block until shutdown)
	if err := server.Start(); err != nil {
		zap.L().Fatal("Server failed to start", zap.Error(err))
	}

	zap.L().Info("Server stopped gracefully")
}
