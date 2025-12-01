package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"encoding/json"

	"go.uber.org/zap"
)

type CampaignCreateConsumer struct {
	appRegistry *application.ApplicationRegistry
	unitOfWork  irepository.UnitOfWork
}

func NewCampaignCreateConsumer(appRegistry *application.ApplicationRegistry) *CampaignCreateConsumer {
	return &CampaignCreateConsumer{
		appRegistry: appRegistry,
		unitOfWork:  appRegistry.InfrastructureRegistry.UnitOfWork,
	}
}

func (c *CampaignCreateConsumer) Handle(ctx context.Context, body []byte) error {
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Recovered from panic in CampaignCreateConsumer.Handle", zap.Any("panic", r))
		}
	}()

	var msg consumers.CampaignCreateMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal campaign creation message", zap.Error(err))
		return err
	}

	zap.L().Info("Processing campaign creation message", zap.String("name", msg.Data.Name))

	uow := c.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	var err error
	if msg.Data.ContractID != "" {
		_, err = c.appRegistry.CampaignService.CreateCampaignFromContract(ctx, msg.UserID, &msg.Data, uow)
	} else {
		_, err = c.appRegistry.CampaignService.CreateInternalCampaign(ctx, uow, &msg.Data, msg.UserID)
	}

	if err != nil {
		uow.Rollback()
		zap.L().Error("Failed to create campaign from message", zap.Error(err))
		return err
	}

	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit campaign creation", zap.Error(err))
		return err
	}

	zap.L().Info("Campaign created successfully from message")
	return nil
}
