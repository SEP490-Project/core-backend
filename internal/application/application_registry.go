// Package application defines the ApplicationRegistry struct that aggregates various services used in the application.
package application

import (
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
)

type ApplicationRegistry struct {
	DatabaseRegistry       *gormrepository.DatabaseRegistry
	InfrastructureRegistry *infrastructure.InfrastructureRegistry
	JWTService             iservice.JWTService
	AuthService            iservice.AuthService
	UserService            iservice.UserService
	ProductService         iservice.ProductService
	BrandService           iservice.BrandService
}

func NewApplicationRegistry(
	databaseRegistry *gormrepository.DatabaseRegistry,
	infrastructureRegistry *infrastructure.InfrastructureRegistry,
) *ApplicationRegistry {
	jwtService := service.NewJwtService()

	return &ApplicationRegistry{
		DatabaseRegistry:       databaseRegistry,
		InfrastructureRegistry: infrastructureRegistry,
		JWTService:             jwtService,
		AuthService:            service.NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:            service.NewUserService(databaseRegistry.UserRepository),
		ProductService:         service.NewProductService(databaseRegistry.ProductRepository),
		BrandService:           service.NewBrandService(databaseRegistry.BrandRepository),
	}
}
