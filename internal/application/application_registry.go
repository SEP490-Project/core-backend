// Package application defines the ApplicationRegistry struct that aggregates various services used in the application.
package application

import (
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	infraService "core-backend/internal/infrastructure/service"
)

type ApplicationRegistry struct {
	configs                *config.AppConfig
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
	OrderService           iservice.OrderService
	ChannelService         iservice.ChannelService
	ContentService         iservice.ContentService
	BlogService            iservice.BlogService
	TaskService            iservice.TaskService
	DeviceTokenService     iservice.DeviceTokenService
	NotificationService    iservice.NotificationService
}

func NewApplicationRegistry(
	configs *config.AppConfig,
	databaseRegistry *gormrepository.DatabaseRegistry,
	infrastructureRegistry *infrastructure.InfrastructureRegistry,
) *ApplicationRegistry {
	jwtService := service.NewJwtService()

	return &ApplicationRegistry{
		configs:                configs,
		DatabaseRegistry:       databaseRegistry,
		InfrastructureRegistry: infrastructureRegistry,
		JWTService:             jwtService,
		FileService:            infraService.NewFileService(infrastructureRegistry.ThirdPartyStorage, infrastructureRegistry.RabbitMQ),
		DeviceTokenService:     service.NewDeviceTokenService(databaseRegistry.DeviceTokenRepository),
		AuthService: service.NewAuthService(
			jwtService,
			databaseRegistry.UserRepository,
			databaseRegistry.LoggedSessionRepository,
			service.NewDeviceTokenService(databaseRegistry.DeviceTokenRepository),
		),
		UserService:            service.NewUserService(databaseRegistry.UserRepository),
		ProductService:         service.NewProductService(databaseRegistry),
		BrandService:           service.NewBrandService(databaseRegistry.BrandRepository),
		StateTransferService:   service.NewStateTransferService(databaseRegistry, infrastructureRegistry.UnitOfWork, infrastructureRegistry.RabbitMQ),
		ContractService:        service.NewContractService(databaseRegistry),
		CampaignService:        service.NewCampaignService(databaseRegistry.CampaignRepository, databaseRegistry.ContractRepository),
		ModifiedHistoryService: service.NewModifiedHistoryService(databaseRegistry.ModifiedHistoryRepository),
		ProductCategoryService: service.NewProductCategoryService(databaseRegistry.ProductCategoryRepository),
		AdminConfigService:     service.NewAdminConfigService(&configs.AdminConfig, databaseRegistry.AdminConfigRepository),
		ContractPaymentService: service.NewContractPaymentService(databaseRegistry),
		ConceptService:         service.NewConceptService(databaseRegistry.ConceptRepository),
		OrderService:           service.NewOrderService(databaseRegistry, infrastructureRegistry.PayOsService),
		ChannelService:         service.NewChannelService(databaseRegistry.ChannelRepository),
		ContentService: service.NewContentService(
			databaseRegistry.ContentRepository,
			databaseRegistry.BlogRepository,
			databaseRegistry.ContentChannelRepository,
			databaseRegistry.ChannelRepository,
			databaseRegistry.TaskRepository,
			infrastructureRegistry.UnitOfWork,
		),
		BlogService: service.NewBlogService(
			databaseRegistry.BlogRepository,
			databaseRegistry.ContentRepository,
		),
		TaskService:         service.NewTaskService(databaseRegistry.TaskRepository),
		NotificationService: service.NewNotificationService(databaseRegistry.NotificationRepository),
	}
}
