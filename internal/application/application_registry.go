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
	StateTransferService   iservice.StateTransferService
	ContractService        iservice.ContractService
	CampaignService        iservice.CampaignService
	ModifiedHistoryService iservice.ModifiedHistoryService
	ProductCategoryService iservice.ProductCategoryService
	AdminConfigService     iservice.AdminConfigService
	ContractPaymentService iservice.ContractPaymentService
	ConceptService         iservice.ConceptService
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
		FileService:            infraService.NewFileService(infrastructureRegistry.ThirdPartyStorage, infrastructureRegistry.RabbitMQ),
		AuthService:            service.NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository),
		UserService:            service.NewUserService(databaseRegistry.UserRepository),
		ProductService:         service.NewProductService(databaseRegistry),
		BrandService:           service.NewBrandService(databaseRegistry.BrandRepository),
		StateTransferService:   service.NewStateTransferService(databaseRegistry, infrastructureRegistry.UnitOfWork),
		ContractService:        service.NewContractService(databaseRegistry),
		CampaignService:        service.NewCampaignService(databaseRegistry.CampaignRepository),
		ModifiedHistoryService: service.NewModifiedHistoryService(databaseRegistry.ModifiedHistoryRepository),
		ProductCategoryService: service.NewProductCategoryService(databaseRegistry.ProductCategoryRepository),
		AdminConfigService:     service.NewAdminConfigService(databaseRegistry.AdminConfigRepository),
		ContractPaymentService: service.NewContractPaymentService(databaseRegistry),
		ConceptService:         service.NewConceptService(databaseRegistry.ConceptRepository),
	}
}
