package iservice

import (
	"context"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// SalesStaffAnalyticsService defines the interface for Sales Staff analytics operations
type SalesStaffAnalyticsService interface {
	GetFinancialsDashboard(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.FinancialsDashboardResponse, error)
	GetOrdersDashboard(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.OrdersDashboardResponse, error)

	// Specific Card APIs
	GetRevenueTrend(ctx context.Context, req *requests.SalesDashboardFilter) (map[string][]responses.SalesTimeSeriesPoint, error)
	GetOrdersTrend(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.OrdersTrendCharts, error)
	GetRevenueGrowth(ctx context.Context, req *requests.SalesDashboardFilter) (float64, error)
}
