package responses

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// SALES STAFF ANALYTICS RESPONSE DTOs
// =============================================================================

// SalesStaffDashboardResponse represents the complete Sales Staff dashboard
type SalesStaffDashboardResponse struct {
	Overview        SalesOverviewMetrics `json:"overview"`
	OrdersBreakdown OrdersBreakdown      `json:"orders_breakdown"`
	RevenueBySource RevenueBySource      `json:"revenue_by_source"`
	TopBrands       []BrandSalesMetric   `json:"top_brands"`
	TopProducts     []ProductSalesMetric `json:"top_products"`
	RecentOrders    []RecentOrderItem    `json:"recent_orders"`
	RevenueTrend    []RevenueTrendPoint  `json:"revenue_trend"`
	Period          PeriodInfo           `json:"period"`
}

// SalesOverviewMetrics represents high-level sales metrics
type SalesOverviewMetrics struct {
	TotalRevenue      float64 `json:"total_revenue"`       // Total revenue from all sources
	OrderRevenue      float64 `json:"order_revenue"`       // Revenue from orders only
	ContractRevenue   float64 `json:"contract_revenue"`    // Revenue from contract payments
	TotalOrders       int64   `json:"total_orders"`        // Total order count
	TotalPreOrders    int64   `json:"total_pre_orders"`    // Total pre-order count
	CompletedOrders   int64   `json:"completed_orders"`    // Orders with status RECEIVED
	PendingOrders     int64   `json:"pending_orders"`      // Orders pending fulfillment
	AverageOrderValue float64 `json:"average_order_value"` // Average order value
	RevenueGrowth     float64 `json:"revenue_growth"`      // Percentage change from previous period
	OrderGrowth       float64 `json:"order_growth"`        // Percentage change in order count
}

// OrdersBreakdown represents order statistics by type and status
type OrdersBreakdown struct {
	StandardOrders OrderTypeStats `json:"standard_orders"` // Order type = STANDARD
	LimitedOrders  OrderTypeStats `json:"limited_orders"`  // Order type = LIMITED
	PreOrders      PreOrderStats  `json:"pre_orders"`      // From pre_orders table
}

// OrderTypeStats represents statistics for a specific order type
type OrderTypeStats struct {
	TotalCount     int64   `json:"total_count"`
	TotalRevenue   float64 `json:"total_revenue"`
	CompletedCount int64   `json:"completed_count"` // Status = RECEIVED
	PendingCount   int64   `json:"pending_count"`   // Status in progress
	CancelledCount int64   `json:"cancelled_count"` // Status = CANCELLED
}

// PreOrderStats represents pre-order statistics
type PreOrderStats struct {
	TotalCount     int64   `json:"total_count"`
	TotalRevenue   float64 `json:"total_revenue"`
	ReceivedCount  int64   `json:"received_count"` // Status = RECEIVED
	PendingCount   int64   `json:"pending_count"`
	CancelledCount int64   `json:"cancelled_count"`
}

// RevenueBySource represents revenue breakdown by source
type RevenueBySource struct {
	StandardProductRevenue float64 `json:"standard_product_revenue"` // Orders with order_type = STANDARD
	LimitedProductRevenue  float64 `json:"limited_product_revenue"`  // Orders with order_type = LIMITED + pre_orders
	AdvertisingRevenue     float64 `json:"advertising_revenue"`      // Contract payments for ADVERTISING
	AffiliateRevenue       float64 `json:"affiliate_revenue"`        // Contract payments for AFFILIATE
	AmbassadorRevenue      float64 `json:"ambassador_revenue"`       // Contract payments for AMBASSADOR
	CoProducingRevenue     float64 `json:"co_producing_revenue"`     // Contract payments for CO_PRODUCING
	TotalRevenue           float64 `json:"total_revenue"`
}

// BrandSalesMetric represents sales metrics for a brand
type BrandSalesMetric struct {
	BrandID      uuid.UUID `json:"brand_id"`
	BrandName    string    `json:"brand_name"`
	TotalRevenue float64   `json:"total_revenue"`
	OrderCount   int64     `json:"order_count"`
	ProductCount int       `json:"product_count"` // Number of products sold
	Rank         int       `json:"rank"`
}

// ProductSalesMetric represents sales metrics for a product
type ProductSalesMetric struct {
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	BrandName    string    `json:"brand_name"`
	ProductType  string    `json:"product_type"` // STANDARD or LIMITED
	TotalRevenue float64   `json:"total_revenue"`
	UnitsSold    int64     `json:"units_sold"`
	Rank         int       `json:"rank"`
}

// RecentOrderItem represents a recent order
type RecentOrderItem struct {
	OrderID      uuid.UUID `json:"order_id"`
	CustomerName string    `json:"customer_name"`
	TotalAmount  float64   `json:"total_amount"`
	Status       string    `json:"status"`
	OrderType    string    `json:"order_type"` // STANDARD or LIMITED
	ItemCount    int       `json:"item_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// RevenueTrendPoint represents a single point in revenue time-series
type RevenueTrendPoint struct {
	Date              time.Time `json:"date"`
	Revenue           float64   `json:"revenue"`
	OrderCount        int64     `json:"order_count"`
	AverageOrderValue float64   `json:"average_order_value"`
}

// PaymentStatusOverview represents contract payment status counts
type PaymentStatusOverview struct {
	TotalPayments   int64   `json:"total_payments"`
	PaidPayments    int64   `json:"paid_payments"`
	PendingPayments int64   `json:"pending_payments"`
	OverduePayments int64   `json:"overdue_payments"`
	TotalAmount     float64 `json:"total_amount"`
	PaidAmount      float64 `json:"paid_amount"`
	PendingAmount   float64 `json:"pending_amount"`
	OverdueAmount   float64 `json:"overdue_amount"`
}
