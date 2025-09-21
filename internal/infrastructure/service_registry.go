package infrastructure

import (
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure/gorm_repository"
	iservice "core-backend/internal/infrastructure/service"
)

type ServiceRegistry struct {
	DatabaseRegistry       *gorm_repository.DatabaseRegistry
	InfrastructureRegistry *InfrastructureRegistry
	JWTService             *service.JWTService
	AuthService            *service.AuthService
	UserService            *service.UserService
	ProductService         service.ProductService
}

func NewServiceRegistry(
	databaseRegistry *gorm_repository.DatabaseRegistry,
	infrastructureRegistry *InfrastructureRegistry,
) *ServiceRegistry {
	jwtService := service.NewJwtService()

	return &ServiceRegistry{
		DatabaseRegistry:       databaseRegistry,
		InfrastructureRegistry: infrastructureRegistry,
		JWTService:             jwtService,
		AuthService:            service.NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:            service.NewUserService(databaseRegistry.UserRepository),
		ProductService:         iservice.NewProductService(databaseRegistry.ProductRepository),
	}
}
