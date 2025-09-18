// Package handler implements HTTP handlers for the application.
package handler

import "core-backend/internal/application/service"

type HandlerRegistry struct {
	ServiceRegistry *service.ServiceRegistry
	AuthHandler     *AuthHandler
	UserHandler     *UserHandler
	HealthHandler   *HealthHandler
}

func NewHandlerRegistry(serviceRegistry *service.ServiceRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ServiceRegistry: serviceRegistry,
		AuthHandler:     NewAuthHandler(serviceRegistry.AuthService),
		UserHandler:     NewUserHandler(serviceRegistry.UserService),
		HealthHandler:   NewHealthHandler(serviceRegistry.InfrastructureRegistry),
	}
}
