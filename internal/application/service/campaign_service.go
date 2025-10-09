package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CampaignService struct {
	campaignRepo irepository.GenericRepository[model.Campaign]
}

// GetCampaignsInfoByUserID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignsInfoByUserID(ctx context.Context, userID uuid.UUID) ([]*responses.CampaignInfoResponse, int64, error) {
	zap.L().Info("Retrieving campaigns info by user ID", zap.String("user_id", userID.String()))

	query := func(db *gorm.DB) *gorm.DB {
		return db.
			InnerJoins("INNER JOIN contracts ON contracts.id = campaigns.contract_id").
			InnerJoins("INNER JOIN brands on brands.id = contracts.brand_id").
			Where("brands.user_id = ?", userID)
	}
	campaigns, totalCount, err := c.campaignRepo.GetAll(ctx, query, []string{"Contract"}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to retrieve campaigns by user ID",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, 0, err
	}

	responses := responses.CampaignInfoResponse{}.ToCampaignInfoResponseList(campaigns)
	return responses, totalCount, nil
}

// GetCampaignDetailsByContractID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignDetailsByContractID(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.CampaignDetailsResponse, error) {
	zap.L().Info("Retrieving campaign details by contract ID", zap.String("contract_id", contractID.String()))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID)
	}
	var campaign *model.Campaign
	var err error
	if campaign, err = c.campaignRepo.GetByCondition(ctx, filterQuery, []string{"Contract", "Milestones.Tasks"}); err != nil {
		zap.L().Error("Failed to retrieve campaign with details by contract ID",
			zap.String("contract_id", contractID.String()),
			zap.Error(err))
		return nil, err
	} else if campaign == nil {
		zap.L().Warn("No campaign found for the given contract ID", zap.String("contract_id", contractID.String()))
		return nil, errors.New("no campaign found for the given contract ID")
	}

	return responses.CampaignDetailsResponse{}.ToCampaignDetailsResponse(campaign), nil
}

// GetCampaignInfoByContractID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignInfoByContractID(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.CampaignInfoResponse, error) {
	zap.L().Info("Retrieving campaign info by contract ID", zap.String("contract_id", contractID.String()))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID)
	}
	var campaign *model.Campaign
	var err error
	if campaign, err = c.campaignRepo.GetByCondition(ctx, filterQuery, []string{"Contract"}); err != nil {
		zap.L().Error("Failed to retrieve campaign with info by contract ID",
			zap.String("contract_id", contractID.String()),
			zap.Error(err))
		return nil, err
	} else if campaign == nil {
		zap.L().Warn("No campaign found for the given contract ID", zap.String("contract_id", contractID.String()))
		return nil, errors.New("no campaign found for the given contract ID")
	}

	return responses.CampaignInfoResponse{}.ToCampaignInfoResponse(campaign), nil
}

// GetCampaignsInfoByBrandID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignsInfoByBrandID(ctx context.Context, brandID uuid.UUID) ([]*responses.CampaignInfoResponse, int64, error) {
	zap.L().Info("Retrieving campaigns info by brand ID", zap.String("brand_id", brandID.String()))

	query := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN contracts ON contracts.id = campaigns.contract_id").
			Where("contracts.brand_id = ?", brandID)
	}
	campaigns, totalCount, err := c.campaignRepo.GetAll(ctx, query, []string{"Contract"}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to retrieve campaigns by brand ID",
			zap.String("brand_id", brandID.String()),
			zap.Error(err))
		return nil, 0, err
	}

	responses := responses.CampaignInfoResponse{}.ToCampaignInfoResponseList(campaigns)
	return responses, totalCount, nil
}

// DeleteCampaign implements iservice.CampaignService.
func (c *CampaignService) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	zap.L().Info("Deleting campaign", zap.String("id", id.String()))

	if exists, err := c.campaignRepo.ExistsByID(ctx, id); err != nil {
		zap.L().Error("Campaign not found", zap.String("id", id.String()), zap.Error(err))
		return err
	} else if !exists {
		zap.L().Warn("Campaign not found", zap.String("id", id.String()))
		return fmt.Errorf("campaign with ID %s not found", id.String())
	}

	if err := c.campaignRepo.DeleteByID(ctx, id); err != nil {
		zap.L().Error("Failed to delete campaign", zap.String("id", id.String()), zap.Error(err))
		return err
	}

	zap.L().Info("Successfully deleted campaign", zap.String("id", id.String()))
	return nil
}

// GetCampaignInfoByID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignInfoByID(ctx context.Context, id uuid.UUID) (*responses.CampaignInfoResponse, error) {
	zap.L().Info("Retrieving campaign info by ID", zap.String("id", id.String()))

	var campaign *model.Campaign
	var err error
	if campaign, err = c.campaignRepo.GetByID(ctx, id, []string{"Contract"}); err != nil {
		zap.L().Error("Failed to retrieve campaign", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	return responses.CampaignInfoResponse{}.ToCampaignInfoResponse(campaign), nil
}

// GetCampaignDetailsByID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignDetailsByID(ctx context.Context, id uuid.UUID) (*responses.CampaignDetailsResponse, error) {
	zap.L().Info("Retrieving campaign details by ID", zap.String("id", id.String()))

	var campaign *model.Campaign
	var err error
	if campaign, err = c.campaignRepo.GetByID(ctx, id, []string{"Contract", "Milestones.Tasks"}); err != nil {
		zap.L().Error("Failed to retrieve campaign with details", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	return responses.CampaignDetailsResponse{}.ToCampaignDetailsResponse(campaign), nil
}

// GetCampaignsByFilter implements iservice.CampaignService.
func (c *CampaignService) GetCampaignsByFilter(
	ctx context.Context,
	filterRequest *requests.CampaignFilterRequest,
) ([]*responses.CampaignInfoResponse, int64, error) {
	zap.L().Info("Retrieving campaigns by filter", zap.Any("filter", filterRequest))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filterRequest.StartDate != nil {
			db = db.Where("start_date >= ?", *filterRequest.StartDate)
		}
		if filterRequest.EndDate != nil {
			db = db.Where("end_date <= ?", *filterRequest.EndDate)
		}
		if filterRequest.Keyword != nil {
			db = db.Where("name ILIKE ?", "%"+*filterRequest.Keyword+"%")
		}
		if filterRequest.Status != nil {
			db = db.Where("status = ?", *filterRequest.Status)
		}
		if filterRequest.Type != nil {
			db = db.Where("type = ?", *filterRequest.Type)
		}
		if filterRequest.Keyword != nil {
			keyword := "%" + *filterRequest.Keyword + "%"
			db = db.Where("name ILIKE ? OR description ILIKE ?", keyword, keyword)
		}

		sortBy := filterRequest.SortBy
		sortOrder := filterRequest.SortOrder
		if sortBy == "" {
			sortBy = "created_at"
		}
		if sortOrder == "" || (sortOrder != "asc" && sortOrder != "desc") {
			sortOrder = "desc"
		}
		db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

		return db
	}
	campaigns, totalCount, err := c.campaignRepo.GetAll(ctx, filterQuery, []string{"Contract"}, filterRequest.Limit, filterRequest.Page)

	return responses.CampaignInfoResponse{}.ToCampaignInfoResponseList(campaigns), totalCount, err
}

// CreateCampaignFromContract implements iservice.CampaignService.
func (c *CampaignService) CreateCampaignFromContract(
	ctx context.Context,
	userID uuid.UUID,
	request *requests.CreateCampaignRequest,
	uow irepository.UnitOfWork,
) (*responses.CampaignDetailsResponse, error) {
	zap.L().Info("Creating campaign from contract", zap.Any("request", request))

	campaignRepo := uow.Campaigns()
	milstoneRepo := uow.Milestones()
	taskRepo := uow.Tasks()

	existFilterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", request.ContractID)
	}
	if exists, err := campaignRepo.Exists(ctx, existFilterQuery); err != nil {
		zap.L().Error("Failed to check if campaign exists for contract", zap.Error(err))
		return nil, err
	} else if exists {
		errorStr := fmt.Sprintf("Campaign already exists for contract %s", request.ContractID)
		zap.L().Warn(errorStr, zap.String("contract_id", request.ContractID))
		return nil, errors.New(errorStr)
	}

	creatingCampaignModel, totalTasksCount, err := request.ToModel(userID)
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}
	creatingMilestoneModels := creatingCampaignModel.Milestones
	creatingCampaignModel.Milestones = nil
	if err = campaignRepo.Add(ctx, creatingCampaignModel); err != nil {
		zap.L().Error("Failed to add campaign to repository", zap.Error(err))
		return nil, err
	}

	var rowsAffected int64
	if len(creatingMilestoneModels) > 0 {
		rowsAffected, err = milstoneRepo.BulkAdd(ctx, creatingMilestoneModels, 0)
		if err != nil {
			zap.L().Error("Failed to bulk add milestones", zap.Error(err))
			return nil, err
		}
		if rowsAffected != int64(len(creatingMilestoneModels)) {
			zap.L().Warn("Not all milestones were added",
				zap.Int64("expected", int64(len(creatingMilestoneModels))),
				zap.Int64("actual", rowsAffected))
		}
	}

	creatingTaskModels := utils.FlatMapMapper(creatingCampaignModel.Milestones, func(m *model.Milestone) []*model.Task { return m.Tasks })
	if totalTasksCount > 0 {
		rowsAffected, err = taskRepo.BulkAdd(ctx, creatingTaskModels, 0)
		if err != nil {
			zap.L().Error("Failed to bulk add tasks", zap.Error(err))
			return nil, err
		}
		if rowsAffected != int64(len(creatingTaskModels)) {
			zap.L().Warn("Not all tasks were added",
				zap.Int64("expected", int64(len(creatingTaskModels))),
				zap.Int64("actual", rowsAffected))
		}
	}

	var createdCampaign *model.Campaign
	if createdCampaign, err = campaignRepo.GetByID(ctx, creatingCampaignModel.ID, []string{"Contract", "Milestones.Tasks"}); err != nil {
		zap.L().Error("Failed to retrieve created campaign", zap.Error(err))
		return nil, err
	}

	response := responses.CampaignDetailsResponse{}.ToCampaignDetailsResponse(createdCampaign)
	zap.L().Info("Successfully created campaign", zap.Any("campaign", response))
	return response, nil
}

func NewCampaignService(campaignRepo irepository.GenericRepository[model.Campaign]) iservice.CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
	}
}
