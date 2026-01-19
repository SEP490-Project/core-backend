package irepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type PaymentTransactionRepository interface {
	GenericRepository[model.PaymentTransaction]
	GetPaymentTransactionByFilter(ctx context.Context, filter *requests.PaymentTransactionFilterRequest) ([]responses.PaymentTransactionResponse, int64, error)

	GetPaymentTransactionByID(ctx context.Context, ID uuid.UUID) (*responses.PaymentTransactionResponse, error)

	GetPaymentTransactionByOrderCode(ctx context.Context, orderCode string) (*model.PaymentTransaction, error)
}
