package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type OrderService interface {
	// Atomic operation to place an order with payment
	PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, isOrderLimited bool, unitOfWork irepository.UnitOfWork) (*model.Order, error)

	GetOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error)
	PayOrder(ctx context.Context, orderID uuid.UUID, shippingPrice int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error)
	MarkAsReceived(ctx context.Context, orderID uuid.UUID) error

	//Staff
	GetStaffAvailableOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string, orderType string) ([]model.Order, int, error)
	MarkAsReadyToPickedUp(ctx context.Context, orderID uuid.UUID) error
	MarkAsReceivedAfterPickedUp(ctx context.Context, orderID uuid.UUID, imageUrl string) error

	//internal delivery service - type = limited & self-delivering = false
	GetSelfDeliveringOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error)
	MarkSelfDeliveringOrderAsInTransit(ctx context.Context, orderID uuid.UUID) error
	MarkSelfDeliveringOrderAsDelivered(ctx context.Context, orderID uuid.UUID, imageUrl string) error
}
