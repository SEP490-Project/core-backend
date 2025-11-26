package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// SalesStaffAnalyticsService defines the interface for Sales Staff analytics operations
type SalesStaffAnalyticsService interface {
	// GetDashboard returns the complete Sales Staff dashboard with all metrics
	GetDashboard(ctx context.Context, req *requests.SalesStaffDashboardRequest) (*responses.SalesStaffDashboardResponse, error)

	// GetOrdersOverview returns orders statistics by type and status
	GetOrdersOverview(ctx context.Context, req *requests.OrdersOverviewRequest) (*responses.OrdersBreakdown, error)

	// GetPreOrdersOverview returns pre-orders statistics
	GetPreOrdersOverview(ctx context.Context, req *requests.PreOrdersOverviewRequest) (*responses.PreOrderStats, error)

	// GetRevenueBySource returns revenue breakdown by source (products, contracts)
	GetRevenueBySource(ctx context.Context, req *requests.RevenueBySourceRequest) (*responses.RevenueBySource, error)

	// GetTopBrands returns top brands by revenue
	GetTopBrands(ctx context.Context, req *requests.TopBrandsRequest) ([]responses.BrandSalesMetric, error)

	// GetTopProducts returns top products by revenue
	GetTopProducts(ctx context.Context, req *requests.TopProductsRequest) ([]responses.ProductSalesMetric, error)

	// GetRevenueTrend returns revenue time-series data
	GetRevenueTrend(ctx context.Context, req *requests.RevenueGrowthRequest) ([]responses.RevenueTrendPoint, error)

	// GetPaymentStatus returns contract payment status overview
	GetPaymentStatus(ctx context.Context, req *requests.PaymentStatusRequest) (*responses.PaymentStatusOverview, error)
}
