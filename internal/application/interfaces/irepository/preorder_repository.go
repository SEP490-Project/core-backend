package irepository

import (
	"context"
	"core-backend/internal/domain/model"
)

// PreOrderRepository extends GenericRepository for PreOrder with custom query methods
type PreOrderRepository interface {
	GenericRepository[model.PreOrder]
	GetStaffAvailablePreOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.PreOrder, int, error)
}
