package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContractService struct {
	contractRepository irepository.GenericRepository[model.Contract]
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
	createRequest *requests.CreateContractRequest,
	unitOfWork irepository.UnitOfWork,
) (*responses.ContractResponse, error) {
	zap.L().Info("Creating new contract",
		zap.String("brand_id", createRequest.BrandID),
		zap.String("title", createRequest.Title))

	// Convert request to contract model
	contract, err := createRequest.ToContract()
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
		unitOfWork.Rollback()
		zap.L().Error("Failed to check contract number uniqueness", zap.Error(err))
		return nil, errors.New("failed to validate contract number")
	}
	if exists {
		unitOfWork.Rollback()
		zap.L().Warn("Contract number already exists", zap.String("contract_number", *contract.ContractNumber))
		return nil, fmt.Errorf("contract number %s already exists", *contract.ContractNumber)
	}

	// Create contract
	if err = contractRepo.Add(ctx, contract); err != nil {
		unitOfWork.Rollback()
		zap.L().Error("Failed to create contract", zap.Error(err))
		return nil, errors.New("failed to create contract")
	}

	// Commit transaction
	if err = unitOfWork.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		return nil, errors.New("failed to save contract")
	}

	// Retrieve created contract with relationships
	createdContract, err := s.contractRepository.GetByID(ctx, contract.ID, []string{"Brand", "ParentContract"})
	if err != nil {
		zap.L().Error("Failed to retrieve created contract", zap.Error(err))
		return nil, errors.New("contract created but failed to retrieve details")
	}

	zap.L().Info("Contract created successfully",
		zap.String("contract_id", contract.ID.String()),
		zap.String("contract_number", *contract.ContractNumber))

	return responses.ToContractResponse(createdContract)
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
	uow := unitOfWork.Begin()
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

	return responses.ToContractResponse(updatedContract)
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

	contract, err := s.contractRepository.GetByID(ctx, contractID, []string{"Brand", "ParentContract"})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
			return nil, errors.New("contract not found")
		}
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return nil, errors.New("failed to fetch contract")
	}

	return responses.ToContractResponse(contract)
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

	contracts, total, err := s.contractRepository.GetAll(ctx, filter, []string{"Brand"}, limit, page)
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
			db = db.Where("start_date >= ?", filterReq.StartDate)
		}
		if filterReq.EndDate != nil {
			db = db.Where("end_date <= ?", filterReq.EndDate)
		}

		// Sorting
		sortBy := filterReq.SortBy
		if sortBy == "" {
			sortBy = "created_at"
		}
		sortOrder := filterReq.SortOrder
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc"
		}
		db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

		return db
	}

	page := max(filterReq.Page, 1)
	limit := filterReq.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	contracts, total, err := s.contractRepository.GetAll(ctx, filter, []string{"Brand"}, limit, page)
	if err != nil {
		zap.L().Error("Failed to fetch contracts", zap.Error(err))
		return nil, 0, errors.New("failed to fetch contracts")
	}

	result := make([]*responses.ContractListResponse, 0, len(contracts))
	for _, contract := range contracts {
		result = append(result, responses.ToContractListResponse(&contract))
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

func NewContractService(contractRepository irepository.GenericRepository[model.Contract]) iservice.ContractService {
	return &ContractService{
		contractRepository: contractRepository,
	}
}
