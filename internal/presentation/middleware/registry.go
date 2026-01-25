// Package middleware provides middlewares for the application.
package middleware

import (
	"core-backend/config"
	"core-backend/internal/application"

	"github.com/gin-gonic/gin"
)

type MiddlewareRegistry struct {
	config      *config.AppConfig
	Recovery    gin.HandlerFunc
	Timeout     gin.HandlerFunc
	RequestID   gin.HandlerFunc
	CORS        gin.HandlerFunc
	Logging     gin.HandlerFunc
	Auth        *AuthMiddleware
	CSRF        *CSRFMiddleware // T111: CSRF protection
	ResponseLog gin.HandlerFunc
}

func NewMiddlewareRegistry(applicationRegistry *application.ApplicationRegistry) *MiddlewareRegistry {
	return &MiddlewareRegistry{
		config:      applicationRegistry.InfrastructureRegistry.Config,
		Recovery:    NewRecoveryMiddleware(),
		Timeout:     NewTimeoutMiddleware(),
		RequestID:   NewRequestIDMiddleware(),
		CORS:        NewCORSMiddleware(),
		Logging:     NewLoggingMiddleware(),
		Auth:        NewAuthMiddleware(applicationRegistry.JWTService),
		CSRF:        NewCSRFMiddleware(config.GetAppConfig().CORS.AllowedOrigins, false), // Non-strict mode for API compatibility
		ResponseLog: NewResponseLogMiddleware(),
	}
}

func (reg *MiddlewareRegistry) ApplyGlobalMiddlewares(r *gin.Engine) {
	r.Use(reg.Recovery)
	r.Use(reg.RequestID)
	r.Use(reg.Logging)
	r.Use(reg.CORS)
	// r.Use(reg.Timeout)
	if reg.config.Log.Level == "debug" && reg.config.Server.Environment != "production" {
		r.Use(reg.ResponseLog)
	}
}
