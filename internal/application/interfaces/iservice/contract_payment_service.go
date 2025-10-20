package iservice

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type ContractPaymentService interface {
	// CreateContractPaymentsFromContract creates contract payments based on the FinancialTerms details of a given contract.
	CreateContractPaymentsFromContract(
		ctx context.Context,
		userID uuid.UUID,
		contractID uuid.UUID,
		uow irepository.UnitOfWork) error

	// GetAllByContractID(ctx context.Context, contractID uuid.UUID) ([]*irepository.ContractPaymentWithStatus, error)

	// UpdateContractPaymentStatus(ctx context.Context, requests.UpdateContractPaymentStatusRequest) error
}
