package irepository

import (
	"context"
	"core-backend/internal/domain/model"
)

// OrderRepository extends GenericRepository for Order with custom query methods
type OrderRepository interface {
	GenericRepository[model.Order]
	GetStaffAvailableOrdersWithPagination(ctx context.Context, limit, page int, search, fullName, phone, provinceID, districtID, wardCode, orderType string, statuses []string) ([]model.Order, int, error)
	GetSelfDeliveryOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error)
	// GetOrdersWithFiltersWithPagination allows searching by GHN order code or Order ID,
	// filter by created date range (createdFrom, createdTo as YYYY-MM-DD) and by status.
	GetOrdersWithFiltersWithPagination(ctx context.Context, limit, page int, search, status, createdFrom, createdTo, fullName, phone, provinceID, districtID, wardCode, orderType string) ([]model.Order, int, error)
}
