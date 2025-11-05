// Package application defines the ApplicationRegistry struct that aggregates various services used in the application.
package application

import (
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/jobs"
	"core-backend/internal/infrastructure/scheduler"
	infraService "core-backend/internal/infrastructure/service"
)

type ApplicationRegistry struct {
	configs                       *config.AppConfig
	DatabaseRegistry              *gormrepository.DatabaseRegistry
	InfrastructureRegistry        *infrastructure.InfrastructureRegistry
	JWTService                    iservice.JWTService
	FileService                   iservice.FileService
	AuthService                   iservice.AuthService
	UserService                   iservice.UserService
	ProductService                iservice.ProductService
	BrandService                  iservice.BrandService
	StateTransferService          iservice.StateTransferService
	ContractService               iservice.ContractService
	CampaignService               iservice.CampaignService
	ModifiedHistoryService        iservice.ModifiedHistoryService
	ProductCategoryService        iservice.ProductCategoryService
	AdminConfigService            iservice.AdminConfigService
	ContractPaymentService        iservice.ContractPaymentService
	ConceptService                iservice.ConceptService
	OrderService                  iservice.OrderService
	ChannelService                iservice.ChannelService
	ContentService                iservice.ContentService
	BlogService                   iservice.BlogService
	TaskService                   iservice.TaskService
	DeviceTokenService            iservice.DeviceTokenService
	NotificationService           iservice.NotificationService
	LocationService               iservice.LocationService
	TagService                    iservice.TagService
	AffiliateLinkService          iservice.AffiliateLinkService
	ClickTrackingService          iservice.ClickTrackingService
	AffiliateLinkAnalyticsService iservice.AffiliateLinkAnalyticsService
	PaymentTransactionService     iservice.PaymentTransactionService

	//Manual Scheduler Trigger
	LocationSchedule scheduler.TaskScheduler
}

func NewApplicationRegistry(
	configs *config.AppConfig,
	databaseRegistry *gormrepository.DatabaseRegistry,
	infrastructureRegistry *infrastructure.InfrastructureRegistry,
) *ApplicationRegistry {
	jwtService := service.NewJwtService()

	affiliateLinkService := service.NewAffiliateLinkService(
		databaseRegistry.AffiliateLinkRepository,
		databaseRegistry.ContractRepository,
		databaseRegistry.ContentRepository,
		databaseRegistry.ChannelRepository,
		configs.Server.BaseURL,
	)
	clickTrackingService := service.NewClickTrackingService(
		databaseRegistry.AffiliateLinkRepository,
		affiliateLinkService, // Pass the service for validation
		infrastructureRegistry.ValkeyCache,
		infrastructureRegistry.RabbitMQ,
	)
	affiliateLinkAnalyticsService := service.NewAffiliateLinkAnalyticsService(
		databaseRegistry.ClickEventRepository,
		databaseRegistry.KPIMetricsRepository,
		databaseRegistry.AffiliateLinkRepository,
		databaseRegistry.ContractRepository,
	)
	contentService := service.NewContentService(
		databaseRegistry.ContentRepository,
		databaseRegistry.BlogRepository,
		databaseRegistry.ContentChannelRepository,
		databaseRegistry.ChannelRepository,
		databaseRegistry.TaskRepository,
		infrastructureRegistry.UnitOfWork,
		affiliateLinkService,
	)

	return &ApplicationRegistry{
		configs:                       configs,
		DatabaseRegistry:              databaseRegistry,
		InfrastructureRegistry:        infrastructureRegistry,
		JWTService:                    jwtService,
		FileService:                   infraService.NewFileService(infrastructureRegistry.ThirdPartyStorage, infrastructureRegistry.RabbitMQ),
		DeviceTokenService:            service.NewDeviceTokenService(databaseRegistry.DeviceTokenRepository),
		AuthService:                   service.NewAuthService(jwtService, databaseRegistry.UserRepository, databaseRegistry.LoggedSessionRepository, service.NewDeviceTokenService(databaseRegistry.DeviceTokenRepository)),
		UserService:                   service.NewUserService(databaseRegistry.UserRepository),
		ProductService:                service.NewProductService(databaseRegistry, infrastructureRegistry.ThirdPartyStorage, infrastructureRegistry.RabbitMQ),
		BrandService:                  service.NewBrandService(databaseRegistry.BrandRepository),
		StateTransferService:          service.NewStateTransferService(databaseRegistry, infrastructureRegistry.UnitOfWork, infrastructureRegistry.RabbitMQ),
		ContractService:               service.NewContractService(databaseRegistry),
		CampaignService:               service.NewCampaignService(databaseRegistry.CampaignRepository, databaseRegistry.ContractRepository),
		ModifiedHistoryService:        service.NewModifiedHistoryService(databaseRegistry.ModifiedHistoryRepository),
		ProductCategoryService:        service.NewProductCategoryService(databaseRegistry.ProductCategoryRepository),
		AdminConfigService:            service.NewAdminConfigService(&configs.AdminConfig, databaseRegistry.AdminConfigRepository),
		ContractPaymentService:        service.NewContractPaymentService(databaseRegistry),
		ConceptService:                service.NewConceptService(databaseRegistry.ConceptRepository),
		OrderService:                  service.NewOrderService(configs, databaseRegistry, infrastructureRegistry),
		ChannelService:                service.NewChannelService(databaseRegistry.ChannelRepository),
		ContentService:                contentService,
		BlogService:                   service.NewBlogService(databaseRegistry.BlogRepository, databaseRegistry.ContentRepository),
		TaskService:                   service.NewTaskService(databaseRegistry.TaskRepository),
		NotificationService:           service.NewNotificationService(databaseRegistry.NotificationRepository),
		LocationService:               service.NewLocationService(databaseRegistry),
		TagService:                    service.NewTagService(databaseRegistry.TagRepository),
		AffiliateLinkService:          affiliateLinkService,
		ClickTrackingService:          clickTrackingService,
		AffiliateLinkAnalyticsService: affiliateLinkAnalyticsService,
		PaymentTransactionService:     service.NewPaymentTransactionService(databaseRegistry.PaymentTransactionRepository, infrastructureRegistry.ProxiesRegistry.PayOSProxy),

		//Manual Scheduler Trigger
		LocationSchedule: scheduler.NewLocationSyncScheduler(configs, infrastructureRegistry.DB),
	}
}

// RegisterApplicationLayerJobs registers cron jobs that depend on application services
func (r *ApplicationRegistry) RegisterApplicationLayerJobs() {
	// Register PayOS Expiry Check Job
	if r.InfrastructureRegistry.CronJobsRegistry != nil {
		payosExpiryJob := jobs.NewPayOSExpiryCheckJob(
			r.PaymentTransactionService,
			r.InfrastructureRegistry.CronJobsRegistry.CronScheduler,
			&r.configs.AdminConfig,
		)
		r.InfrastructureRegistry.CronJobsRegistry.RegisterJob("payos_expiry_check_job", payosExpiryJob)
		r.InfrastructureRegistry.CronJobsRegistry.PayOSExpiryCheckJob = payosExpiryJob
	}
}
