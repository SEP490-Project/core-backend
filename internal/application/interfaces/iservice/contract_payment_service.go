package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
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

	GetContractPaymentsByFilter(ctx context.Context, filter *requests.ContractPaymentFilterRequest) (*[]responses.ContractPaymenntResponse, int64, error)
	GetContractPaymentByID(ctx context.Context, contractPaymentID uuid.UUID) (*responses.ContractPaymenntResponse, error)

	CreatePaymentLinkFromContractPayment(
		ctx context.Context,
		uow irepository.UnitOfWork,
		contractPaymentID uuid.UUID,
		paymentTransactionService PaymentTransactionService) (*responses.PayOSLinkResponse, error)
}
