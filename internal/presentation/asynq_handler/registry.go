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
	}
}

func (r *AsynqHandlerRegistry) RegisterHandlers() {
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.ContentSchedule, r.ContentScheduleHandler)
	r.client.RegisterHandler(r.config.Asynq.TaskTypes.NotificationSchedule, r.NotificationScheduledHandler)
}
