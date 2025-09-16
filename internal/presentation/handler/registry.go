package handler

import "core-backend/internal/application/service"

type HandlerRegistry struct {
	ServiceRegistry *service.ServiceRegistry
	AuthHandler     *AuthHandler
	UserHandler     *UserHandler
}

func NewHandlerRegistry(serviceRegistry *service.ServiceRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ServiceRegistry: serviceRegistry,
		AuthHandler: NewAuthHandler(serviceRegistry.AuthService),
		UserHandler: NewUserHandler(serviceRegistry.UserService),
	}
}
