package responses

// BrandRevenueResponse represents a single brand's revenue ranking
type BrandRevenueResponse struct {
	BrandID   string  `json:"brand_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BrandName string  `json:"brand_name" example:"Acme Corp"`
	Revenue   float64 `json:"revenue" example:"150000000.50"`
	Rank      int     `json:"rank" example:"1"`
}

// RevenueByTypeResponse represents revenue breakdown by contract type and products
type RevenueByTypeResponse struct {
	Advertising     float64 `json:"advertising" example:"50000000.00"`
	Affiliate       float64 `json:"affiliate" example:"30000000.00"`
	BrandAmbassador float64 `json:"brand_ambassador" example:"20000000.00"`
	CoProduce       float64 `json:"co_produce" example:"15000000.00"`
	StandardProduct float64 `json:"standard_product" example:"25000000.00"`
	TotalRevenue    float64 `json:"total_revenue" example:"140000000.00"`
}

// UpcomingCampaignResponse represents a campaign approaching deadline
type UpcomingCampaignResponse struct {
	CampaignID    string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name          string `json:"name" example:"Summer Sale 2024"`
	EndDate       string `json:"end_date" example:"2024-11-16 23:59:59"`
	DaysRemaining int    `json:"days_remaining" example:"5"`
	ContractID    string `json:"contract_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	BrandName     string `json:"brand_name" example:"Acme Corp"`
}

// MarketingDashboardResponse represents aggregated dashboard data
type MarketingDashboardResponse struct {
	// Counts
	ActiveBrands    int64 `json:"active_brands" example:"45"`
	ActiveCampaigns int64 `json:"active_campaigns" example:"12"`
	DraftCampaigns  int64 `json:"draft_campaigns" example:"5"`

	// Revenue for specified period
	// GrossRevenue = sum of all PAID payments + sum of all KOL_REFUND_APPROVED payments (before refund deduction)
	GrossRevenue float64 `json:"gross_revenue" example:"100000000.00"`
	// NetRevenue = PAID amounts + (KOL_REFUND_APPROVED amounts - refund_amount)
	NetRevenue float64 `json:"net_revenue" example:"85000000.00"`
	// TotalRefunds = sum of refund_amount from KOL_REFUND_APPROVED payments
	TotalRefunds float64 `json:"total_refunds" example:"15000000.00"`
	// Period metadata
	RevenueYear  int `json:"revenue_year" example:"2024"`
	RevenueMonth int `json:"revenue_month" example:"11"`

	// Top performers (for the same period as revenue)
	TopBrands     []BrandRevenueResponse `json:"top_brands"`
	RevenueByType RevenueByTypeResponse  `json:"revenue_by_type"`

	// Upcoming deadlines (always relative to current date)
	UpcomingDeadlines []UpcomingCampaignResponse `json:"upcoming_deadlines"`
}
