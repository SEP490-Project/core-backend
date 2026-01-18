// Package asynqhandler provides Asynq task handlers for Asynq scheduled tasks.
package asynqhandler

import (
	"core-backend/config"
	"core-backend/internal/application"
	asynqClient "core-backend/internal/infrastructure/asynq"
)

type AsynqHandlerRegistry struct {
	config                              *config.AppConfig
	client                              *asynqClient.AsynqClient
	ContentScheduleHandler              *ContentScheduleHandler
	NotificationScheduledHandler        *NotificationScheduledHandler
	CancelPaymentHandler                *CancelPaymentHandler
	AutoReceiveOrderHandler             *AutoReceiveOrderHandler
	PreOrderOpeningHandler              *PreOrderOpeningHandler
	PreOrderAutoReceiveHandler          *PreOrderAutoReceiveHandler
	LimitedProductAnnouncementHandler   *LimitedProductAnnouncementHandler
}

func NewAsynqHandlerRegistry(
	config *config.AppConfig,
	client *asynqClient.AsynqClient,
	appReg *application.ApplicationRegistry,
) *AsynqHandlerRegistry {
	return &AsynqHandlerRegistry{
		config:                              config,
		client:                              client,
		ContentScheduleHandler:              NewContentScheduleHandler(appReg.ContentScheduleService, appReg.ScheduleService, appReg.AlertManagerService),
		NotificationScheduledHandler:        NewNotificationScheduledHandler(appReg.NotificationService, appReg.InfrastructureRegistry.UnitOfWork),
		CancelPaymentHandler:                NewCancelPaymentHandler(appReg.PaymentTransactionService, appReg.InfrastructureRegistry.UnitOfWork),
		AutoReceiveOrderHandler:             NewAutoReceiveOrderHandler(appReg.OrderService),
		PreOrderOpeningHandler:              NewPreOrderOpeningHandler(appReg.PreOrderService, appReg.StateTransferService, appReg.InfrastructureRegistry.UnitOfWork),
		PreOrderAutoReceiveHandler:          NewPreOrderAutoReceiveHandler(appReg.PreOrderService, appReg.StateTransferService, appReg.InfrastructureRegistry.UnitOfWork),
		LimitedProductAnnouncementHandler:   NewLimitedProductAnnouncementHandler(appReg.NotificationService, appReg.InfrastructureRegistry.UnitOfWork),
	}
}

func (r *AsynqHandlerRegistry) RegisterHandlers() {
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.ContentSchedule, r.ContentScheduleHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.NotificationSchedule, r.NotificationScheduledHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.CancelPaymentSchedule, r.CancelPaymentHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.AutoReceiveOrder, r.AutoReceiveOrderHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.PreOrderOpening, r.PreOrderOpeningHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.PreOrderAutoReceive, r.PreOrderAutoReceiveHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.LimitedProductAnnouncement, r.LimitedProductAnnouncementHandler)
}
