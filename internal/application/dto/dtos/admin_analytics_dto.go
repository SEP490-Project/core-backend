package dtos

import "time"

// UserGrowthResult represents user growth query result
type UserGrowthResult struct {
	Date     time.Time
	NewUsers int64
	Total    int64
}

// GrowthTrendResult represents growth trend query result
type GrowthTrendResult struct {
	Date         time.Time
	NewUsers     int64
	NewOrders    int64
	NewContracts int64
	Revenue      float64
}

// =============================================================================
// CONSOLIDATED DASHBOARD DTOs (for batch queries)
// =============================================================================

// DashboardUsersResult represents consolidated users metrics from a single query
type DashboardUsersResult struct {
	TotalUsers     int64
	ActiveUsers    int64
	Admin          int64
	MarketingStaff int64
	SalesStaff     int64
	ContentStaff   int64
	BrandPartner   int64
	Customer       int64
	NewThisMonth   int64
}

// DashboardContractsResult represents consolidated contracts metrics from a single query
type DashboardContractsResult struct {
	TotalContracts  int64
	Draft           int64
	Approved        int64
	Active          int64
	Completed       int64
	Terminated      int64
	TotalValue      float64
	CollectedAmount float64
	PendingAmount   float64
}

// RevenueTrendResult represents revenue trend query result
type RevenueTrendResult struct {
	Date              time.Time
	Revenue           float64
	OrderCount        int64
	AverageOrderValue float64
}

// DashboardCampaignsResult represents consolidated campaigns metrics from a single query
type DashboardCampaignsResult struct {
	TotalCampaigns int64
	Draft          int64
	Running        int64
	Completed      int64
	Cancelled      int64
	ContentCreated int64
	ContentPosted  int64
}

// DashboardBrandsResult represents consolidated brands metrics
type DashboardBrandsResult struct {
	TotalBrands  int64
	ActiveBrands int64
}

// DashboardOrdersResult represents consolidated orders metrics
type DashboardOrdersResult struct {
	TotalOrders   int64
	MonthlyOrders int64
}

// DashboardRevenueResult represents consolidated revenue metrics from a single query
type DashboardRevenueResult struct {
	TotalRevenue           float64
	MonthlyRevenue         float64
	AdvertisingRevenue     float64
	AffiliateRevenue       float64
	AmbassadorRevenue      float64
	CoProducingRevenue     float64
	StandardProductRevenue float64
	LimitedProductRevenue  float64
	ShippingRevenue        float64
}
