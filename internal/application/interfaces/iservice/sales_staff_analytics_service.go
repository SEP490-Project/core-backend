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

	// Revenue Detail APIs - Returns orders contributing to each revenue metric
	GetTotalRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error)
	GetStandardRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error)
	GetLimitedRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error)
	//GetStandardNetRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersResponse, *responses.Pagination, error)
	GetLimitedNetRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error)
	GetRefundedOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error)
}
