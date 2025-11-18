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
	PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork) (*model.PreOrder, error)
	GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string, status []string) ([]responses.PreOrderResponse, int, error)
	PayForPreservationSlot(ctx context.Context, preOrderID uuid.UUID, returnURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error)

	// Staff-facing listing similar to staff orders
	GetStaffAvailablePreOrdersWithPagination(limit, page int, search, fullName, phone, provinceID, districtID, wardCode string, status []string) ([]responses.PreOrderResponse, int, error)
}
