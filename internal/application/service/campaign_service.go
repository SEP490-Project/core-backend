package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CampaignService struct {
	campaignRepo irepository.GenericRepository[model.Campaign]
	contractRepo irepository.GenericRepository[model.Contract]
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

// SuggestCampaignFromContract implements iservice.CampaignService.
func (c *CampaignService) SuggestCampaignFromContract(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.CampaignSuggestionResponse, error) {
	zap.L().Info("Suggesting campaign from contract", zap.String("contract_id", contractID.String()))

	// Retrieve contract with necessary fields
	contract, err := c.contractRepo.GetByID(ctx, contractID, nil)
	if err != nil {
		zap.L().Error("Failed to retrieve contract", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, errors.New("contract not found")
	}
	if contract == nil {
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
	var scopeOfWork map[string]any
	if err = json.Unmarshal(contract.ScopeOfWork, &scopeOfWork); err != nil {
		zap.L().Error("Failed to parse scope of work", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, errors.New("invalid scope of work format")
	}

	// Extract deliverables based on contract type
	var suggestedCampaign *responses.SuggestedCampaign
	switch contract.Type {
	case "ADVERTISING":
		suggestedCampaign, err = c.extractAdvertisingTasks(scopeOfWork, contract)
	case "AFFILIATE":
		suggestedCampaign, err = c.extractAffiliateTasks(scopeOfWork, contract)
	case "BRAND_AMBASSADOR":
		suggestedCampaign, err = c.extractBrandAmbassadorTasks(scopeOfWork, contract)
	case "CO_PRODUCING":
		suggestedCampaign, err = c.extractCoProducingStructure(scopeOfWork, contract)
	default:
		zap.L().Error("Unsupported contract type", zap.String("contract_id", contractID.String()), zap.String("type", string(contract.Type)))
		return nil, errors.New("unsupported contract type")
	}

	if err != nil {
		zap.L().Error("Failed to extract campaign structure", zap.String("contract_id", contractID.String()), zap.Error(err))
		return nil, err
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
	scopeOfWork map[string]any,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	deliverables, ok := scopeOfWork["deliverables"].([]any)
	if !ok || len(deliverables) == 0 {
		return nil, errors.New("no deliverables found in scope of work")
	}

	var tasks []responses.SuggestedTask
	for _, item := range deliverables {
		deliverable, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Extract advertised_items array
		advertisedItems, ok := deliverable["advertised_items"].([]any)
		if !ok || len(advertisedItems) == 0 {
			continue
		}

		for _, adItem := range advertisedItems {
			adItemMap, ok := adItem.(map[string]any)
			if !ok {
				continue
			}

			taskName := fmt.Sprintf("Advertise %s", adItemMap["name"])
			taskDesc := map[string]any{
				"item_name":        adItemMap["name"],
				"item_description": adItemMap["description"],
				"quantity":         adItemMap["quantity"],
				"content_type":     deliverable["content_type"],
				"platform":         deliverable["platform"],
			}

			tasks = append(tasks, responses.SuggestedTask{
				Name:            taskName,
				DescriptionJSON: taskDesc,
			})
		}
	}

	if len(tasks) == 0 {
		return nil, errors.New("no valid advertising tasks could be extracted")
	}

	milestone := responses.SuggestedMilestone{
		Name:  "Content Creation & Publication",
		Tasks: tasks,
	}

	campaignName := "Advertising Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	return &responses.SuggestedCampaign{
		Name:       campaignName,
		Milestones: []responses.SuggestedMilestone{milestone},
	}, nil
}

// extractAffiliateTasks extracts tasks from AFFILIATE contract deliverables
func (c *CampaignService) extractAffiliateTasks(
	scopeOfWork map[string]any,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	// Affiliate contracts extend advertising with tracking links
	suggestedCampaign, err := c.extractAdvertisingTasks(scopeOfWork, contract)
	if err != nil {
		return nil, err
	}

	// Add tracking link information to each task
	deliverables, _ := scopeOfWork["deliverables"].([]any)
	for i, item := range deliverables {
		deliverable, ok := item.(map[string]any)
		if !ok {
			continue
		}

		trackingLink, _ := deliverable["tracking_link"].(string)
		platform, _ := deliverable["platform"].(string)

		if i < len(suggestedCampaign.Milestones[0].Tasks) {
			task := &suggestedCampaign.Milestones[0].Tasks[i]
			if task.DescriptionJSON == nil {
				task.DescriptionJSON = make(map[string]any)
			}
			task.DescriptionJSON["tracking_link"] = trackingLink
			task.DescriptionJSON["affiliate_platform"] = platform
		}
	}

	// Update campaign name
	suggestedCampaign.Name = "Affiliate Marketing Campaign"
	if contract.Title != nil {
		suggestedCampaign.Name = *contract.Title
	}

	return suggestedCampaign, nil
}

// extractBrandAmbassadorTasks extracts tasks from BRAND_AMBASSADOR contract deliverables
func (c *CampaignService) extractBrandAmbassadorTasks(
	scopeOfWork map[string]any,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	deliverables, ok := scopeOfWork["deliverables"].([]any)
	if !ok || len(deliverables) == 0 {
		return nil, errors.New("no deliverables found in scope of work")
	}

	var tasks []responses.SuggestedTask
	for _, item := range deliverables {
		deliverable, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Extract events array
		events, ok := deliverable["events"].([]any)
		if !ok || len(events) == 0 {
			continue
		}

		for _, evt := range events {
			eventMap, ok := evt.(map[string]any)
			if !ok {
				continue
			}

			eventName, _ := eventMap["name"].(string)
			taskName := fmt.Sprintf("Represent Brand at %s", eventName)

			taskDesc := map[string]any{
				"event_name":    eventMap["name"],
				"event_date":    eventMap["date"],
				"location":      eventMap["location"],
				"activities":    eventMap["activities"],
				"expected_kpis": eventMap["expected_kpis"],
				"content_type":  deliverable["content_type"],
				"platform":      deliverable["platform"],
			}

			tasks = append(tasks, responses.SuggestedTask{
				Name:            taskName,
				DescriptionJSON: taskDesc,
			})
		}
	}

	if len(tasks) == 0 {
		return nil, errors.New("no valid brand ambassador tasks could be extracted")
	}

	milestone := responses.SuggestedMilestone{
		Name:  "Brand Representation & Events",
		Tasks: tasks,
	}

	campaignName := "Brand Ambassador Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	return &responses.SuggestedCampaign{
		Name:       campaignName,
		Milestones: []responses.SuggestedMilestone{milestone},
	}, nil
}

// extractCoProducingStructure extracts milestones and tasks from CO_PRODUCING contract deliverables
func (c *CampaignService) extractCoProducingStructure(
	scopeOfWork map[string]any,
	contract *model.Contract,
) (*responses.SuggestedCampaign, error) {
	deliverables, ok := scopeOfWork["deliverables"].([]any)
	if !ok || len(deliverables) == 0 {
		return nil, errors.New("no deliverables found in scope of work")
	}

	var milestones []responses.SuggestedMilestone

	// Each product becomes a milestone
	for _, item := range deliverables {
		deliverable, ok := item.(map[string]any)
		if !ok {
			continue
		}

		products, ok := deliverable["products"].([]any)
		if !ok || len(products) == 0 {
			continue
		}

		for _, prod := range products {
			productMap, ok := prod.(map[string]any)
			if !ok {
				continue
			}

			productName, _ := productMap["name"].(string)
			milestoneName := fmt.Sprintf("Co-Produce: %s", productName)

			var tasks []responses.SuggestedTask

			// Each concept within a product becomes a task
			concepts, ok := productMap["concepts"].([]any)
			if ok && len(concepts) > 0 {
				for _, concept := range concepts {
					conceptMap, ok := concept.(map[string]any)
					if !ok {
						continue
					}

					conceptName, _ := conceptMap["name"].(string)
					taskName := fmt.Sprintf("Develop Concept: %s", conceptName)

					taskDesc := map[string]any{
						"product_name":        productName,
						"concept_name":        conceptName,
						"concept_description": conceptMap["description"],
						"milestones":          conceptMap["milestones"],
						"deliverable_date":    conceptMap["deliverable_date"],
					}

					tasks = append(tasks, responses.SuggestedTask{
						Name:            taskName,
						DescriptionJSON: taskDesc,
					})
				}
			}

			if len(tasks) > 0 {
				milestones = append(milestones, responses.SuggestedMilestone{
					Name:  milestoneName,
					Tasks: tasks,
				})
			}
		}
	}

	if len(milestones) == 0 {
		return nil, errors.New("no valid co-producing milestones could be extracted")
	}

	campaignName := "Co-Production Campaign"
	if contract.Title != nil {
		campaignName = *contract.Title
	}

	return &responses.SuggestedCampaign{
		Name:       campaignName,
		Milestones: milestones,
	}, nil
}

func NewCampaignService(
	campaignRepo irepository.GenericRepository[model.Campaign],
	contractRepo irepository.GenericRepository[model.Contract],
) iservice.CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
		contractRepo: contractRepo,
	}
}
