package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type PreOrderService interface {
	PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork, userID uuid.UUID) (*model.PreOrder, error)
	GetPreOrdersByUserIDWithPagination(ctx context.Context, userID uuid.UUID, limit, page int, search string, statuses []string, createdFrom, createdTo string) ([]responses.PreOrderResponse, int, error)
	PayForPreservationSlot(ctx context.Context, preOrderID uuid.UUID, returnURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error)

	// Mark Pre-Order as Received - By Customer
	MarkPreOrderAsReceived(ctx context.Context, preOrderID, updatedBy uuid.UUID) error
	//Request Compensation
	RequestCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error
	ProcessCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, isApproved bool, reason, fileURL *string) error

	// Staff-facing listing similar to staff orders
	GetStaffAvailablePreOrdersWithPagination(limit, page int, search, fullName, phone, provinceID, districtID, wardCode, createdFrom, createdTo, brandID string, status []string) ([]responses.PreOrderResponse, int, error)

	//Job to check and expire pre-orders (total count, failed count, upcomming)
	PreOrderOpeningChecker(ctx context.Context) (int, int, int)
	PreOrderOpeningManualTrigger(ctx context.Context, preOrderID, actionBy uuid.UUID) error

	RefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason *string) error

	//Staff
	ObligateRefund(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error
	ApproveRefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error

	OpeningPreOrderEarly(ctx context.Context, uow irepository.UnitOfWork, productID uuid.UUID, updatedBy uuid.UUID) error
	GetPreOrderPricePercentage(ctx context.Context, preOrderID uuid.UUID) ([]responses.PriceBreakdown, error)

	// Update GHN Order Code for a PreOrder
	UpdateGHNOrderCode(ctx context.Context, preOrderID uuid.UUID, ghnOrderCode string) error
}
