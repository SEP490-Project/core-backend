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
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

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
		Description: fmt.Sprintf("Campaign for contract %s with %d advertised items", *contract.ContractNumber, len(deliverables.AdvertisedItems)),
		StartDate:   utils.FormatLocalTime(&contract.StartDate, ""),
		EndDate:     utils.FormatLocalTime(&contract.EndDate, ""),
		Type:        contract.Type.String(),
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
			DueDate: dueDate.Format(utils.DateFormat),
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
			DueDate: dueDate.Format(utils.DateFormat),
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
		Description: fmt.Sprintf("Affiliate campaign with %d content pieces and %d payment periods", len(contentTasks), len(assignedMilestones)),
		StartDate:   utils.FormatLocalTime(&contract.StartDate, ""),
		EndDate:     utils.FormatLocalTime(&contract.EndDate, ""),
		Type:        contract.Type.String(),
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
			DueDate: dueDate.Format(utils.DateFormat),
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
		Description: fmt.Sprintf("Brand ambassador campaign with %d events", len(eventTasks)),
		StartDate:   utils.FormatLocalTime(&contract.StartDate, ""),
		EndDate:     utils.FormatLocalTime(&contract.EndDate, ""),
		Type:        contract.Type.String(),
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
			DueDate:     dueDate.Format(utils.DateFormat),
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
		Name:       campaignName,
		Type:       string(contract.Type),
		StartDate:  contract.StartDate.Format(utils.DateFormat),
		EndDate:    contract.EndDate.Format(utils.DateFormat),
		Milestones: milestones,
	}, nil
}

// endregion

func NewCampaignService(
	campaignRepo irepository.GenericRepository[model.Campaign],
	contractRepo irepository.GenericRepository[model.Contract],
) iservice.CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
		contractRepo: contractRepo,
	}
}
