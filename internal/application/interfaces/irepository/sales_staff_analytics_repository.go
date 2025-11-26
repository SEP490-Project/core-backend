package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"time"

	"github.com/google/uuid"
)

// SalesStaffAnalyticsRepository defines the interface for sales staff analytics data access
type SalesStaffAnalyticsRepository interface {
	// Orders
	GetOrdersCountByType(ctx context.Context, orderType string, startDate, endDate *time.Time) (int64, error)
	GetOrdersRevenueByType(ctx context.Context, orderType string, startDate, endDate *time.Time) (float64, error)
	GetOrdersCountByStatus(ctx context.Context, orderType, status string, startDate, endDate *time.Time) (int64, error)

	// PreOrders
	GetPreOrdersCount(ctx context.Context, startDate, endDate *time.Time) (int64, error)
	GetPreOrdersRevenue(ctx context.Context, startDate, endDate *time.Time) (float64, error)
	GetPreOrdersCountByStatus(ctx context.Context, status string, startDate, endDate *time.Time) (int64, error)

	// Revenue by Contract Type
	GetContractRevenueByType(ctx context.Context, contractType string, startDate, endDate *time.Time) (float64, error)

	// Top Performers
	GetTopBrandsByRevenue(ctx context.Context, limit int, startDate, endDate *time.Time) ([]dtos.BrandRevenueResult, error)
	GetTopProductsByRevenue(ctx context.Context, productType string, limit int, startDate, endDate *time.Time) ([]dtos.ProductRevenueResult, error)

	// Trends
	GetRevenueTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.RevenueTrendResult, error)

	// Recent Activity
	GetRecentOrders(ctx context.Context, limit int) ([]dtos.RecentOrderResult, error)

	// Payment Status
	GetPaymentStatusCounts(ctx context.Context, contractID *uuid.UUID, startDate, endDate *time.Time) (*dtos.PaymentStatusResult, error)
}
