package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type OrderService interface {
	PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, unitOfWork irepository.UnitOfWork) (*model.Order, error)
	GetOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error)
	PayOrder(ctx context.Context, orderID uuid.UUID, shippingPrice int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error)

	//Staff
	GetStaffAvailableOrdersWithPagination(limit, page int, search string, status string) ([]model.Order, int, error)
}
