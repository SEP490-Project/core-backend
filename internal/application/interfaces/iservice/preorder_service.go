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
	GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error)
	PayPreOrder(ctx context.Context, orderID uuid.UUID, shippingPrice int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error)
}
