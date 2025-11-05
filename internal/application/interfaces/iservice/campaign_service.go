package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type CampaignService interface {
	// CreateCampaignFromContract creates a new campaign based on the provided contract details.
	CreateCampaignFromContract(
		ctx context.Context,
		userID uuid.UUID,
		request *requests.CreateCampaignRequest,
		uow irepository.UnitOfWork,
	) (*responses.CampaignDetailsResponse, error)

	// CreateInternalCampaign creates a new campaign internally without linking to a contract.
	CreateInternalCampaign(
		ctx context.Context,
		uow irepository.UnitOfWork,
		request *requests.CreateCampaignRequest,
		createdBy uuid.UUID,
	) (*responses.CampaignDetailsResponse, error)

	// GetCampaignByID returns the campaign with the provided ID.
	GetCampaignInfoByID(ctx context.Context, id uuid.UUID) (*responses.CampaignInfoResponse, error)

	// GetCampaignDetailsByID returns detailed information about the campaign with the provided ID.
	// Details includes milestones and tasks info.
	GetCampaignDetailsByID(ctx context.Context, id uuid.UUID) (*responses.CampaignDetailsResponse, error)

	// GetCampaignInfoByContractID returns the campaign info associated with the given contract ID.
	GetCampaignInfoByContractID(ctx context.Context, contractID uuid.UUID) (*responses.CampaignInfoResponse, error)

	// GetCampaignDetailsByContractID returns detailed information about the campaign associated with the given contract ID.
	GetCampaignDetailsByContractID(ctx context.Context, contractID uuid.UUID) (*responses.CampaignDetailsResponse, error)

	// GetCampaignsInfoByBrandID returns a list of campaigns associated with the given brand ID.
	GetCampaignsInfoByBrandID(ctx context.Context, brandID uuid.UUID) ([]*responses.CampaignInfoResponse, int64, error)

	// GetCampaignsInfoByUserID returns a list of campaigns associated with the given user ID (of role BRAND_PARTNER)
	GetCampaignsInfoByUserID(ctx context.Context, userID uuid.UUID) ([]*responses.CampaignInfoResponse, int64, error)

	// GetCampaignsByFilter returns a list of campaigns based on the provided filter criteria.
	GetCampaignsByFilter(ctx context.Context, filter *requests.CampaignFilterRequest) ([]*responses.CampaignInfoResponse, int64, error)

	// DeleteCampaign deletes the campaign with the provided ID.
	DeleteCampaign(ctx context.Context, id uuid.UUID) error

	// SuggestCampaignFromContract generates campaign suggestions based on contract deliverables.
	SuggestCampaignFromContract(ctx context.Context, contractID uuid.UUID) (*responses.CampaignSuggestionResponse, error)
}
