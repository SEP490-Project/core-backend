package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type PreOrderService interface {
	PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork) (*model.PreOrder, error)
	GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string, status string) ([]model.PreOrder, int, error)
	PayPreOrder(ctx context.Context, preOrderID uuid.UUID, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error)
}
