package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type ContractService interface {
	// CreateContract creates a new contract and optionally updates brand information
	CreateContract(ctx context.Context,
		userID uuid.UUID,
		createRequest *requests.CreateContractRequest,
		unitOfWork irepository.UnitOfWork,
	) (*responses.ContractResponse, error)

	ApproveContract(ctx context.Context, contractID uuid.UUID) error

	// UpdateContract updates an existing contract and optionally updates brand information
	UpdateContract(ctx context.Context,
		contractID uuid.UUID,
		updateRequest *requests.UpdateContractRequest,
		unitOfWork irepository.UnitOfWork,
	) (*responses.ContractResponse, error)

	// UpdateContractFileURL updates the file URL of a contract
	UpdateContractFileURL(ctx context.Context, contractID uuid.UUID, fileURL string, uow irepository.UnitOfWork) error

	// GetContractByID retrieves a contract by its ID with related data
	GetContractByID(ctx context.Context, contractID uuid.UUID) (*responses.ContractResponse, error)

	// GetContractsByBrandID retrieves all contracts for a specific brand
	GetContractsByBrandID(ctx context.Context, brandID uuid.UUID, page, limit int) ([]*responses.ContractListResponse, int64, error)

	// GetByFilter retrieves contracts based on filter criteria
	GetByFilter(ctx context.Context, filter *requests.ContractFilterRequest) ([]*responses.ContractListResponse, int64, error)

	// DeleteContractByID soft deletes a contract
	DeleteContractByID(ctx context.Context, contractID uuid.UUID) error
}
