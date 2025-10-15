// Package middleware provides middlewares for the application.
package middleware

import (
	"core-backend/internal/application"

	"github.com/gin-gonic/gin"
)

type MiddlewareRegistry struct {
	Recovery  gin.HandlerFunc
	Timeout   gin.HandlerFunc
	RequestID gin.HandlerFunc
	CORS      gin.HandlerFunc
	Logging   gin.HandlerFunc
	Auth      *AuthMiddleware
}

func NewMiddlewareRegistry(applicationRegistry *application.ApplicationRegistry) *MiddlewareRegistry {
	return &MiddlewareRegistry{
		Recovery:  NewRecoveryMiddleware(),
		Timeout:   NewTimeoutMiddleware(),
		RequestID: NewRequestIDMiddleware(),
		CORS:      NewCORSMiddleware(),
		Logging:   NewLoggingMiddleware(),
		Auth:      NewAuthMiddleware(applicationRegistry.JWTService),
	}
}

func (reg *MiddlewareRegistry) ApplyGlobalMiddlewares(r *gin.Engine) {
	r.Use(reg.Recovery)
	r.Use(reg.RequestID)
	r.Use(reg.Logging)
	r.Use(reg.CORS)
	r.Use(reg.Timeout)
}
