package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"

	"go.uber.org/zap"
)

type CampaignService struct {
	campaignRepo irepository.GenericRepository[model.Campaign]
}

// CreateCampaignFromContract implements iservice.CampaignService.
func (c *CampaignService) CreateCampaignFromContract(
	ctx context.Context,
	request *requests.CreateCampaignRequest,
	uow irepository.UnitOfWork,
) (*responses.CampaignResponse, error) {
	zap.L().Info("Creating campaign from contract", zap.Any("request", request))

	campaignRepo := uow.Campaigns()

	creatingModel, err := request.ToModel()
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}
	if err = campaignRepo.Add(ctx, creatingModel); err != nil {
		zap.L().Error("Failed to add campaign to repository", zap.Error(err))
		return nil, err
	}

	var createdCampaign *model.Campaign
	if createdCampaign, err = campaignRepo.GetByID(ctx, creatingModel.ID, []string{"Contract"}); err != nil {
		zap.L().Error("Failed to retrieve created campaign", zap.Error(err))
		return nil, err
	}

	zap.L().Info("Successfully created campaign", zap.Any("campaign", createdCampaign))
	return (&responses.CampaignResponse{}).ToCampaignResponse(createdCampaign), nil
}

func NewCampaignService(campaignRepo irepository.GenericRepository[model.Campaign]) iservice.CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
	}
}
