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
		PayOsHandler:                  NewPayOsHandler(appConfig, applicationReg.PaymentTransactionService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.ProxiesRegistry.PayOSProxy, applicationReg.InfrastructureRegistry.UnitOfWork),
		StateHandler:                  NewStateHandler(applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork, validator.New()),
		ContractHandler:               NewContractHandler(applicationReg.ContractService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.InfrastructureRegistry.RabbitMQ),
		CampaignHandler:               NewCampaignHandler(applicationReg.CampaignService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork),
		CategoryHandler:               NewCategoryHandler(applicationReg.ProductCategoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ModifiedHistoryHandler:        NewModifiedHistoryHandler(applicationReg.ModifiedHistoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		AdminConfigHandler:            NewAdminConfigHandler(applicationReg.AdminConfigService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContractPaymentHandler:        NewContractPaymentHandler(applicationReg.ContractPaymentService, applicationReg.PaymentTransactionService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ConceptHandler:                NewConceptHandler(applicationReg.ConceptService),
		OrderHandler:                  NewOrderHandler(applicationReg.OrderService, applicationReg.InfrastructureRegistry.GHNService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.StateTransferService),
		ChannelHandler:                NewChannelHandler(applicationReg.ChannelService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContentHandler:                NewContentHandler(applicationReg.ContentService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork),
		BlogHandler:                   NewBlogHandler(applicationReg.BlogService, applicationReg.InfrastructureRegistry.UnitOfWork),
		TaskHandler:                   NewTaskHandler(applicationReg.TaskService, applicationReg.InfrastructureRegistry.UnitOfWork),
		DeviceTokenHandler:            NewDeviceTokenHandler(applicationReg.DeviceTokenService),
		NotificationHandler:           NewNotificationHandler(applicationReg.NotificationService),
		LocationHandler:               NewLocationHandler(applicationReg.LocationService, applicationReg.InfrastructureRegistry.LocationSyncTask),
		TagHandler:                    NewTagHandler(applicationReg.TagService, applicationReg.InfrastructureRegistry.UnitOfWork),
		GHNHandler:                    NewGHNHandler(applicationReg.InfrastructureRegistry.GHNService, applicationReg.InfrastructureRegistry.UnitOfWork),
		AffiliateLinkHandler:          NewAffiliateLinkHandler(applicationReg.AffiliateLinkService),
		RedirectHandler:               NewRedirectHandler(applicationReg.ClickTrackingService, appConfig),
		AffiliateLinkAnalyticsHandler: NewAffiliateLinkAnalyticsHandler(applicationReg.AffiliateLinkAnalyticsService),
		PreOrderHandler:               NewPreOrderHandler(applicationReg.PreOrderService, applicationReg.InfrastructureRegistry.UnitOfWork),
		MarketingAnalyticsHandler:     NewMarketingAnalyticsHandler(applicationReg.MarketingAnalyticsService),
	}
}
