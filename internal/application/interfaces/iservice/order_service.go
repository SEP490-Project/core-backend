package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type OrderService interface {
	// Atomic operation to place an order with payment
	PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, isOrderLimited bool, unitOfWork irepository.UnitOfWork) (*model.Order, error)

	// Get orders by user with optional filtering by search (GHN order code or order ID), status and created date range
	GetOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search, status, createdFrom, createdTo string) ([]responses.OrderResponse, int, error)
	PayOrder(ctx context.Context, orderID uuid.UUID, shippingPrice int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error)
	MarkAsReceived(ctx context.Context, orderID, userID uuid.UUID) error

	//Staff
	GetStaffAvailableOrdersWithPagination(limit, page int, search, fullName, phone, provinceID, districtID, wardCode string, orderType string, statuses []string) ([]responses.OrderResponse, int, error)
	MarkAsReadyToPickedUp(ctx context.Context, orderID, userID uuid.UUID) error
	MarkAsReceivedAfterPickedUp(ctx context.Context, orderID, userID uuid.UUID, imageUrl string) error

	//internal delivery service - type = limited & self-delivering = false
	GetSelfDeliveringOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error)
	MarkSelfDeliveringOrderAsInTransit(ctx context.Context, orderID, userID uuid.UUID) error
	MarkSelfDeliveringOrderAsDelivered(ctx context.Context, orderID, userID uuid.UUID, imageUrl string) error

	//Request Refund - By Customer
	RequestEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, requestTime time.Time) error
	//Process Refund - By Staff & Cancelled
	ApproveEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, fileURL string) error
	ObligateEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, reason, fileURL *string) error

	//Request Compensation - By Customer
	RequestCompensation(ctx context.Context, orderID, actionBy uuid.UUID, reason, fileURL *string) error
	ProcessCompensation(ctx context.Context, orderID, actionBy uuid.UUID, isApproved bool, reason, fileURL *string) error

	GetOrderPricePercentage(ctx context.Context, orderID uuid.UUID, orderType string) ([]responses.PriceBreakdown, error)
}
