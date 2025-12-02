package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CampaignService struct {
	campaignRepo       irepository.GenericRepository[model.Campaign]
	contractRepo       irepository.GenericRepository[model.Contract]
	orderRepo          irepository.OrderRepository
	preOrderRepo       irepository.PreOrderRepository
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	affiliateLinkRepo  irepository.AffiliateLinkRepository
	kpiMetricsRepo     irepository.GenericRepository[model.KPIMetrics]
}

// SetRejectReason implements iservice.CampaignService.
func (c *CampaignService) SetRejectReason(ctx context.Context, uow irepository.UnitOfWork, campaignID uuid.UUID, reason string, updatedBy uuid.UUID) error {
	zap.L().Info("CampaignService - SetRejectReason called",
		zap.String("campaign_id", campaignID.String()),
		zap.String("reason", reason),
		zap.String("updated_by", updatedBy.String()))

	campaignRepo := uow.Campaigns()
	campaign, err := campaignRepo.GetByID(ctx, campaignID, nil)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("Campaign not found", zap.String("campaign_id", campaignID.String()))
			return fmt.Errorf("campaign with ID %s not found", campaignID.String())
		}
		zap.L().Error("Failed to retrieve campaign", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return err
	} else if campaign == nil {
		zap.L().Warn("Campaign not found", zap.String("campaign_id", campaignID.String()))
		return fmt.Errorf("campaign with ID %s not found", campaignID.String())
	}

	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", campaignID)
	}
	if err := campaignRepo.UpdateByCondition(ctx, filterQuery, map[string]any{
		"reject_reason": &reason,
		"updated_by":    &updatedBy,
	}); err != nil {
		zap.L().Error("Failed to update campaign", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return err
	}

	zap.L().Info("Successfully set reject reason for campaign",
		zap.String("campaign_id", campaignID.String()))
	return nil
}

// UpdateCampaign implements iservice.CampaignService.
func (c *CampaignService) UpdateCampaign(ctx context.Context, uow irepository.UnitOfWork, campaignID uuid.UUID, request *requests.UpdateCampaignRequest) (*responses.CampaignDetailsResponse, error) {
	zap.L().Info("Updating campaign",
		zap.String("campaign_id", campaignID.String()),
		zap.Any("request", request))

	// 1. Load existing campaign
	campaignRepo := uow.Campaigns()
	existing, err := campaignRepo.GetByID(ctx, campaignID, nil)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("Campaign not found", zap.String("campaign_id", campaignID.String()))
			return nil, errors.New("campaign not found")
		}
		zap.L().Error("Failed to retrieve campaign", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return nil, err
	} else if existing == nil {
		zap.L().Warn("Campaign not found", zap.String("campaign_id", campaignID.String()))
		return nil, errors.New("campaign not found")
	}

	if existing.Status != enum.CampaignDraft {
		zap.L().Warn("CampaignService - UpdateCampaign: Only DRAFT campaigns can be updated",
			zap.String("campaign_id", campaignID.String()))
		return nil, errors.New("only DRAFT campaigns can be updated")
	}

	// 2. Apply updates
	updatingCampaign, err := request.ToExistingModel(existing)
	if err != nil {
		zap.L().Error("Failed to apply updates", zap.Error(err))
		return nil, err
	}

	// 3. Persist changes
	if err := campaignRepo.Update(ctx, updatingCampaign); err != nil {
		zap.L().Error("Failed to update campaign", zap.Error(err))
		return nil, err
	}

	zap.L().Info("Successfully updated campaign")
	return c.GetCampaignDetailsByID(ctx, campaignID)
}

// GetCampaignsInfoByUserID implements iservice.CampaignService.
func (c *CampaignService) GetCampaignsInfoByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filterRequest *requests.CampaignFilterRequest,
) ([]*responses.CampaignInfoResponse, int64, error) {
	zap.L().Info("Retrieving campaigns info by user ID", zap.String("user_id", userID.String()))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filterRequest.StartDate != nil {
			db = db.Where("start_date >= ?", *filterRequest.StartDate)
		}
		if filterRequest.EndDate != nil {
			db = db.Where("end_date <= ?", *filterRequest.EndDate)
		}
		if filterRequest.Status != nil {
			db = db.Where("campaigns.status = ?", *filterRequest.Status)
		}
		if filterRequest.Type != nil {
			db = db.Where("campaigns.type = ?", *filterRequest.Type)
		}
		if filterRequest.Keyword != nil {
			keyword := "%" + *filterRequest.Keyword + "%"
			db = db.Where("campaigns.name ILIKE ? OR campaigns.description ILIKE ?", keyword, keyword)
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

	query := func(db *gorm.DB) *gorm.DB {
		return filterQuery(db).
			Joins("INNER JOIN contracts ON contracts.id = campaigns.contract_id").
			Joins("INNER JOIN brands ON brands.id = contracts.brand_id").
			Where("brands.user_id = ?", userID)
	}

	campaigns, totalCount, err := c.campaignRepo.GetAll(ctx, query, []string{"Contract"}, filterRequest.Limit, filterRequest.Page)
	if err != nil {
		zap.L().Error("Failed to retrieve campaigns by user ID",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, 0, err
	}

	return responses.CampaignInfoResponse{}.ToCampaignInfoResponseList(campaigns), totalCount, nil
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
		zap.L().Error("Failed to check if campaign exists", zap.String("id", id.String()), zap.Error(err))
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

	response := responses.CampaignDetailsResponse{}.ToCampaignDetailsResponse(campaign)

	// Calculate metrics
	if campaign.Contract != nil {
		metrics, err := c.calculateMetricsComparison(ctx, campaign.Contract)
		if err != nil {
			zap.L().Error("Failed to calculate metrics comparison", zap.Error(err))
			// Don't fail the request
		} else {
			response.MetricsComparison = metrics
		}
	}

	return response, nil
}

func (c *CampaignService) calculateMetricsComparison(ctx context.Context, contract *model.Contract) (*responses.CampaignMetricsComparison, error) {
	var sow dtos.ScopeOfWork
	if err := json.Unmarshal(contract.ScopeOfWork, &sow); err != nil {
		return nil, err
	}

	comparison := &responses.CampaignMetricsComparison{
		ExpectedMetrics:  make(map[string]float64),
		RealisticMetrics: make(map[string]float64),
		Items:            make([]responses.CampaignItemComparison, 0),
	}

	// Helper to process items
	processItem := func(id *int8, name string, metrics []dtos.KPIGoal, productIDs []uuid.UUID, contentIDs []uuid.UUID, trackingLink string) {
		if id == nil {
			return
		}

		itemComp := responses.CampaignItemComparison{
			ItemID:           *id,
			ItemName:         name,
			ExpectedMetrics:  make([]any, len(metrics)),
			RealisticMetrics: make(map[string]float64),
		}

		// Expected Metrics
		for i, m := range metrics {
			itemComp.ExpectedMetrics[i] = m
			val, _ := strconv.ParseFloat(m.Target, 64)
			comparison.ExpectedMetrics[m.Metric] += val
		}

		// Realistic Metrics
		// 1. Products
		if len(productIDs) > 0 {
			// Query Orders
			orders, _, _ := c.orderRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Joins("JOIN order_items ON order_items.order_id = orders.id").
					Where("order_items.product_id IN ?", productIDs).
					Where("orders.status = ?", enum.OrderStatusReceived).
					Where("orders.order_type = ?", string(enum.ProductTypeLimited))
			}, nil, 10000, 1)

			var revenue float64
			for _, o := range orders {
				revenue += o.TotalAmount
			}
			itemComp.RealisticMetrics["revenue"] += revenue
			itemComp.RealisticMetrics["units_sold"] += float64(len(orders))

			// Query PreOrders
			preOrders, _, _ := c.preOrderRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Joins("JOIN product_variants ON product_variants.id = pre_orders.variant_id").
					Where("product_variants.product_id IN ?", productIDs).
					Where("pre_orders.status = ?", enum.PreOrderStatusPaid)
			}, nil, 10000, 1)

			var preOrderRevenue float64
			for _, po := range preOrders {
				preOrderRevenue += po.TotalAmount
			}
			itemComp.RealisticMetrics["revenue"] += preOrderRevenue
		}

		// 2. Content
		if len(contentIDs) > 0 {
			contentChannels, _, _ := c.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("content_id IN ?", contentIDs)
			}, nil, 1000, 1)

			for _, cc := range contentChannels {
				var metrics model.ContentChannelMetrics
				if err := json.Unmarshal(cc.Metrics, &metrics); err == nil {
					for k, v := range metrics.Current {
						itemComp.RealisticMetrics[k] += v
					}
				}
			}
		}

		// 3. Affiliate
		if trackingLink != "" {
			// Find AffiliateLink by tracking URL
			affiliateLinks, _, _ := c.affiliateLinkRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("tracking_url = ?", trackingLink)
			}, nil, 1, 1)

			if len(affiliateLinks) > 0 {
				linkID := affiliateLinks[0].ID
				// Query KPI Metrics
				kpis, _, _ := c.kpiMetricsRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
					return db.Where("reference_id = ?", linkID).
						Where("reference_type = ?", enum.KPIReferenceTypeAffiliateLink)
				}, nil, 100, 1)

				for _, kpi := range kpis {
					itemComp.RealisticMetrics[string(kpi.Type)] += kpi.Value
				}
			}
		}

		// Aggregate item realistic metrics to total
		for k, v := range itemComp.RealisticMetrics {
			comparison.RealisticMetrics[k] += v
		}

		comparison.Items = append(comparison.Items, itemComp)
	}

	// Iterate Deliverables
	if sow.Deliverables.AdvertisedItems != nil {
		for _, item := range sow.Deliverables.AdvertisedItems {
			processItem(item.ID, item.Name, item.KPIs, nil, item.ContentIDs, sow.Deliverables.TrackingLink)
		}
	}

	if sow.Deliverables.Events != nil {
		for _, item := range sow.Deliverables.Events {
			processItem(item.ID, item.Name, item.KPIs, nil, nil, "")
		}
	}

	if sow.Deliverables.Products != nil {
		for _, item := range sow.Deliverables.Products {
			processItem(item.ID, item.Name, item.KPIs, item.ProductIDs, nil, "")
		}
	}

	if sow.Deliverables.Concepts != nil {
		for _, item := range sow.Deliverables.Concepts {
			processItem(item.ID, item.Name, item.KPIs, item.ProductIDs, item.ContentIDs, "")
		}
	}

	return comparison, nil
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

	contractID, _ := uuid.Parse(request.ContractID)
	var contract *model.Contract
	existCampaignFunc := func(ctx context.Context) error {
		existFilterQuery := func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ?", contractID)
		}
		if exists, err := c.campaignRepo.Exists(ctx, existFilterQuery); err != nil {
			zap.L().Error("Failed to check if campaign exists for contract", zap.Error(err))
			return err
		} else if exists {
			errorStr := fmt.Sprintf("Campaign already exists for contract %s", request.ContractID)
			zap.L().Warn(errorStr, zap.String("contract_id", request.ContractID))
			return errors.New(errorStr)
		}
		return nil
	}
	contractStatusFunc := func(ctx context.Context) error {
		var err error
		contract, err = c.contractRepo.GetByID(ctx, contractID, nil)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				zap.L().Warn("Contract not found", zap.String("contract_id", request.ContractID))
				return errors.New("contract not found")
			}
			zap.L().Error("Failed to retrieve contract", zap.String("contract_id", request.ContractID), zap.Error(err))
			return err
		} else if contract == nil {
			zap.L().Warn("Contract not found", zap.String("contract_id", request.ContractID))
			return errors.New("contract not found")
		}
		if contract.Status != enum.ContractStatusActive {
			zap.L().Warn("Contract is not active", zap.String("contract_id", request.ContractID), zap.String("status", string(contract.Status)))
			return errors.New("contract is not active")
		}
		return nil
	}
	if err := utils.RunParallel(ctx, 2, existCampaignFunc, contractStatusFunc); err != nil {
		zap.L().Error("Pre-creation checks failed", zap.Error(err))
		return nil, err
	}

	creatingCampaignModel, totalTasksCount, err := request.ToModel(userID)
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}
	creatingMilestoneModels := creatingCampaignModel.Milestones
	creatingCampaignModel.Milestones = nil
	creatingCampaignModel.Status = enum.CampaignDraft

	// Map tasks to SOW items (Best Effort)
	if contract != nil {
		if err = helper.MapTasksToScopeOfWork(contract, creatingMilestoneModels); err != nil {
			zap.L().Warn("Failed to map tasks to SOW", zap.Error(err))
		} else {
			// Update contract SOW in DB
			if err = c.contractRepo.Update(ctx, contract); err != nil {
				zap.L().Error("Failed to update contract SOW", zap.Error(err))
			}
		}
	}

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

	creatingTaskModels := utils.FlatMapMapper(creatingMilestoneModels, func(m *model.Milestone) []*model.Task { return m.Tasks })
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

func (c *CampaignService) CreateInternalCampaign(
	ctx context.Context,
	uow irepository.UnitOfWork,
	request *requests.CreateCampaignRequest,
	createdBy uuid.UUID,
) (*responses.CampaignDetailsResponse, error) {
	zap.L().Info("Creating internal campaigns without contract", zap.Any("request", request))

	campaignRepo := uow.Campaigns()
	milstoneRepo := uow.Milestones()
	taskRepo := uow.Tasks()

	creatingCampaignModel, totalTasksCount, err := request.ToModel(createdBy)
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}
	creatingMilestoneModels := creatingCampaignModel.Milestones
	creatingCampaignModel.Milestones = nil
	// Set campaign status to RUNNING for internal campaigns
	creatingCampaignModel.ContractID = uuid.Nil
	creatingCampaignModel.Status = enum.CampaignRunning
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

// region: ======= Suggest Campaign from Contract  =======

// SuggestCampaignFromContract implements iservice.CampaignService.
func (c *CampaignService) SuggestCampaignFromContract(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.CampaignSuggestionResponse, error) {
	zap.L().Info("Suggesting campaign from contract", zap.String("contract_id", contractID.String()))

	// Retrieve contract with necessary fields
	contract, err := c.contractRepo.GetByID(ctx, contractID, []string{"Brand", "ContractPayments"})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return nil, errors.New("contract not found")
		}
		zap.L().Error("Failed to retrieve contract", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, errors.New("contract not found")
	} else if contract == nil {
		zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
		return nil, errors.New("contract not found")
	}

	// Validate contract status - only ACTIVE contracts can be used for suggestions
	if contract.Status != enum.ContractStatusActive {
		zap.L().Warn("Contract is not active", zap.String("contract_id", contractID.String()), zap.String("status", string(contract.Status)))
		return nil, errors.New("only ACTIVE contracts can be used for campaign suggestions")
	}

	// Validate scope of work exists
	if len(contract.ScopeOfWork) == 0 {
		zap.L().Warn("Contract has no scope of work", zap.String("contract_id", contractID.String()))
		return nil, errors.New("contract has no deliverables defined in scope of work")
	}

	// Parse scope of work
	// var scopeOfWork map[string]any
	// if err = json.Unmarshal(contract.ScopeOfWork, &scopeOfWork); err != nil {
	// 	zap.L().Error("Failed to parse scope of work", zap.String("contract_id", contractID.String()), zap.Error(err))
	// 	return nil, errors.New("invalid scope of work format")
	// }
	var scopeOfWorks dtos.ScopeOfWork
	if err = json.Unmarshal(contract.ScopeOfWork, &scopeOfWorks); err != nil {
		zap.L().Error("Failed to parse scope of work", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, errors.New("invalid scope of work format")
	}

	// Extract deliverables based on contract type
	var suggestedCampaign *responses.SuggestedCampaign
	switch contract.Type {
	case "ADVERTISING":
		suggestedCampaign, err = c.extractAdvertisingTasks(ctx, scopeOfWorks, contract)
	case "AFFILIATE":
		suggestedCampaign, err = c.extractAffiliateTasks(ctx, scopeOfWorks, contract)
	case "BRAND_AMBASSADOR":
		suggestedCampaign, err = c.extractBrandAmbassadorTasks(ctx, scopeOfWorks, contract)
	case "CO_PRODUCING":
		suggestedCampaign, err = c.extractCoProducingStructure(ctx, scopeOfWorks, contract)
	default:
		zap.L().Error("Unsupported contract type", zap.String("contract_id", contractID.String()), zap.String("type", string(contract.Type)))
		return nil, errors.New("unsupported contract type")
	}

	if err != nil {
		zap.L().Error("Failed to extract campaign structure", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, err
	}

	// Validate milestone-payment alignment if contract payments exist
	if len(contract.ContractPayments) > 0 {
		// Filter out deposit payment (first payment is always deposit)
		regularPayments := contract.ContractPayments
		if len(regularPayments) > 0 && regularPayments[0].DueDate.Equal(contract.StartDate) {
			regularPayments = regularPayments[1:] // Skip deposit payment
		}

		// Convert to pointer slice for validation function
		paymentPointers := make([]*model.ContractPayment, len(regularPayments))
		for i := range regularPayments {
			paymentPointers[i] = &regularPayments[i]
		}

		if err := helper.ValidateMilestonePaymentAlignment(suggestedCampaign.Milestones, paymentPointers); err != nil {
			zap.L().Warn("Milestone-payment alignment validation failed",
				zap.String("contract_id", contractID.String()),
				zap.Error(err))
			// Log warning but don't fail the suggestion - payments might not be created yet
		} else {
			zap.L().Info("Milestone-payment alignment validated successfully",
				zap.String("contract_id", contractID.String()),
				zap.Int("milestone_count", len(suggestedCampaign.Milestones)),
				zap.Int("payment_count", len(regularPayments)))
		}
	}

	response := &responses.CampaignSuggestionResponse{
		ContractID:        contractID,
		ContractType:      string(contract.Type),
		SuggestedCampaign: suggestedCampaign,
	}

	zap.L().Info("Successfully suggested campaign from contract",
		zap.String("contract_id", contractID.String()),
		zap.String("contract_type", string(contract.Type)),
		zap.Int("milestones_count", len(suggestedCampaign.Milestones)))

	return response, nil
}

// extractAdvertisingTasks extracts tasks from ADVERTISING contract deliverables
func (c *CampaignService) extractAdvertisingTasks(
	ctx context.Context,
	scopeOfWork dtos.ScopeOfWork,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	// Validate contract first
	if err := helper.ValidateContractForSuggestion(contract); err != nil {
		zap.L().Error("Contract validation failed", zap.Error(err))
		return nil, fmt.Errorf("invalid contract: %w", err)
	}

	// Parse deliverables and financial terms
	deliverables, err := scopeOfWork.Deliverables.ToAdvertisingDeliverable()
	if err != nil {
		zap.L().Error("Failed to convert deliverables to AdvertisingDeliverable", zap.Error(err))
		return nil, fmt.Errorf("failed to parse advertising deliverables: %w", err)
	}

	var financialTerms dtos.AdvertisingFinancialTerms
	if err = json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
		zap.L().Error("Failed to unmarshal financial terms to AdvertisingFinancialTerms", zap.Error(err))
		return nil, fmt.Errorf("failed to parse financial terms: %w", err)
	}

	// Parallel extraction of tasks and milestones
	var suggestedTasks []responses.SuggestedTask
	var suggestedMilestones []responses.SuggestedMilestone
	var tasksErr, milestonesErr error

	err = utils.RunParallel(ctx, 2,
		// Extract tasks
		func(ctx context.Context) error {
			suggestedTasks, tasksErr = c.extractAdvertisingTasksAsync(ctx, deliverables, contract.StartDate)
			return tasksErr
		},
		// Extract milestones
		func(ctx context.Context) error {
			suggestedMilestones, milestonesErr = c.extractAdvertisingMilestonesAsync(ctx, contract, financialTerms)
			return milestonesErr
		},
	)

	if err != nil {
		zap.L().Error("Failed to extract advertising tasks and milestones in parallel", zap.Error(err))
		return nil, fmt.Errorf("parallel extraction failed: %w", err)
	}

	// Assign tasks to milestones using even distribution
	assignedMilestones := helper.DistributeTasksEvenly(suggestedTasks, suggestedMilestones)

	// Generate campaign name and description
	campaignName := "Advertising Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	return &responses.SuggestedCampaign{
		Name:        campaignName,
		Description: utils.PtrOrNil(fmt.Sprintf("Campaign for contract %s with %d advertised items", *contract.ContractNumber, len(deliverables.AdvertisedItems))),
		StartDate:   contract.StartDate,
		EndDate:     contract.EndDate,
		Type:        contract.Type,
		ContractID:  contract.ID,
		Milestones:  assignedMilestones,
	}, nil
}

// extractAdvertisingTasksAsync extracts tasks from advertised items with item-level parallelization
func (c *CampaignService) extractAdvertisingTasksAsync(
	_ context.Context,
	deliverables *dtos.AdvertisingDeliverable,
	contractStartDate time.Time,
) ([]responses.SuggestedTask, error) {
	items := deliverables.AdvertisedItems
	if len(items) == 0 {
		return []responses.SuggestedTask{}, nil
	}

	tasks := make([]responses.SuggestedTask, len(items))
	var wg sync.WaitGroup
	errChan := make(chan error, len(items))

	for i, item := range items {
		wg.Add(1)
		go func(idx int, advertisedItem dtos.AdvertisedItem) {
			defer wg.Done()

			// Use contract start date as initial deadline (will be refined by milestone assignment)
			task := helper.TransformAdvertisedItemToTask(advertisedItem, contractStartDate)
			tasks[idx] = task
		}(i, item)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	zap.L().Info("Extracted advertising tasks",
		zap.Int("task_count", len(tasks)),
		zap.String("contract_type", "ADVERTISING"))

	return tasks, nil
}

// extractAdvertisingMilestonesAsync extracts milestones from financial terms schedules
func (c *CampaignService) extractAdvertisingMilestonesAsync(
	_ context.Context,
	contract *model.Contract,
	financialTerms dtos.AdvertisingFinancialTerms,
) ([]responses.SuggestedMilestone, error) {
	// Generate milestone due dates using shared payment cycle calculator
	dueDates, err := helper.GenerateMilestoneDueDatesFromFinancialTerms(
		contract,
		financialTerms,
		5, // minimumDayBeforeDueDate - should be fetched from config in production
	)
	if err != nil {
		zap.L().Error("Failed to generate milestone due dates", zap.Error(err))
		return nil, fmt.Errorf("failed to generate milestone dates: %w", err)
	}

	if len(dueDates) == 0 {
		return nil, errors.New("no milestone dates generated from financial terms")
	}

	// Calculate base payment per period
	totalCost, err := helper.ExtractTotalCostFromFinancialTerms(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to extract total cost: %w", err)
	}

	depositPercent := float64(0)
	if contract.DepositPercent != nil {
		depositPercent = float64(*contract.DepositPercent)
	}

	basePayment := helper.CalculateBasePaymentPerPeriod(totalCost, depositPercent, len(dueDates))

	// Create milestones
	milestones := make([]responses.SuggestedMilestone, len(dueDates))
	for i, dueDate := range dueDates {
		milestones[i] = responses.SuggestedMilestone{
			Description: fmt.Sprintf("Phase %d: Payment (Due: %s) - Amount: %.0f VND",
				i+1,
				dueDate.Format(utils.DateFormat),
				basePayment),
			DueDate: dueDate,
			Tasks:   []responses.SuggestedTask{}, // Will be assigned later
		}
	}

	zap.L().Info("Extracted advertising milestones",
		zap.Int("milestone_count", len(milestones)),
		zap.Float64("base_payment_per_milestone", basePayment))

	return milestones, nil
}

// extractAffiliateTasks extracts tasks from AFFILIATE contract deliverables
func (c *CampaignService) extractAffiliateTasks(
	_ context.Context,
	scopeOfWork dtos.ScopeOfWork,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	// Validate contract
	if err := helper.ValidateContractForSuggestion(contract); err != nil {
		zap.L().Error("Contract validation failed", zap.Error(err))
		return nil, fmt.Errorf("invalid contract: %w", err)
	}

	// Parse deliverables and financial terms
	deliverables, err := scopeOfWork.Deliverables.ToAffiliateDeliverable()
	if err != nil {
		zap.L().Error("Failed to convert deliverables to AffiliateDeliverable", zap.Error(err))
		return nil, fmt.Errorf("failed to parse affiliate deliverables: %w", err)
	}

	var financialTerms dtos.AffiliateFinancialTerms
	if err = json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
		zap.L().Error("Failed to unmarshal financial terms to AffiliateFinancialTerms", zap.Error(err))
		return nil, fmt.Errorf("failed to parse financial terms: %w", err)
	}

	// Generate milestones from payment cycle
	dueDates, err := helper.GenerateMilestoneDueDatesFromFinancialTerms(contract, financialTerms, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to generate milestone dates: %w", err)
	}

	if len(dueDates) == 0 {
		return nil, errors.New("no milestone dates generated from financial terms")
	}

	// Calculate base payment per period
	totalCost, err := helper.ExtractTotalCostFromFinancialTerms(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to extract total cost: %w", err)
	}

	depositPercent := float64(0)
	if contract.DepositPercent != nil {
		depositPercent = float64(*contract.DepositPercent)
	}

	basePayment := helper.CalculateBasePaymentPerPeriod(totalCost, depositPercent, len(dueDates))

	// Create milestones
	milestones := make([]responses.SuggestedMilestone, len(dueDates))
	for i, dueDate := range dueDates {
		milestones[i] = responses.SuggestedMilestone{
			Description: fmt.Sprintf("Payment Period (Due: %s) - Base: %.0f VND + CTR Performance",
				dueDate.Format(utils.DateFormat),
				basePayment),
			DueDate: dueDate,
			Tasks:   []responses.SuggestedTask{},
		}
	}

	// Extract content creation tasks (all go to first milestone)
	contentTasks := make([]responses.SuggestedTask, 0, len(deliverables.AdvertisedItems))
	for _, item := range deliverables.AdvertisedItems {
		task := helper.TransformAdvertisedItemToTask(item, dueDates[0])

		// Add tracking link to task description
		task.Description["tracking_link"] = deliverables.TrackingLink
		task.Description["is_affiliate_content"] = true

		contentTasks = append(contentTasks, task)
	}

	// Assign tasks to milestones using affiliate strategy
	assignedMilestones := helper.AssignAffiliateTasksToMilestones(contentTasks, milestones, deliverables.TrackingLink)

	// Generate campaign name
	campaignName := "Affiliate Marketing Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	zap.L().Info("Extracted affiliate campaign structure",
		zap.Int("content_tasks", len(contentTasks)),
		zap.Int("milestones", len(assignedMilestones)),
		zap.String("tracking_link", deliverables.TrackingLink))

	return &responses.SuggestedCampaign{
		Name:        campaignName,
		Description: utils.PtrOrNil(fmt.Sprintf("Affiliate campaign with %d content pieces and %d payment periods", len(contentTasks), len(assignedMilestones))),
		StartDate:   contract.StartDate,
		EndDate:     contract.EndDate,
		Type:        contract.Type,
		ContractID:  contract.ID,
		Milestones:  assignedMilestones,
	}, nil
}

// extractBrandAmbassadorTasks extracts tasks from BRAND_AMBASSADOR contract deliverables
func (c *CampaignService) extractBrandAmbassadorTasks(
	_ context.Context,
	scopeOfWork dtos.ScopeOfWork,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	// Validate contract
	if err := helper.ValidateContractForSuggestion(contract); err != nil {
		zap.L().Error("Contract validation failed", zap.Error(err))
		return nil, fmt.Errorf("invalid contract: %w", err)
	}

	// Parse deliverables and financial terms
	deliverables, err := scopeOfWork.Deliverables.ToBrandAmbassadorDeliverable()
	if err != nil {
		zap.L().Error("Failed to convert deliverables to BrandAmbassadorDeliverable", zap.Error(err))
		return nil, fmt.Errorf("failed to parse brand ambassador deliverables: %w", err)
	}

	var financialTerms dtos.AdvertisingFinancialTerms
	if err = json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
		zap.L().Error("Failed to unmarshal financial terms to AdvertisingFinancialTerms", zap.Error(err))
		return nil, fmt.Errorf("failed to parse financial terms: %w", err)
	}

	// Generate milestones from schedules (same as ADVERTISING)
	dueDates, err := helper.GenerateMilestoneDueDatesFromFinancialTerms(contract, financialTerms, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to generate milestone dates: %w", err)
	}

	if len(dueDates) == 0 {
		return nil, errors.New("no milestone dates generated from financial terms")
	}

	// Calculate base payment per period
	totalCost, err := helper.ExtractTotalCostFromFinancialTerms(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to extract total cost: %w", err)
	}

	depositPercent := float64(0)
	if contract.DepositPercent != nil {
		depositPercent = float64(*contract.DepositPercent)
	}

	basePayment := helper.CalculateBasePaymentPerPeriod(totalCost, depositPercent, len(dueDates))

	// Create milestones
	milestones := make([]responses.SuggestedMilestone, len(dueDates))
	for i, dueDate := range dueDates {
		milestones[i] = responses.SuggestedMilestone{
			Description: fmt.Sprintf("Phase %d: Event Period (Due: %s) - Amount: %.0f VND",
				i+1,
				dueDate.Format(utils.DateFormat),
				basePayment),
			DueDate: dueDate,
			Tasks:   []responses.SuggestedTask{},
		}
	}

	// Extract event tasks
	eventTasks := make([]responses.SuggestedTask, 0, len(deliverables.Events))
	for _, event := range deliverables.Events {
		task := helper.TransformEventToTask(event)
		eventTasks = append(eventTasks, task)
	}

	// Assign tasks to milestones by date (closest milestone to event date)
	assignedMilestones := helper.AssignTasksByDate(eventTasks, milestones)

	// Generate campaign name
	campaignName := "Brand Ambassador Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	zap.L().Info("Extracted brand ambassador campaign structure",
		zap.Int("events", len(eventTasks)),
		zap.Int("milestones", len(assignedMilestones)))

	return &responses.SuggestedCampaign{
		Name:        campaignName,
		Description: utils.PtrOrNil(fmt.Sprintf("Brand ambassador campaign with %d events", len(eventTasks))),
		StartDate:   contract.StartDate,
		EndDate:     contract.EndDate,
		Type:        contract.Type,
		ContractID:  contract.ID,
		Milestones:  assignedMilestones,
	}, nil
}

// extractCoProducingStructure extracts milestones and tasks from CO_PRODUCING contract deliverables
func (c *CampaignService) extractCoProducingStructure(
	_ context.Context,
	scopeOfWork dtos.ScopeOfWork,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	// Convert to type-safe deliverables
	deliverables, err := scopeOfWork.Deliverables.ToCoProducingDeliverable()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to co-producing deliverables: %w", err)
	}

	// Extract financial terms to get profit distribution cycle
	var financialTerms dtos.CoProducingFinancialTerms
	if err = json.Unmarshal(contract.FinancialTerms, &financialTerms); err != nil {
		return nil, fmt.Errorf("failed to unmarshal co-producing financial terms: %w", err)
	}

	// Generate milestone due dates based on profit distribution cycle
	milestoneDueDates, err := helper.GenerateMilestoneDueDatesFromFinancialTerms(
		contract,
		financialTerms,
		5,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate milestone due dates: %w", err)
	}

	if len(milestoneDueDates) == 0 {
		return nil, errors.New("no milestone due dates generated for co-producing contract")
	}

	// Create milestone structures with due dates (empty tasks initially)
	milestones := make([]responses.SuggestedMilestone, len(milestoneDueDates))
	for i, dueDate := range milestoneDueDates {
		milestones[i] = responses.SuggestedMilestone{
			Description: fmt.Sprintf("Co-Production Milestone %d", i+1),
			DueDate:     dueDate,
			Tasks:       []responses.SuggestedTask{},
		}
	}

	// Extract all product creation tasks and concept tasks from deliverables
	// Use first milestone due date as deadline for all development tasks
	firstMilestoneDueDate := milestoneDueDates[0]
	productTasks := helper.ExtractProductCreationTasks(deliverables.Products, firstMilestoneDueDate)
	conceptTasks := helper.ExtractConceptTasks(deliverables.Concepts, deliverables.Products, firstMilestoneDueDate)

	// Combine all development tasks
	allDevelopmentTasks := append(productTasks, conceptTasks...)

	if len(allDevelopmentTasks) == 0 {
		return nil, errors.New("no product or concept tasks found in co-producing deliverables")
	}

	// Extract product names for tracking task metadata
	productNames := make([]string, 0, len(deliverables.Products))
	for _, product := range deliverables.Products {
		productNames = append(productNames, product.Name)
	}

	// Assign tasks to milestones (all dev tasks to first milestone, sales review to others)
	milestones = helper.AssignCoProducingTasksToMilestones(allDevelopmentTasks, milestones, productNames)

	campaignName := "Co-Production Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	return &responses.SuggestedCampaign{
		Name: campaignName,
		Description: utils.PtrOrNil(fmt.Sprintf("Co-Production Campaign for contract %s with %d products and %d concepts adveritsement",
			*contract.ContractNumber, len(deliverables.Products), len(deliverables.Concepts))),
		Type:       contract.Type,
		StartDate:  contract.StartDate,
		EndDate:    contract.EndDate,
		ContractID: contract.ID,
		Milestones: milestones,
	}, nil
}

// endregion

func NewCampaignService(
	dbReg *gormrepository.DatabaseRegistry,
) iservice.CampaignService {
	return &CampaignService{
		campaignRepo:       dbReg.CampaignRepository,
		contractRepo:       dbReg.ContractRepository,
		orderRepo:          dbReg.OrderRepository,
		preOrderRepo:       dbReg.PreOrderRepository,
		contentChannelRepo: dbReg.ContentChannelRepository,
		affiliateLinkRepo:  dbReg.AffiliateLinkRepository,
		kpiMetricsRepo:     dbReg.KPIMetricsRepository,
	}
}
