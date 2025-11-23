// Package consumer provides the ConsumerRegistry struct that holds various consumer services.
package consumer

import (
	"core-backend/internal/application"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"

	"go.uber.org/zap"
)

// ConsumerRegistry holds all consumer handlers
type ConsumerRegistry struct {
	ContractCreateConsumer        *ContractCreateConsumer
	ContractCreatePaymentConsumer *ContractCreatePaymentConsumer
	ExcelImportProductsConsumer   *ExcelImportProductsConsumer
	NotificationEmailConsumer     *NotificationEmailConsumer
	NotificationPushConsumer      *NotificationPushConsumer
	NotificationInAppConsumer     *NotificationInAppConsumer
	VideoUploadConsumer           *VideoUploadConsumer
	ClickEventConsumer            *ClickEventConsumer
	ContentPublishConsumer        *ContentPublishConsumer
	ContentPublishAllConsumer     *ContentPublishAllConsumer
}

// NewConsumerRegistry creates a new consumer registry with all consumers initialized
func NewConsumerRegistry(
	appRegistry *application.ApplicationRegistry,
	infraRegistry *infrastructure.InfrastructureRegistry,
	dbRegistry *gormrepository.DatabaseRegistry,
) *ConsumerRegistry {
	zap.L().Info("Initializing consumer registry")

	registry := &ConsumerRegistry{
		ContractCreateConsumer:        NewContractCreateConsumer(appRegistry),
		ContractCreatePaymentConsumer: NewContractCreatePaymentConsumer(appRegistry),
		ExcelImportProductsConsumer:   NewExcelImportProductsConsumer(appRegistry),
		NotificationEmailConsumer:     NewNotificationEmailConsumer(infraRegistry, dbRegistry, appRegistry.UserService),
		NotificationPushConsumer:      NewNotificationPushConsumer(infraRegistry, dbRegistry, appRegistry.UserService),
		NotificationInAppConsumer:     NewNotificationInAppConsumer(infraRegistry, dbRegistry, appRegistry.UserService),
		VideoUploadConsumer:           NewVideoUploadConsumer(appRegistry),
		ClickEventConsumer:            NewClickEventConsumer(dbRegistry.ClickEventRepository),
		ContentPublishConsumer:        NewContentPublishConsumer(appRegistry.ContentPublishingService),
		ContentPublishAllConsumer:     NewContentPublishAllConsumer(appRegistry.ContentPublishingService),
	}

	zap.L().Info("Consumer registry initialized successfully")
	return registry
}
