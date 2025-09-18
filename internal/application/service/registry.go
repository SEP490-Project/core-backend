// Package service provides various application services.
package service

import (
	"core-backend/internal/infrastructure"
	"core-backend/internal/infrastructure/gorm_repository"
)

type ServiceRegistry struct {
	DatabaseRegistry        *gorm_repository.DatabaseRegistry
	InfrastructureRegistry *infrastructure.InfrastructureRegistry
	JWTService             *JWTService
	AuthService            *AuthService
	UserService            *UserService
}

func NewServiceRegistry(
	databaseRegistry *gorm_repository.DatabaseRegistry,
	infrastructureRegistry *infrastructure.InfrastructureRegistry,
) *ServiceRegistry {
	jwtService := NewJwtService()

	return &ServiceRegistry{
		DatabaseRegistry:        databaseRegistry,
		InfrastructureRegistry: infrastructureRegistry,
		JWTService:             jwtService,
		AuthService:            NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:            NewUserService(databaseRegistry.UserRepository),
	}
}
