// Package handler implements HTTP handlers for the application.
package handler

import (
	"core-backend/config"
	"core-backend/internal/application"

	"github.com/go-playground/validator/v10"
)

type HandlerRegistry struct {
	ApplicationRegistry           *application.ApplicationRegistry
	AuthHandler                   *AuthHandler
	UserHandler                   *UserHandler
	HealthHandler                 *HealthHandler
	ProductHandler                *ProductHandler
	BrandHandler                  *BrandHandler
	FileHandler                   *S3Handler
	PayOsHandler                  *PayOsHandler
	PaymentTransactionsHandler    *PaymentTransactionsHandler
	StateHandler                  *StateHandler
	ContractHandler               *ContractHandler
	CampaignHandler               *CampaignHandler
	CategoryHandler               *ProductCategoryHandler
	ModifiedHistoryHandler        *ModifiedHistoryHandler
	AdminConfigHandler            *AdminConfigHandler
	ContractPaymentHandler        *ContractPaymentHandler
	ConceptHandler                *ConceptHandler
	OrderHandler                  *OrderHandler
	ChannelHandler                *ChannelHandler
	ContentHandler                *ContentHandler
	BlogHandler                   *BlogHandler
	LocationHandler               *LocationHandler
	TaskHandler                   *TaskHandler
	DeviceTokenHandler            *DeviceTokenHandler
	NotificationHandler           *NotificationHandler
	TagHandler                    *TagHandler
	GHNHandler                    *GHNHandler
	AffiliateLinkHandler          *AffiliateLinkHandler
	RedirectHandler               *RedirectHandler
	AffiliateLinkAnalyticsHandler *AffiliateLinkAnalyticsHandler
	PreOrderHandler               *PreOrderHandler
	MarketingAnalyticsHandler     *MarketingAnalyticsHandler
	SalesStaffAnalyticsHandler    *SalesStaffAnalyticsHandler
	ContentStaffAnalyticsHandler  *ContentStaffAnalyticsHandler
	BrandPartnerAnalyticsHandler  *BrandPartnerAnalyticsHandler
	AdminAnalyticsHandler         *AdminAnalyticsHandler
	FacebookSocialHandler         *FacebookSocialHandler
	TikTokSocialHandler           *TikTokSocialHandler
	AIHandler                     *AIHandler
	JobHandler                    *JobHandler
	RabbitMQHandler               *RabbitMQHandler
	TestHandler                   *TestHandler
	ScheduleHandler               *ScheduleHandler
	ContentScheduleHandler        *ContentScheduleHandler
	ContentEngagementHandler      *ContentEngagementHandler
	AlertHandler                  *AlertHandler
	SystemHandler                 *SystemHandler
	AsynqHandler                  *AsynqHandler
	CacheHandler                  *CacheHandler
	ProductOptionHandler          *ProductOptionHandler
	ViolationHandler              *ViolationHandler
}

func NewHandlerRegistry(applicationReg *application.ApplicationRegistry, appConfig *config.AppConfig) *HandlerRegistry {
	return &HandlerRegistry{
		ApplicationRegistry:           applicationReg,
		AuthHandler:                   NewAuthHandler(applicationReg.AuthService),
		UserHandler:                   NewUserHandler(applicationReg.UserService, applicationReg.InfrastructureRegistry.UnitOfWork),
		HealthHandler:                 NewHealthHandler(applicationReg.InfrastructureRegistry),
		ProductHandler:                NewProductHandler(applicationReg.ProductService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork),
		BrandHandler:                  NewBrandHandler(applicationReg.BrandService, applicationReg.InfrastructureRegistry.UnitOfWork),
		FileHandler:                   NewS3Handler(applicationReg.FileService),
		PayOsHandler:                  NewPayOsHandler(appConfig, applicationReg.PaymentTransactionService, applicationReg.StateTransferService, applicationReg.WebhookDataService, applicationReg.InfrastructureRegistry.ProxiesRegistry.PayOSProxy, applicationReg.ScheduleService, applicationReg.InfrastructureRegistry.UnitOfWork),
		PaymentTransactionsHandler:    NewPaymentTransactionsHandler(applicationReg.PaymentTransactionService),
		StateHandler:                  NewStateHandler(applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork, validator.New(), applicationReg.FileService),
		ContractHandler:               NewContractHandler(applicationReg.ContractService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.InfrastructureRegistry.RabbitMQ),
		CampaignHandler:               NewCampaignHandler(applicationReg.CampaignService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.InfrastructureRegistry.RabbitMQ),
		CategoryHandler:               NewCategoryHandler(applicationReg.ProductCategoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ModifiedHistoryHandler:        NewModifiedHistoryHandler(applicationReg.ModifiedHistoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		AdminConfigHandler:            NewAdminConfigHandler(applicationReg.AdminConfigService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContractPaymentHandler:        NewContractPaymentHandler(applicationReg.ContractPaymentService, applicationReg.PaymentTransactionService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ConceptHandler:                NewConceptHandler(applicationReg.ConceptService),
		OrderHandler:                  NewOrderHandler(applicationReg.OrderService, applicationReg.InfrastructureRegistry.ProxiesRegistry.GHNProxy, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.StateTransferService, applicationReg.FileService),
		ChannelHandler:                NewChannelHandler(applicationReg.ChannelService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContentHandler:                NewContentHandler(applicationReg, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.InfrastructureRegistry.RabbitMQ),
		BlogHandler:                   NewBlogHandler(applicationReg.BlogService, applicationReg.InfrastructureRegistry.UnitOfWork),
		TaskHandler:                   NewTaskHandler(applicationReg.TaskService, applicationReg.InfrastructureRegistry.UnitOfWork),
		DeviceTokenHandler:            NewDeviceTokenHandler(applicationReg.DeviceTokenService),
		NotificationHandler:           NewNotificationHandler(applicationReg.NotificationService),
		LocationHandler:               NewLocationHandler(applicationReg.LocationService, applicationReg.InfrastructureRegistry.LocationSyncTask),
		TagHandler:                    NewTagHandler(applicationReg.TagService, applicationReg.InfrastructureRegistry.UnitOfWork),
		GHNHandler:                    NewGHNHandler(applicationReg.InfrastructureRegistry.ProxiesRegistry.GHNProxy, applicationReg.InfrastructureRegistry.UnitOfWork),
		AffiliateLinkHandler:          NewAffiliateLinkHandler(applicationReg.AffiliateLinkService),
		RedirectHandler:               NewRedirectHandler(applicationReg.ClickTrackingService, appConfig),
		AffiliateLinkAnalyticsHandler: NewAffiliateLinkAnalyticsHandler(applicationReg.AffiliateLinkAnalyticsService),
		PreOrderHandler:               NewPreOrderHandler(applicationReg.PreOrderService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.StateTransferService, applicationReg.FileService),
		MarketingAnalyticsHandler:     NewMarketingAnalyticsHandler(applicationReg.MarketingAnalyticsService),
		SalesStaffAnalyticsHandler:    NewSalesStaffAnalyticsHandler(applicationReg.SalesStaffAnalyticsService),
		ContentStaffAnalyticsHandler:  NewContentStaffAnalyticsHandler(applicationReg.ContentStaffAnalyticsService),
		BrandPartnerAnalyticsHandler:  NewBrandPartnerAnalyticsHandler(applicationReg.BrandPartnerAnalyticsService),
		AdminAnalyticsHandler:         NewAdminAnalyticsHandler(applicationReg.AdminAnalyticsService),
		FacebookSocialHandler:         NewFacebookSocialHandler(appConfig, applicationReg.FacebookSocialService, applicationReg.InfrastructureRegistry.UnitOfWork),
		TikTokSocialHandler:           NewTikTokSocialHandler(appConfig, applicationReg.TikTokSocialService, applicationReg.InfrastructureRegistry.UnitOfWork),
		AIHandler:                     NewAIHandler(applicationReg.AIService),
		JobHandler:                    NewJobHandler(applicationReg.InfrastructureRegistry.CronJobsRegistry),
		RabbitMQHandler:               NewRabbitMQHandler(applicationReg.InfrastructureRegistry.RabbitMQManagementService, applicationReg.InfrastructureRegistry.RabbitMQ),
		TestHandler:                   NewTestHandler(appConfig, applicationReg),
		ScheduleHandler:               NewScheduleHandler(applicationReg.ScheduleService),
		ContentScheduleHandler:        NewContentScheduleHandler(applicationReg.ContentScheduleService),
		ContentEngagementHandler:      NewContentEngagementHandler(applicationReg.ContentEngagementService),
		AlertHandler:                  NewAlertHandler(applicationReg.AlertManagerService),
		SystemHandler:                 NewSystemHandler(applicationReg.SystemService),
		AsynqHandler:                  NewAsynqHandler(applicationReg.InfrastructureRegistry.AsynqClient),
		CacheHandler:                  NewCacheHandler(applicationReg.InfrastructureRegistry.ValkeyCache),
		ProductOptionHandler:          NewProductOptionHandler(applicationReg.ProductOptionService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ViolationHandler:              NewViolationHandler(applicationReg.ViolationService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork),
	}
}
