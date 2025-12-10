package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"

	"go.uber.org/zap"
)

type webhookDataService struct {
	webhookRepo irepository.GenericRepository[model.WebhookData]
	unitOfWork  irepository.UnitOfWork
}

func NewWebhookDataService(webhookRepo irepository.GenericRepository[model.WebhookData], unitOfWork irepository.UnitOfWork) iservice.WebhookDataService {
	return &webhookDataService{
		webhookRepo: webhookRepo,
		unitOfWork:  unitOfWork,
	}
}

// SaveWebhookData implements iservice.WebhookDataService.
func (s *webhookDataService) SaveWebhookData(ctx context.Context, async bool, webhookData *model.WebhookData) error {
	zap.L().Debug("WebhookDataService - SaveWebhookData called",
		zap.Bool("is_async", async),
		zap.Any("webhook_data", webhookData))

	if async {
		go s.processAndSaveWebhookData(context.Background(), webhookData)
	} else {
		return s.processAndSaveWebhookData(ctx, webhookData)
	}

	return nil
}

func (s *webhookDataService) processAndSaveWebhookData(ctx context.Context, webhookData *model.WebhookData) error {
	return helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err := uow.WebhookData().Add(ctx, webhookData); err != nil {
			zap.L().Error("Failed to save webhook data", zap.Error(err))
			return err
		}

		return nil
	})
}
