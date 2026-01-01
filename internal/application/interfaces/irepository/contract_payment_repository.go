package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContractPaymentRepository interface {
	GenericRepository[model.ContractPayment]

	GetNextUnpaidContractPaymentFromCurrentPaymentID(ctx context.Context, currentPaymentID uuid.UUID) (*model.ContractPayment, error)
}
