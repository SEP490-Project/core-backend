package responses

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// ADMIN ANALYTICS RESPONSE DTOs
// =============================================================================

// AdminDashboardResponse represents the complete Admin dashboard
type AdminDashboardResponse struct {
	Overview         AdminOverviewMetrics  `json:"overview"`
	UsersBreakdown   UsersBreakdown        `json:"users_breakdown"`
	RevenueBreakdown AdminRevenueBreakdown `json:"revenue_breakdown"`
	ContractsSummary ContractsSummary      `json:"contracts_summary"`
	CampaignsSummary AdminCampaignsSummary `json:"campaigns_summary"`
	RecentActivity   []SystemActivityItem  `json:"recent_activity"`
	GrowthTrend      []GrowthTrendPoint    `json:"growth_trend"`
	Period           PeriodInfo            `json:"period"`
}

// AdminOverviewMetrics represents platform-wide high-level metrics
type AdminOverviewMetrics struct {
	TotalUsers      int64   `json:"total_users"`
	ActiveUsers     int64   `json:"active_users"` // Users active in last 30 days
	TotalBrands     int64   `json:"total_brands"`
	ActiveBrands    int64   `json:"active_brands"` // Brands with active contracts
	TotalContracts  int64   `json:"total_contracts"`
	ActiveContracts int64   `json:"active_contracts"`
	TotalCampaigns  int64   `json:"total_campaigns"`
	ActiveCampaigns int64   `json:"active_campaigns"`
	TotalRevenue    float64 `json:"total_revenue"`
	MonthlyRevenue  float64 `json:"monthly_revenue"`
	TotalOrders     int64   `json:"total_orders"`
	MonthlyOrders   int64   `json:"monthly_orders"`
	UserGrowth      float64 `json:"user_growth"`    // Percentage change
	RevenueGrowth   float64 `json:"revenue_growth"` // Percentage change
}

// UsersBreakdown represents user statistics by role
type UsersBreakdown struct {
	Admin          int64 `json:"admin"`
	MarketingStaff int64 `json:"marketing_staff"`
	SalesStaff     int64 `json:"sales_staff"`
	ContentStaff   int64 `json:"content_staff"`
	BrandPartner   int64 `json:"brand_partner"`
	Customer       int64 `json:"customer"`
	TotalActive    int64 `json:"total_active"`   // Active in last 30 days
	NewThisMonth   int64 `json:"new_this_month"` // Registered this month
}

// AdminRevenueBreakdown represents platform-wide revenue breakdown
type AdminRevenueBreakdown struct {
	// Contract Revenue by Type
	AdvertisingRevenue float64 `json:"advertising_revenue"`
	AffiliateRevenue   float64 `json:"affiliate_revenue"`
	AmbassadorRevenue  float64 `json:"ambassador_revenue"`
	CoProducingRevenue float64 `json:"co_producing_revenue"`

	// Product Revenue
	StandardProductRevenue float64 `json:"standard_product_revenue"`
	LimitedProductRevenue  float64 `json:"limited_product_revenue"`
	ShippingRevenue        float64 `json:"shipping_revenue"`

	// Totals
	TotalContractRevenue float64 `json:"total_contract_revenue"`
	TotalProductRevenue  float64 `json:"total_product_revenue"`
	TotalRevenue         float64 `json:"total_revenue"`
}

// ContractsSummary represents contract statistics
type ContractsSummary struct {
	TotalContracts  int64   `json:"total_contracts"`
	Draft           int64   `json:"draft"`
	Approved        int64   `json:"pending"`
	Active          int64   `json:"active"`
	Completed       int64   `json:"completed"`
	Terminated      int64   `json:"cancelled"`
	TotalValue      float64 `json:"total_value"`
	CollectedAmount float64 `json:"collected_amount"` // Total paid amount
	PendingAmount   float64 `json:"pending_amount"`   // Outstanding payment amount
}

// AdminCampaignsSummary represents campaign statistics
type AdminCampaignsSummary struct {
	TotalCampaigns int64 `json:"total_campaigns"`
	Draft          int64 `json:"draft"`
	Running        int64 `json:"running"`
	Completed      int64 `json:"completed"`
	Cancelled      int64 `json:"cancelled"`
	ContentCreated int64 `json:"content_created"` // Total content pieces
	ContentPosted  int64 `json:"content_posted"`  // Content with status POSTED
}

// SystemActivityItem represents a recent system activity
type SystemActivityItem struct {
	ActivityType string    `json:"activity_type"` // e.g., "CONTRACT_CREATED", "ORDER_PLACED"
	Description  string    `json:"description"`
	EntityID     string    `json:"entity_id"`
	EntityType   string    `json:"entity_type"` // e.g., "CONTRACT", "ORDER", "USER"
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	Timestamp    time.Time `json:"timestamp"`
}

// GrowthTrendPoint represents a single point in growth time-series
type GrowthTrendPoint struct {
	Date         time.Time `json:"date"`
	NewUsers     int64     `json:"new_users"`
	NewOrders    int64     `json:"new_orders"`
	NewContracts int64     `json:"new_contracts"`
	Revenue      float64   `json:"revenue"`
}

// UsersOverviewResponse represents users overview
type UsersOverviewResponse struct {
	TotalUsers        int64             `json:"total_users"`
	ActiveUsers       int64             `json:"active_users"`
	NewUsersThisMonth int64             `json:"new_users_this_month"`
	RoleBreakdown     UsersBreakdown    `json:"role_breakdown"`
	GrowthTrend       []UserGrowthPoint `json:"growth_trend"`
	Period            PeriodInfo        `json:"period"`
}

// UserGrowthPoint represents a single point in user growth time-series
type UserGrowthPoint struct {
	Date     time.Time `json:"date"`
	NewUsers int64     `json:"new_users"`
	Total    int64     `json:"total"`
}

// PlatformRevenueResponse represents platform-wide revenue analytics
type PlatformRevenueResponse struct {
	TotalRevenue     float64               `json:"total_revenue"`
	RevenueBreakdown AdminRevenueBreakdown `json:"revenue_breakdown"`
	RevenueTrend     []RevenueTrendPoint   `json:"revenue_trend"`
	TopBrands        []BrandSalesMetric    `json:"top_brands"`
	Period           PeriodInfo            `json:"period"`
}

// SystemHealthResponse represents system health metrics
type SystemHealthResponse struct {
	DatabaseStatus    string  `json:"database_status"` // OK, WARNING, ERROR
	CacheStatus       string  `json:"cache_status"`    // OK, WARNING, ERROR
	QueueStatus       string  `json:"queue_status"`    // OK, WARNING, ERROR
	PendingJobs       int64   `json:"pending_jobs"`    // Jobs in queue
	FailedJobs24h     int64   `json:"failed_jobs_24h"` // Failed jobs in last 24h
	AverageResponseMs int64   `json:"average_response_ms"`
	ErrorRate         float64 `json:"error_rate"` // Percentage in last hour
	Uptime            string  `json:"uptime"`
}

// RevenueTrendPoint represents a single point in revenue time-series
type RevenueTrendPoint struct {
	Date              time.Time `json:"date"`
	Revenue           float64   `json:"revenue"`
	OrderCount        int64     `json:"order_count"`
	AverageOrderValue float64   `json:"average_order_value"`
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
