// Package middleware provides middlewares for the application.
package middleware

import (
	"core-backend/internal/application/service"

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

func NewMiddlewareRegistry(serviceRegistry *service.ServiceRegistry) *MiddlewareRegistry {
	return &MiddlewareRegistry{
		Recovery:  NewReocoveryMiddleware(),
		Timeout:   NewTimeoutMiddleware(),
		RequestID: NewRequestIDMiddleware(),
		CORS:      NewCORSMiddleware(),
		Logging:   NewLoggingMiddleware(),
		Auth:      NewAuthMiddleware(serviceRegistry.JWTService),
	}
}

func (reg *MiddlewareRegistry) ApplyGlobalMiddlewares(r *gin.Engine) {
	r.Use(reg.Recovery)
	r.Use(reg.RequestID)
	r.Use(reg.CORS)
	r.Use(reg.Logging)
	r.Use(reg.Timeout)
}
