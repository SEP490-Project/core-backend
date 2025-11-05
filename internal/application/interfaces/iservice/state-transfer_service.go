package iservice

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type StateTransferService interface {
	MoveTaskToState(ctx context.Context, taskID uuid.UUID, targetState enum.TaskStatus, updatedBy uuid.UUID) error
	MoveProductToState(ctx context.Context, productID uuid.UUID, targetState enum.ProductStatus, updatedBy uuid.UUID) error
	MoveMileStoneToState(ctx context.Context, mileStoneID uuid.UUID, targetState enum.MilestoneStatus, updatedBy uuid.UUID) error

	// MoveCampaignToState moves a campaign to a new state
	// Parameters:
	// - ctx: context for the operation
	// - uow: unit of work for transactional operations
	// - campaignID: ID of the campaign to move
	// - targetState: desired state to move the campaign to
	// - updatedBy: ID of the user performing the operation
	//
	// Returns:
	// - error: error if the operation fails, nil otherwise
	MoveCampaignToState(
		ctx context.Context,
		uow irepository.UnitOfWork,
		campaignID uuid.UUID,
		targetState enum.CampaignStatus,
		updatedBy uuid.UUID,
	) error

	MoveContractToState(ctx context.Context, uow irepository.UnitOfWork, contractID uuid.UUID, targetState enum.ContractStatus, updatedBy uuid.UUID) error
	MoveContentToState(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, targetState enum.ContentStatus, updatedBy uuid.UUID) error
	MovePaymentTransactionToState(ctx context.Context, uow irepository.UnitOfWork, transactionID uuid.UUID, targetState enum.PaymentTransactionStatus, updatedBy uuid.UUID) error
}
