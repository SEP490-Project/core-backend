package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContractPaymentService interface {
	// CreateContractPaymentsFromContract creates contract payments based on the FinancialTerms details of a given contract.
	CreateContractPaymentsFromContract(
		ctx context.Context,
		userID uuid.UUID,
		contractID uuid.UUID,
		uow irepository.UnitOfWork) error

	GetContractPaymentsByFilter(ctx context.Context, filter *requests.ContractPaymentFilterRequest) (*[]responses.ContractPaymenntResponse, int64, error)
	GetContractPaymentByID(ctx context.Context, contractPaymentID uuid.UUID) (*responses.ContractPaymenntResponse, error)

	CreatePaymentLinkFromContractPayment(
		ctx context.Context,
		uow irepository.UnitOfWork,
		request *requests.GenerateContractPaymentLinkRequest,
		paymentTransactionService PaymentTransactionService) (*responses.PayOSLinkResponse, error)

	// LockPaymentAmount locks the current calculated amount when creating a payment link.
	// This prevents the amount from changing while payment is in progress.
	LockPaymentAmount(ctx context.Context, payment *model.ContractPayment) error

	// UnlockPaymentOnFailure clears the locked state when payment fails.
	UnlockPaymentOnFailure(ctx context.Context, payment *model.ContractPayment) error
}
