package irepository

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// PreOrderRepository extends GenericRepository for PreOrder with custom query methods
type PreOrderRepository interface {
	GenericRepository[model.PreOrder]
	GetStaffAvailablePreOrdersWithPagination(ctx context.Context, limit, page int, search, fullName, phone, provinceID, districtID, wardCode, createdFrom, createdTo, brandID string, statuses []string) ([]model.PreOrder, int, error)
	GetPreOrderCountsAndTotalAmountByStatuses(ctx context.Context, statuses []enum.PreOrderStatus, productIDs []uuid.UUID) (count int64, totalAmount float64, err error)
}
