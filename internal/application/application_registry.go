// Package application defines the ApplicationRegistry struct that aggregates various services used in the application.
package application

import (
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	infraService "core-backend/internal/infrastructure/service"
)

type ApplicationRegistry struct {
	DatabaseRegistry       *gormrepository.DatabaseRegistry
	InfrastructureRegistry *infrastructure.InfrastructureRegistry
	JWTService             iservice.JWTService
	FileService            iservice.FileService
	AuthService            iservice.AuthService
	UserService            iservice.UserService
	ProductService         iservice.ProductService
	BrandService           iservice.BrandService
	TaskService            iservice.StateTransferService
	ContractService        iservice.ContractService
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
		FileService:            infraService.NewFileService(infrastructureRegistry.S3Repository),
		AuthService:            service.NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:            service.NewUserService(databaseRegistry.UserRepository),
		ProductService:         service.NewProductService(databaseRegistry.ProductRepository, databaseRegistry.ProductVariantRepository, databaseRegistry.TaskRepository, databaseRegistry.BrandRepository, databaseRegistry.ProductCategoryRepository),
		BrandService:           service.NewBrandService(databaseRegistry.BrandRepository),
		TaskService:            service.NewStateTransferService(databaseRegistry.ContractRepository, databaseRegistry.CampaignRepository, databaseRegistry.MilestoneRepository, databaseRegistry.TaskRepository, databaseRegistry.ProductRepository, infrastructureRegistry.UnitOfWork),
		ContractService:        service.NewContractService(databaseRegistry.ContractRepository),
	}
}
