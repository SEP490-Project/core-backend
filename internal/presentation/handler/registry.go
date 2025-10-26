// Package handler implements HTTP handlers for the application.
package handler

import (
	"core-backend/internal/application"

	"github.com/go-playground/validator/v10"
)

type HandlerRegistry struct {
	ApplicationRegistry    *application.ApplicationRegistry
	AuthHandler            *AuthHandler
	UserHandler            *UserHandler
	HealthHandler          *HealthHandler
	ProductHandler         *ProductHandler
	BrandHandler           *BrandHandler
	FileHandler            *S3Handler
	PayOsHandler           *PayOsHandler
	StateHandler           *StateHandler
	ContractHandler        *ContractHandler
	CampaignHandler        *CampaignHandler
	CategoryHandler        *ProductCategoryHandler
	ModifiedHistoryHandler *ModifiedHistoryHandler
	AdminConfigHandler     *AdminConfigHandler
	ContractPaymentHandler *ContractPaymentHandler
	ConceptHandler         *ConceptHandler
	OrderHandler           *OrderHandler
	ChannelHandler         *ChannelHandler
	ContentHandler         *ContentHandler
	BlogHandler            *BlogHandler
	TaskHandler            *TaskHandler
	DeviceTokenHandler     *DeviceTokenHandler
	NotificationHandler    *NotificationHandler
}

func NewHandlerRegistry(applicationReg *application.ApplicationRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ApplicationRegistry:    applicationReg,
		AuthHandler:            NewAuthHandler(applicationReg.AuthService),
		UserHandler:            NewUserHandler(applicationReg.UserService, applicationReg.InfrastructureRegistry.UnitOfWork),
		HealthHandler:          NewHealthHandler(applicationReg.InfrastructureRegistry),
		ProductHandler:         NewProductHandler(applicationReg.ProductService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork),
		BrandHandler:           NewBrandHandler(applicationReg.BrandService, applicationReg.InfrastructureRegistry.UnitOfWork),
		FileHandler:            NewS3Handler(applicationReg.FileService),
		PayOsHandler:           NewPayOsHandler(applicationReg.InfrastructureRegistry.PayOsService),
		StateHandler:           NewStateHandler(applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork, validator.New()),
		ContractHandler:        NewContractHandler(applicationReg.ContractService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork, applicationReg.InfrastructureRegistry.RabbitMQ),
		CampaignHandler:        NewCampaignHandler(applicationReg.CampaignService, applicationReg.InfrastructureRegistry.UnitOfWork),
		CategoryHandler:        NewCategoryHandler(applicationReg.ProductCategoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ModifiedHistoryHandler: NewModifiedHistoryHandler(applicationReg.ModifiedHistoryService, applicationReg.InfrastructureRegistry.UnitOfWork),
		AdminConfigHandler:     NewAdminConfigHandler(applicationReg.AdminConfigService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContractPaymentHandler: NewContractPaymentHandler(applicationReg.ContractPaymentService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ConceptHandler:         NewConceptHandler(applicationReg.ConceptService),
		OrderHandler:           NewOrderHandler(applicationReg.OrderService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ChannelHandler:         NewChannelHandler(applicationReg.ChannelService, applicationReg.InfrastructureRegistry.UnitOfWork),
		ContentHandler:         NewContentHandler(applicationReg.ContentService, applicationReg.StateTransferService, applicationReg.InfrastructureRegistry.UnitOfWork),
		BlogHandler:            NewBlogHandler(applicationReg.BlogService),
		TaskHandler:            NewTaskHandler(applicationReg.TaskService, applicationReg.InfrastructureRegistry.UnitOfWork),
		DeviceTokenHandler:     NewDeviceTokenHandler(applicationReg.DeviceTokenService),
		NotificationHandler:    NewNotificationHandler(applicationReg.NotificationService),
	}
}
