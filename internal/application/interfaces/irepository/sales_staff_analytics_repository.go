package irepository

import (
	"context"
	"time"

	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"
)

type SalesStaffAnalyticsRepository interface {
	// Financials Tab
	// Row 1: Summary Metrics (Revenue, Growth, AOV, Conversion, Returning Customers)
	GetFinancialsSummary(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (*responses.FinancialsSummary, error)

	GetTotalSoldRevenue(ctx context.Context, from, to time.Time, orderStatuses []enum.OrderStatus, preOrderStatuses []enum.PreOrderStatus) (float64, error)

	// Row 2: Revenue Breakdown (By Product Type, By Category)
	GetRevenueBreakdown(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (byProduct []responses.RevenueByProductType, byCategory []responses.RevenueByCategory, err error)

	// Row 3: Revenue Trend
	GetRevenueTrend(ctx context.Context, from, to time.Time, periodGap string, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (map[string][]responses.SalesTimeSeriesPoint, error)

	// Row 4: Top Selling by Revenue
	GetTopSellingByRevenue(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int, sortBy, sortOrder string) (products, categories, brands []responses.TopEntity, err error)

	// =============================================================================
	// REVENUE DETAIL QUERIES - Returns orders contributing to each revenue metric
	// =============================================================================

	// GetTotalRevenueOrders returns all orders (both standard and limited) and preorders contributing to total revenue
	// Includes payment transaction information
	GetTotalRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error)

	// GetStandardRevenueOrders returns only STANDARD type orders
	// Includes payment transaction information
	GetStandardRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error)

	// GetLimitedRevenueOrders returns LIMITED type orders and all PreOrders
	// Includes payment transaction information
	GetLimitedRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error)

	// GetStandardNetRevenueOrders returns STANDARD orders with net revenue (total_amount - shipping_fee)
	GetStandardNetRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItem, int64, float64, error)

	// GetLimitedNetRevenueOrders returns LIMITED orders and PreOrders with KOL net revenue calculation
	// KOL Net Revenue = (item_total) * kol_percent / 100 - shipping_fee
	// Includes payment transaction information
	GetLimitedNetRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error)

	// GetRefundedOrders returns all refunded orders and preorders with payment transaction information
	GetRefundedOrders(ctx context.Context, from, to time.Time, refundedOrderStatuses []enum.OrderStatus, refundedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error)

	// Orders Tab
	// Row 1: Summary Metrics (Counts, Rates)
	// Note: Rates require checking Cancelled/Refunded statuses vs Total
	GetOrdersSummary(ctx context.Context, from, to time.Time) (*responses.OrdersSummary, error)

	// Row 2: Status Distribution (Pie Charts)
	GetOrderStatusDistribution(ctx context.Context, from, to time.Time) (orders, preOrders responses.OrderStatusDistribution, err error)

	// Row 3: Orders Trend
	GetOrdersTrend(ctx context.Context, from, to time.Time, periodGap string) (orders, preOrders, standard, limited []responses.SalesTimeSeriesPoint, err error)

	// Row 4: Top Selling by Volume (Count)
	GetTopSellingByVolume(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int, sortBy, sortOrder string) (products, categories, brands []responses.TopEntity, err error)

	// Row 5: Latest Orders
	GetLatestOrders(ctx context.Context, from, to time.Time, limit int) ([]responses.LatestOrder, error)

	// Helper
	GetFirstOrderDate(ctx context.Context) (*time.Time, error)
}
