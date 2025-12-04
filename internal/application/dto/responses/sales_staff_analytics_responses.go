package responses

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// FINANCIALS DASHBOARD
// =============================================================================

type FinancialsDashboardResponse struct {
	Summary           FinancialsSummary      `json:"summary"`
	RevenueByProduct  []RevenueByProductType `json:"revenue_by_product_type"` // Pie Chart
	RevenueByCategory []RevenueByCategory    `json:"revenue_by_category"`     // Column Chart
	RevenueTrend      RevenueTrendCharts     `json:"revenue_trend"`           // Line Charts
	TopLists          FinancialsTopLists     `json:"top_lists"`
}

type FinancialsSummary struct {
	TotalSoldRevenue             float64    `json:"total_sold_revenue"`
	RevenueGrowth                float64    `json:"revenue_growth"` // Compared to previous period
	AverageOrderValue            AOVMetrics `json:"average_order_value"`
	LimitedProductConversionRate float64    `json:"limited_product_conversion_rate"`
	ReturningCustomerCount       int64      `json:"returning_customer_count"`
}

type AOVMetrics struct {
	Combined  float64 `json:"combined"`
	Orders    float64 `json:"orders"`
	PreOrders float64 `json:"pre_orders"`
}

type RevenueByProductType struct {
	ProductType string  `json:"product_type"` // STANDARD, LIMITED
	Revenue     float64 `json:"revenue"`
	Percentage  float64 `json:"percentage"`
}

type RevenueByCategory struct {
	CategoryName string  `json:"category_name"`
	Revenue      float64 `json:"revenue"`
}

type RevenueTrendCharts struct {
	OrdersVsPreOrders map[string][]SalesTimeSeriesPoint `json:"orders_vs_pre_orders"`
	StandardVsLimited map[string][]SalesTimeSeriesPoint `json:"standard_vs_limited"`
}

type FinancialsTopLists struct {
	TopProducts   []TopEntity `json:"top_products"`
	TopCategories []TopEntity `json:"top_categories"`
	TopBrands     []TopEntity `json:"top_brands"`
}

// =============================================================================
// ORDERS DASHBOARD
// =============================================================================

type OrdersDashboardResponse struct {
	Summary           OrdersSummary           `json:"summary"`
	OrdersPieChart    OrderStatusDistribution `json:"orders_pie_chart"`
	PreOrdersPieChart OrderStatusDistribution `json:"pre_orders_pie_chart"`
	OrdersTrend       OrdersTrendCharts       `json:"orders_trend"`
	TopLists          OrdersTopLists          `json:"top_lists"`
	LatestOrders      []LatestOrder           `json:"latest_orders"`
}

type OrdersSummary struct {
	TotalOrders      int64   `json:"total_orders"`
	TotalPreOrders   int64   `json:"total_pre_orders"`
	CancellationRate float64 `json:"cancellation_rate"`
	RefundRate       float64 `json:"refund_rate"`
}

type OrderStatusDistribution struct {
	Pending   int64 `json:"pending"`
	Completed int64 `json:"completed"`
	Cancelled int64 `json:"cancelled"`
	Refunded  int64 `json:"refunded"`
}

type OrdersTrendCharts struct {
	OrdersVsPreOrders map[string][]SalesTimeSeriesPoint `json:"orders_vs_pre_orders"`
	StandardVsLimited map[string][]SalesTimeSeriesPoint `json:"standard_vs_limited"`
}

type OrdersTopLists struct {
	TopProducts   []TopEntity `json:"top_products"`
	TopCategories []TopEntity `json:"top_categories"`
	TopBrands     []TopEntity `json:"top_brands"`
}

// =============================================================================
// SHARED TYPES
// =============================================================================

type SalesTimeSeriesPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
	Type  string    `json:"type,omitempty"` // e.g., "ORDER", "PRE_ORDER", "STANDARD", "LIMITED"
}

type TopEntity struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Value float64   `json:"value"` // Revenue or Count
}

type LatestOrder struct {
	ID           uuid.UUID `json:"id"`
	CustomerName string    `json:"customer_name"`
	TotalAmount  float64   `json:"total_amount"`
	Status       string    `json:"status"`
	Type         string    `json:"type"` // ORDER or PRE_ORDER
	CreatedAt    time.Time `json:"created_at"`
}
