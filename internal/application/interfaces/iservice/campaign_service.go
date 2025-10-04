package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
)

type CampaignService interface {
	// CreateCampaignFromContract creates a new campaign based on the provided contract details.
	CreateCampaignFromContract(
		ctx context.Context,
		request *requests.CreateCampaignRequest,
		uow irepository.UnitOfWork,
	) (*responses.CampaignResponse, error)
}
