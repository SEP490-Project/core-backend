// Package service provides various application services.
package service

import "core-backend/internal/infrastructure/gorm_repository"

type ServiceRegistry struct {
	DatabaseRegistry *gorm_repository.DatabaseRegistry
	JWTService       *JWTService
	AuthService      *AuthService
	UserService      *UserService
}

func NewServiceRegistry(databaseRegistry *gorm_repository.DatabaseRegistry) *ServiceRegistry {
	jwtService := NewJwtService()

	return &ServiceRegistry{
		DatabaseRegistry: databaseRegistry,
		JWTService:       jwtService,
		AuthService:      NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:      NewUserService(databaseRegistry.UserRepository),
	}
}
