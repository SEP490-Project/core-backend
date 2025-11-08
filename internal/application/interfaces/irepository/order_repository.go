package irepository

import (
	"context"
	"core-backend/internal/domain/model"
)

// OrderRepository extends GenericRepository for Order with custom query methods
type OrderRepository interface {
	GenericRepository[model.Order]
	GetStaffAvailableOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error)
}
