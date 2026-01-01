package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContractService struct {
	brandRepository    irepository.GenericRepository[model.Brand]
	contractRepository irepository.GenericRepository[model.Contract]
	taskRepository     irepository.TaskRepository
	unitOfWork         irepository.UnitOfWork
}

// GetContractsByUserID implements iservice.ContractService.
func (s *ContractService) GetContractsByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filterRequest *requests.ContractFilterRequest,
) ([]*responses.ContractListResponse, int64, error) {
	zap.L().Info("Retrieving contracts by user ID", zap.String("user_id", userID.String()))

	query := func(db *gorm.DB) *gorm.DB {
		db = db.
			InnerJoins("INNER JOIN brands on brands.id = contracts.brand_id").
			Where("brands.user_id = ?", userID)
		// Filter by brand ID
		if filterRequest.BrandID != nil && *filterRequest.BrandID != "" {
			brandID, err := uuid.Parse(*filterRequest.BrandID)
			if err == nil {
				db = db.Where("brand_id = ?", brandID)
			}
		}

		// Filter by type
		if filterRequest.Type != nil && *filterRequest.Type != "" {
			contractType := enum.ContractType(*filterRequest.Type)
			if contractType.IsValid() {
				db = db.Where("type = ?", contractType)
			}
		}

		// Filter by status
		if filterRequest.Status != nil && *filterRequest.Status != "" {
			contractStatus := enum.ContractStatus(*filterRequest.Status)
			if contractStatus.IsValid() {
				db = db.Where("status = ?", contractStatus)
			}
		}

		// Filter by keyword (search in title and contract number)
		if filterRequest.Keyword != nil && *filterRequest.Keyword != "" {
			likePattern := fmt.Sprintf("%%%s%%", *filterRequest.Keyword)
			db = db.Where("title ILIKE ? OR contract_number ILIKE ?", likePattern, likePattern)
		}

		// Filter by date range
		if filterRequest.StartDate != nil {
			startDate := utils.ParseLocalTimeWithFallback(*filterRequest.StartDate, utils.DateFormat)
			if startDate != nil {
				db.Where("start_date >= ?", startDate)
			}
		}
		if filterRequest.EndDate != nil {
			endDate := utils.ParseLocalTimeWithFallback(*filterRequest.EndDate, utils.DateFormat)
			if endDate != nil {
				db = db.Where("end_date <= ?", endDate)
			}
		}

		// Sorting
		db = db.Order(helper.ConvertToSortString(filterRequest.PaginationRequest))

		return db
	}
	contracts, totalCount, err := s.contractRepository.GetAll(ctx, query, []string{"Brand", "Campaign", "ContractPayments"}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to retrieve campaigns by user ID",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, 0, err
	}

	var result []*responses.ContractListResponse
	for _, contract := range contracts {
		result = append(result, responses.ToContractListResponse(&contract))
	}

	zap.L().Info("Contracts fetched successfully",
		zap.String("user_id", userID.String()),
		zap.Int64("total", totalCount))

	return result, totalCount, nil
}

// ApproveContract implements iservice.ContractService.
func (s *ContractService) ApproveContract(ctx context.Context, contractID uuid.UUID) error {
	zap.L().Info("Approving contract", zap.String("contract_id", contractID.String()))

	var contract *model.Contract
	var err error
	if contract, err = s.contractRepository.GetByID(ctx, contractID, []string{"Brand"}); err != nil {
		zap.L().Error("Failed to check contract existence", zap.Error(err))
		return errors.New("failed to verify contract")
	} else if contract == nil {
		zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
		return errors.New("contract not found")
	}

	contract.Status = enum.ContractStatusActive
	if err = s.contractRepository.Update(ctx, contract); err != nil {
		zap.L().Error("Failed to approve contract", zap.Error(err))
		return errors.New("failed to approve contract")
	}

	zap.L().Info("Contract approved successfully",
		zap.String("contract_id", contractID.String()),
		zap.String("contract_number", *contract.ContractNumber),
		zap.String("status", string(contract.Status)),
	)
	return nil
}

// CreateContract implements iservice.ContractService.
func (s *ContractService) CreateContract(
	ctx context.Context,
	userID uuid.UUID,
	createRequest *requests.CreateContractRequest,
	unitOfWork irepository.UnitOfWork,
) (*responses.ContractResponse, error) {
	zap.L().Info("Creating new contract",
		zap.String("brand_id", createRequest.BrandID),
		zap.String("title", createRequest.Title))

	// Convert request to contract model
	contract, err := createRequest.ToContract(ctx)
	if err != nil {
		zap.L().Error("Failed to convert create request to contract model", zap.Error(err))
		return nil, err
	}

	contractRepo := unitOfWork.Contracts()

	// Verify brand exists
	brandID, _ := uuid.Parse(createRequest.BrandID)
	brandRepo := unitOfWork.Brands()
	_, err = brandRepo.GetByID(ctx, brandID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Brand not found", zap.String("brand_id", createRequest.BrandID))
			return nil, errors.New("brand not found")
		}
		zap.L().Error("Failed to fetch brand", zap.Error(err))
		return nil, errors.New("failed to verify brand")
	}

	// Check for duplicate contract number
	exists, err := contractRepo.Exists(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_number = ?", contract.ContractNumber)
	})
	if err != nil {
		zap.L().Error("Failed to check contract number uniqueness", zap.Error(err))
		return nil, errors.New("failed to validate contract number")
	}
	if exists {
		zap.L().Warn("Contract number already exists", zap.String("contract_number", *contract.ContractNumber))
		return nil, fmt.Errorf("contract number %s already exists", *contract.ContractNumber)
	}

	// Create contract
	contract.CreatedByID = userID
	if err = contractRepo.Add(ctx, contract); err != nil {
		zap.L().Error("Failed to create contract", zap.Error(err))
		return nil, errors.New("failed to create contract")
	}

	// Retrieve created contract with relationships
	createdContract, err := contractRepo.GetByID(ctx, contract.ID, []string{"Brand", "ParentContract"})
	if err != nil {
		zap.L().Error("Failed to retrieve created contract", zap.Error(err))
		return nil, errors.New("contract created but failed to retrieve details")
	}

	zap.L().Info("Contract created successfully",
		zap.String("contract_id", contract.ID.String()),
		zap.String("contract_number", *contract.ContractNumber))

	return responses.ContractResponse{}.ToContractResponse(createdContract, nil)
}

// UpdateContract implements iservice.ContractService.
func (s *ContractService) UpdateContract(
	ctx context.Context,
	contractID uuid.UUID,
	updateRequest *requests.UpdateContractRequest,
	unitOfWork irepository.UnitOfWork,
) (*responses.ContractResponse, error) {
	zap.L().Info("Updating contract",
		zap.String("contract_id", contractID.String()))

	// Begin transaction
	uow := unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	contractRepo := uow.Contracts()

	// Get existing contract
	contract, err := contractRepo.GetByID(ctx, contractID, nil)
	if err != nil {
		uow.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return nil, errors.New("contract not found")
		}
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return nil, errors.New("failed to fetch contract")
	}

	// If contract number is being updated, check uniqueness
	if updateRequest.ContractNumber != nil && *updateRequest.ContractNumber != *contract.ContractNumber {
		var exists bool
		exists, err = contractRepo.Exists(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_number = ? AND id != ?", *updateRequest.ContractNumber, contractID)
		})
		if err != nil {
			uow.Rollback()
			zap.L().Error("Failed to check contract number uniqueness", zap.Error(err))
			return nil, errors.New("failed to validate contract number")
		}
		if exists {
			uow.Rollback()
			zap.L().Warn("Contract number already exists", zap.String("contract_number", *updateRequest.ContractNumber))
			return nil, fmt.Errorf("contract number %s already exists", *updateRequest.ContractNumber)
		}
	}

	// Apply updates to contract
	if err = updateRequest.ApplyToContract(contract); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to apply updates to contract", zap.Error(err))
		return nil, err
	}

	// Update contract
	if err = contractRepo.Update(ctx, contract); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to update contract", zap.Error(err))
		return nil, errors.New("failed to update contract")
	}

	// Commit transaction
	if err = uow.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		return nil, errors.New("failed to save contract updates")
	}

	// Retrieve updated contract with relationships
	updatedContract, err := s.contractRepository.GetByID(ctx, contractID, []string{"Brand", "ParentContract"})
	if err != nil {
		zap.L().Error("Failed to retrieve updated contract", zap.Error(err))
		return nil, errors.New("contract updated but failed to retrieve details")
	}

	zap.L().Info("Contract updated successfully",
		zap.String("contract_id", contractID.String()))

	return responses.ContractResponse{}.ToContractResponse(updatedContract, nil)
}

// UpdateContractFileURL implements iservice.ContractService.
func (s *ContractService) UpdateContractFileURL(
	ctx context.Context,
	contractID uuid.UUID,
	fileURL string,
	uow irepository.UnitOfWork,
) error {
	zap.L().Info("Updating contract file URL",
		zap.String("contract_id", contractID.String()),
		zap.String("file_url", fileURL))

	var contract *model.Contract
	var err error
	if contract, err = s.contractRepository.GetByID(ctx, contractID, nil); err != nil {
		zap.L().Error("Failed to check contract existence", zap.Error(err))
		return errors.New("failed to verify contract")
	}

	contract.ContractFileURL = &fileURL
	if err = s.contractRepository.Update(ctx, contract); err != nil {
		zap.L().Error("Failed to update contract file URL", zap.Error(err))
		return errors.New("failed to update contract file URL")
	}

	zap.L().Info("Contract file URL updated successfully",
		zap.String("contract_id", contractID.String()),
		zap.String("file_url", fileURL))
	return nil
}

// GetContractByID implements iservice.ContractService.
func (s *ContractService) GetContractByID(ctx context.Context, contractID uuid.UUID) (*responses.ContractResponse, error) {
	zap.L().Info("Fetching contract by ID", zap.String("contract_id", contractID.String()))

	contract, err := s.contractRepository.GetByID(ctx, contractID, []string{"Brand", "ParentContract", "Campaign", "ContractPayments"})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return nil, errors.New("contract not found")
		}
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return nil, errors.New("failed to fetch contract")
	}

	return responses.ContractResponse{}.ToContractResponse(contract, nil)
}

// GetContractsByBrandID implements iservice.ContractService.
func (s *ContractService) GetContractsByBrandID(ctx context.Context, brandID uuid.UUID, page, limit int) ([]*responses.ContractListResponse, int64, error) {
	zap.L().Info("Fetching contracts by brand ID",
		zap.String("brand_id", brandID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit))

	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ?", brandID).Order("created_at DESC")
	}

	contracts, total, err := s.contractRepository.GetAll(ctx, filter, []string{"Brand", "Campaign"}, limit, page)
	if err != nil {
		zap.L().Error("Failed to fetch contracts", zap.Error(err))
		return nil, 0, errors.New("failed to fetch contracts")
	}

	result := make([]*responses.ContractListResponse, 0, len(contracts))
	for _, contract := range contracts {
		result = append(result, responses.ToContractListResponse(&contract))
	}

	zap.L().Info("Contracts fetched successfully",
		zap.String("brand_id", brandID.String()),
		zap.Int64("total", total))

	return result, total, nil
}

// GetByFilter implements iservice.ContractService.
func (s *ContractService) GetByFilter(ctx context.Context, filterReq *requests.ContractFilterRequest) ([]*responses.ContractListResponse, int64, error) {
	zap.L().Info("Fetching contracts with filter", zap.Any("filter", filterReq))

	filter := func(db *gorm.DB) *gorm.DB {
		// Filter by brand ID
		if filterReq.BrandID != nil && *filterReq.BrandID != "" {
			brandID, err := uuid.Parse(*filterReq.BrandID)
			if err == nil {
				db = db.Where("brand_id = ?", brandID)
			}
		}

		// Filter by type
		if filterReq.Type != nil && *filterReq.Type != "" {
			contractType := enum.ContractType(*filterReq.Type)
			if contractType.IsValid() {
				db = db.Where("type = ?", contractType)
			}
		}

		// Filter by status
		if filterReq.Status != nil && *filterReq.Status != "" {
			contractStatus := enum.ContractStatus(*filterReq.Status)
			if contractStatus.IsValid() {
				db = db.Where("status = ?", contractStatus)
			}
		}

		// Filter by keyword (search in title and contract number)
		if filterReq.Keyword != nil && *filterReq.Keyword != "" {
			likePattern := fmt.Sprintf("%%%s%%", *filterReq.Keyword)
			db = db.Where("title ILIKE ? OR contract_number ILIKE ?", likePattern, likePattern)
		}

		// Filter by date range
		if filterReq.StartDate != nil {
			startDate := utils.ParseLocalTimeWithFallback(*filterReq.StartDate, utils.DateFormat)
			if startDate != nil {
				db.Where("start_date >= ?", startDate)
			}
		}
		if filterReq.EndDate != nil {
			endDate := utils.ParseLocalTimeWithFallback(*filterReq.EndDate, utils.DateFormat)
			if endDate != nil {
				db = db.Where("end_date <= ?", endDate)
			}
		}

		// if filterReq.NoCampaign != nil {
		// 	if *filterReq.NoCampaign {
		// 		db = db.Where("campaign_id IS NULL")
		// 	} else {
		// 		db = db.Where("campaign_id IS NOT NULL")
		// 	}
		// }

		if filterReq.NoCampaign != nil {
			if *filterReq.NoCampaign {
				db = db.Where("NOT EXISTS (SELECT 1 FROM campaigns WHERE campaigns.contract_id = contracts.id)")
			} else {
				db = db.Where("EXISTS (SELECT 1 FROM campaigns WHERE campaigns.contract_id = contracts.id)")
			}
		}

		db = db.Order(helper.ConvertToSortString(filterReq.PaginationRequest))

		return db
	}

	contracts, total, err := s.contractRepository.GetAll(ctx, filter, []string{"Brand", "Campaign"}, filterReq.Limit, filterReq.Page)
	if err != nil {
		zap.L().Error("Failed to fetch contracts", zap.Error(err))
		return nil, 0, errors.New("failed to fetch contracts")
	}

	result := make([]*responses.ContractListResponse, len(contracts))
	for i, contract := range contracts {
		result[i] = responses.ToContractListResponse(&contract)
	}

	zap.L().Info("Contracts fetched successfully", zap.Int64("total", total))

	return result, total, nil
}

// DeleteContractByID implements iservice.ContractService.
func (s *ContractService) DeleteContractByID(ctx context.Context, contractID uuid.UUID) error {
	zap.L().Info("Deleting contract", zap.String("contract_id", contractID.String()))

	// Check if contract exists
	contract, err := s.contractRepository.GetByID(ctx, contractID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return errors.New("contract not found")
		}
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return errors.New("failed to fetch contract")
	}

	// Soft delete
	if err := s.contractRepository.Delete(ctx, contract); err != nil {
		zap.L().Error("Failed to delete contract", zap.Error(err))
		return errors.New("failed to delete contract")
	}

	zap.L().Info("Contract deleted successfully", zap.String("contract_id", contractID.String()))

	return nil
}

// ValidateBrandAndContractNumber implements iservice.ContractService.
func (s *ContractService) ValidateBrandAndContractNumber(ctx context.Context, brandID uuid.UUID, contractNumber string) error {
	zap.L().Info("Validating brand ID and contract number",
		zap.String("brand_id", brandID.String()),
		zap.String("contract_number", contractNumber))

	brandExists, err := s.brandRepository.Exists(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	})
	if err != nil {
		zap.L().Error("Failed to check brand existence", zap.Error(err))
		return errors.New("failed to verify brand")
	} else if !brandExists {
		zap.L().Warn("Brand not found", zap.String("brand_id", brandID.String()))
		return errors.New("brand not found")
	}

	// Check if contract with same brand ID and contract number exists
	exists, err := s.contractRepository.Exists(ctx, func(db *gorm.DB) *gorm.DB {
		return db.
			Where("brand_id = ?", brandID).
			Where("contract_number = ?", contractNumber)
	})
	if err != nil {
		zap.L().Error("Failed to check contract existence", zap.Error(err))
		return errors.New("failed to verify contract")
	} else if exists {
		zap.L().Warn("Contract already exists", zap.String("brand_id", brandID.String()), zap.String("contract_number", contractNumber))
		return fmt.Errorf("contract number %s already exists", contractNumber)
	}

	zap.L().Info("Brand ID and contract number are valid",
		zap.String("brand_id", brandID.String()),
		zap.String("contract_number", contractNumber))
	return nil
}

// GetScopeOfWorkByContractID implements iservice.ContractService.
func (s *ContractService) GetScopeOfWorkByContractID(ctx context.Context, contractID uuid.UUID) (any, error) {
	zap.L().Info("Fetching scope of work by contract ID", zap.String("contract_id", contractID.String()))

	contract, err := s.contractRepository.GetByID(ctx, contractID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return nil, errors.New("contract not found")
		}
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return nil, errors.New("failed to fetch contract")
	}

	if contract.ScopeOfWork == nil {
		return nil, nil
	}
	var scopeOfWork any
	if err = json.Unmarshal(contract.ScopeOfWork, &scopeOfWork); err != nil {
		zap.L().Error("Failed to unmarshal scope of work", zap.Error(err))
		return nil, errors.New("failed to unmarshal scope of work")
	}

	return scopeOfWork, nil
}

func (s *ContractService) UpdateContractScopeOfWorkWithReferencinnTaskIDs(
	ctx context.Context, contractID uuid.UUID,
) error {
	zap.L().Info("Updating contract scope of work with referencing task IDs",
		zap.String("contract_id", contractID.String()))

	var (
		err                      error
		contract                 *model.Contract
		scopeOfWork              *dtos.ScopeOfWork
		taskIDs                  []uuid.UUID
		taskWithScopeOfWork      []dtos.TaskWithScopeOfWorkID
		scopeOfWorkItemIDTypeMap = make(map[constant.ScopeOfWorkIDType]map[int8]*dtos.TaskWithScopeOfWorkID)
	)

	contractFunc := func(ctx context.Context) error {
		contract, err = s.contractRepository.GetByID(ctx, contractID, nil)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
				return errors.New("contract not found")
			}
			zap.L().Error("Failed to retrieve contract", zap.String("contract_id", contractID.String()), zap.Error(err))
			return err
		}
		scopeOfWork, err = s.unmarshalScopeOfWork(contract)
		if err != nil {
			zap.L().Error("Failed to unmarshal scope of work", zap.Error(err))
			return err
		}
		return nil
	}
	taskFunc := func(ctx context.Context) error {
		taskIDs, err = s.taskRepository.GetTaskIDsByContractID(ctx, contractID)
		if err != nil {
			zap.L().Error("Failed to retrieve task IDs by contract ID", zap.Error(err))
			return err
		}

		taskWithScopeOfWork, err = s.taskRepository.GetListTasksByIDs(ctx, taskIDs)
		if err != nil {
			zap.L().Error("Failed to retrieve tasks by IDs", zap.Error(err))
			return err
		}
		for _, task := range taskWithScopeOfWork {
			if task.ScopeOfWorkItemID == nil || task.ScopeOfWorkItemType == nil || task.ItemID == nil {
				continue
			}
			if _, exists := scopeOfWorkItemIDTypeMap[*task.ScopeOfWorkItemType]; !exists {
				scopeOfWorkItemIDTypeMap[*task.ScopeOfWorkItemType] = make(map[int8]*dtos.TaskWithScopeOfWorkID)
			}
			scopeOfWorkItemIDTypeMap[*task.ScopeOfWorkItemType][*task.ItemID] = &task
		}
		return nil
	}
	if err = utils.RunParallel(ctx, 2, contractFunc, taskFunc); err != nil {
		zap.L().Error("Failed to fetch contract and tasks in parallel", zap.Error(err))
		return err
	}

	switch contract.Type {
	case enum.ContractTypeAdvertising:
		advertiseDeliverables, err := scopeOfWork.Deliverables.ToAdvertisingDeliverable()
		if err != nil {
			zap.L().Error("Failed to convert deliverables to AffiliateDeliverable", zap.Error(err))
			return err
		}
		for i, item := range advertiseDeliverables.AdvertisedItems {
			if item.ID == nil {
				continue
			}
			taskMap, exists := scopeOfWorkItemIDTypeMap[constant.ScopeOfWorkIDTypeAdvertise]
			if !exists {
				break
			}
			if task, taskExists := taskMap[*item.ID]; taskExists && task.ContentInfo != nil {
				if len(item.ContentIDs) > 0 {
					advertiseDeliverables.AdvertisedItems[i].ContentIDs = []uuid.UUID{task.ContentInfo.ID}
				} else {
					advertiseDeliverables.AdvertisedItems[i].ContentIDs = append(advertiseDeliverables.AdvertisedItems[i].ContentIDs, task.ContentInfo.ID)
				}
			}
		}
		scopeOfWork.Deliverables.AdvertisingDeliverable = *advertiseDeliverables

	case enum.ContractTypeAffiliate:
		affiliateDeliverables, err := scopeOfWork.Deliverables.ToAffiliateDeliverable()
		if err != nil {
			zap.L().Error("Failed to convert deliverables to AffiliateDeliverable", zap.Error(err))
			return err
		}
		for i, item := range affiliateDeliverables.AdvertisedItems {
			if item.ID == nil {
				continue
			}
			taskMap, exists := scopeOfWorkItemIDTypeMap[constant.ScopeOfWorkIDTypeAffiliate]
			if !exists {
				break
			}
			if task, taskExists := taskMap[*item.ID]; taskExists && task.ContentInfo != nil {
				if len(item.ContentIDs) > 0 {
					affiliateDeliverables.AdvertisedItems[i].ContentIDs = []uuid.UUID{task.ContentInfo.ID}
				} else {
					affiliateDeliverables.AdvertisedItems[i].ContentIDs = append(affiliateDeliverables.AdvertisedItems[i].ContentIDs, task.ContentInfo.ID)
				}
			}
		}
		scopeOfWork.Deliverables.AffiliateDeliverable = *affiliateDeliverables

	case enum.ContractTypeCoProduce:
		coProducingDeliverables, err := scopeOfWork.Deliverables.ToCoProducingDeliverable()
		if err != nil {
			zap.L().Error("Failed to convert deliverables to CoProducingDeliverable", zap.Error(err))
			return err
		}
		// Product tasks
		for i, product := range coProducingDeliverables.Products {
			if product.ID == nil {
				continue
			}
			taskMap, exists := scopeOfWorkItemIDTypeMap[constant.ScopeOfWorkIDTypeProduct]
			if !exists {
				break
			}
			if task, taskExists := taskMap[*product.ID]; taskExists && task.ProductInfo != nil {
				if len(product.ProductIDs) > 0 {
					coProducingDeliverables.Products[i].ProductIDs = []uuid.UUID{task.ProductInfo.ID}
				} else {
					coProducingDeliverables.Products[i].ProductIDs = append(coProducingDeliverables.Products[i].ProductIDs, task.ProductInfo.ID)
				}
			}
		}

		// Concept tasks
		for i, concept := range coProducingDeliverables.Concepts {
			if concept.ID == nil {
				continue
			}
			taskMap, exists := scopeOfWorkItemIDTypeMap[constant.ScopeOfWorkIDTypeConcept]
			if !exists {
				break
			}
			if task, taskExists := taskMap[*concept.ID]; taskExists && task.ContentInfo != nil {
				if len(concept.ContentIDs) == 0 {
					coProducingDeliverables.Concepts[i].ContentIDs = []uuid.UUID{task.ContentInfo.ID}
				} else {
					coProducingDeliverables.Concepts[i].ContentIDs = append(coProducingDeliverables.Concepts[i].ContentIDs, task.ContentInfo.ID)
				}
			}
		}
		scopeOfWork.Deliverables.CoProducingDeliverable = *coProducingDeliverables
	}

	if err := helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		updatedScopeOfWorkBytes, err := json.Marshal(scopeOfWork)
		if err != nil {
			zap.L().Error("Failed to marshal scope of work", zap.Error(err))
			return err
		}
		return uow.Contracts().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", contractID)
		}, map[string]any{"scope_of_work": updatedScopeOfWorkBytes})
	}); err != nil {
		zap.L().Error("Failed to update contract scope of work", zap.String("contract_id", contractID.String()), zap.Error(err))
		return err
	}

	return nil
}

func (s *ContractService) unmarshalScopeOfWork(contract *model.Contract) (*dtos.ScopeOfWork, error) {
	var scopeOfWorks dtos.ScopeOfWork
	if err := json.Unmarshal(contract.ScopeOfWork, &scopeOfWorks); err != nil {
		return nil, err
	}
	return &scopeOfWorks, nil
}

func NewContractService(
	dbReg *gormrepository.DatabaseRegistry,
	infraReg *infrastructure.InfrastructureRegistry,
) iservice.ContractService {
	return &ContractService{
		contractRepository: dbReg.ContractRepository,
		brandRepository:    dbReg.BrandRepository,
		taskRepository:     dbReg.TaskRepository,
		unitOfWork:         infraReg.UnitOfWork,
	}
}
