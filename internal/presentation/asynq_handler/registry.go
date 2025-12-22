// Package asynqhandler provides Asynq task handlers for Asynq scheduled tasks.
package asynqhandler

import (
	"core-backend/config"
	"core-backend/internal/application"
	asynqClient "core-backend/internal/infrastructure/asynq"
)

type AsynqHandlerRegistry struct {
	config                       *config.AppConfig
	client                       *asynqClient.AsynqClient
	ContentScheduleHandler       *ContentScheduleHandler
	NotificationScheduledHandler *NotificationScheduledHandler
	CancelPaymentHandler         *CancelPaymentHandler
	AutoReceiveOrderHandler      *AutoReceiveOrderHandler
}

func NewAsynqHandlerRegistry(
	config *config.AppConfig,
	client *asynqClient.AsynqClient,
	appReg *application.ApplicationRegistry,
) *AsynqHandlerRegistry {
	return &AsynqHandlerRegistry{
		config:                       config,
		client:                       client,
		ContentScheduleHandler:       NewContentScheduleHandler(appReg.ContentScheduleService, appReg.AlertManagerService),
		NotificationScheduledHandler: NewNotificationScheduledHandler(appReg.NotificationService, appReg.InfrastructureRegistry.UnitOfWork),
		CancelPaymentHandler:         NewCancelPaymentHandler(appReg.PaymentTransactionService, appReg.InfrastructureRegistry.UnitOfWork),
		AutoReceiveOrderHandler:      NewAutoReceiveOrderHandler(appReg.OrderService),
	}
}

func (r *AsynqHandlerRegistry) RegisterHandlers() {
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.ContentSchedule, r.ContentScheduleHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.NotificationSchedule, r.NotificationScheduledHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.CancelPaymentSchedule, r.CancelPaymentHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.AutoReceiveOrder, r.AutoReceiveOrderHandler)
}
