// Package consumer provides the ConsumerRegistry struct that holds various consumer services.
package consumer

import (
	"core-backend/internal/application"

	"go.uber.org/zap"
)

// ConsumerRegistry holds all consumer handlers
type ConsumerRegistry struct {
	ContractCreateConsumer        *ContractCreateConsumer
	ContractCreatePaymentConsumer *ContractCreatePaymentConsumer
	ExcelImportProductsConsumer   *ExcelImportProductsConsumer
	NotificationEmailConsumer     *NotificationEmailConsumer
	NotificationPushConsumer      *NotificationPushConsumer
	VideoUploadConsumer           *VideoUploadConsumer
}

// NewConsumerRegistry creates a new consumer registry with all consumers initialized
func NewConsumerRegistry(appRegistry *application.ApplicationRegistry) *ConsumerRegistry {
	zap.L().Info("Initializing consumer registry")

	registry := &ConsumerRegistry{
		ContractCreateConsumer:        NewContractCreateConsumer(appRegistry),
		ContractCreatePaymentConsumer: NewContractCreatePaymentConsumer(appRegistry),
		ExcelImportProductsConsumer:   NewExcelImportProductsConsumer(appRegistry),
		NotificationEmailConsumer:     NewNotificationEmailConsumer(appRegistry),
		NotificationPushConsumer:      NewNotificationPushConsumer(appRegistry),
		VideoUploadConsumer:           NewVideoUploadConsumer(appRegistry),
	}

	zap.L().Info("Consumer registry initialized successfully")
	return registry
}
