// Package handler implements HTTP handlers for the application.
package handler

import (
	"core-backend/internal/infrastructure"
)

type HandlerRegistry struct {
	ServiceRegistry *infrastructure.ServiceRegistry
	AuthHandler     *AuthHandler
	UserHandler     *UserHandler
	HealthHandler   *HealthHandler
	ProductHandler  *ProductHandler
}

func NewHandlerRegistry(serviceRegistry *infrastructure.ServiceRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ServiceRegistry: serviceRegistry,
		AuthHandler:     NewAuthHandler(serviceRegistry.AuthService),
		UserHandler:     NewUserHandler(serviceRegistry.UserService),
		HealthHandler:   NewHealthHandler(serviceRegistry.InfrastructureRegistry),
		ProductHandler:  NewProductHandler(serviceRegistry.ProductService),
	}
}
