package responses

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// BRAND PARTNER ANALYTICS RESPONSE DTOs
// =============================================================================

// BrandPartnerDashboardResponse represents the complete Brand Partner dashboard
type BrandPartnerDashboardResponse struct {
	Overview         BrandOverviewMetrics     `json:"overview"`
	TopProducts      []BrandProductMetric     `json:"top_products"`
	Campaigns        []BrandCampaignMetric    `json:"campaigns"`
	ContentMetrics   BrandContentMetric       `json:"content_metrics"`
	RevenueTrend     []BrandRevenueTrendPoint `json:"revenue_trend"`
	AffiliateMetrics BrandAffiliateMetric     `json:"affiliate_metrics"`
	Contracts        []BrandContractDetail    `json:"contracts"`
	Period           PeriodInfo               `json:"period"`
}

// BrandOverviewMetrics represents high-level metrics for the brand
type BrandOverviewMetrics struct {
	TotalContracts  int64   `json:"total_contracts"`
	ActiveContracts int64   `json:"active_contracts"`
	TotalCampaigns  int64   `json:"total_campaigns"`
	ActiveCampaigns int64   `json:"active_campaigns"`
	TotalProducts   int64   `json:"total_products"`
	TotalOrders     int64   `json:"total_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
	PendingPayments float64 `json:"pending_payments"`
}

// BrandContractsBreakdown represents contract statistics by type
type BrandContractsBreakdown struct {
	Advertising BrandContractTypeStats `json:"advertising"`
	Affiliate   BrandContractTypeStats `json:"affiliate"`
	Ambassador  BrandContractTypeStats `json:"ambassador"`
	CoProduce   BrandContractTypeStats `json:"co_produce"`
}

// BrandContractTypeStats represents statistics for a contract type
type BrandContractTypeStats struct {
	Count         int64   `json:"count"`
	ActiveCount   int64   `json:"active_count"`
	TotalValue    float64 `json:"total_value"`    // Total contract value
	PaidAmount    float64 `json:"paid_amount"`    // Amount already paid
	PendingAmount float64 `json:"pending_amount"` // Amount pending payment
}

// BrandRevenueBySource represents brand's revenue breakdown
type BrandRevenueBySource struct {
	StandardProducts float64 `json:"standard_products"` // Revenue from standard product sales
	LimitedProducts  float64 `json:"limited_products"`  // Revenue from limited product sales
	AffiliateClicks  float64 `json:"affiliate_clicks"`  // Revenue from affiliate clicks
	TotalRevenue     float64 `json:"total_revenue"`
}

// BrandCampaignsSummary represents campaign statistics for the brand
type BrandCampaignsSummary struct {
	TotalCampaigns int64               `json:"total_campaigns"`
	Draft          int64               `json:"draft"`
	Active         int64               `json:"active"`
	InProgress     int64               `json:"in_progress"`
	Finished       int64               `json:"finished"`
	Cancelled      int64               `json:"cancelled"`
	TopCampaigns   []BrandCampaignItem `json:"top_campaigns"` // Top 5 by performance
}

// BrandCampaignItem represents a campaign item for the brand
type BrandCampaignItem struct {
	CampaignID   uuid.UUID  `json:"campaign_id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	StartDate    *time.Time `json:"start_date"`
	EndDate      *time.Time `json:"end_date"`
	ContentCount int        `json:"content_count"`
	TotalViews   int64      `json:"total_views"`
	TotalClicks  int64      `json:"total_clicks"`
}

// UpcomingPaymentItem represents an upcoming payment for the brand
type UpcomingPaymentItem struct {
	PaymentID     uuid.UUID `json:"payment_id"`
	ContractID    uuid.UUID `json:"contract_id"`
	ContractName  string    `json:"contract_name"`
	Amount        float64   `json:"amount"`
	DueDate       time.Time `json:"due_date"`
	DaysRemaining int       `json:"days_remaining"`
	Status        string    `json:"status"`
}

// UpcomingMilestoneItem represents an upcoming milestone for the brand
type UpcomingMilestoneItem struct {
	MilestoneID   uuid.UUID `json:"milestone_id"`
	CampaignID    uuid.UUID `json:"campaign_id"`
	CampaignName  string    `json:"campaign_name"`
	MilestoneName string    `json:"milestone_name"`
	DueDate       time.Time `json:"due_date"`
	DaysRemaining int       `json:"days_remaining"`
	Status        string    `json:"status"`
}

// BrandContentItem represents content performance for the brand
type BrandContentItem struct {
	ContentID      uuid.UUID  `json:"content_id"`
	Title          string     `json:"title"`
	Platform       string     `json:"platform"`
	CampaignName   string     `json:"campaign_name"`
	Views          int64      `json:"views"`
	Likes          int64      `json:"likes"`
	Comments       int64      `json:"comments"`
	Shares         int64      `json:"shares"`
	Clicks         int64      `json:"clicks"`
	EngagementRate float64    `json:"engagement_rate"`
	PostedAt       *time.Time `json:"posted_at"`
	Rank           int        `json:"rank"`
}

// BrandAffiliateMetrics represents affiliate link performance for the brand
type BrandAffiliateMetrics struct {
	TotalClicks  int64                    `json:"total_clicks"`
	UniqueUsers  int64                    `json:"unique_users"`
	TotalRevenue float64                  `json:"total_revenue"`
	TopLinks     []BrandAffiliateLinkItem `json:"top_links"`
	ClickTrend   []ClickTrendPoint        `json:"click_trend"`
	Period       PeriodInfo               `json:"period"`
}

// BrandAffiliateLinkItem represents an affiliate link for the brand
type BrandAffiliateLinkItem struct {
	LinkID      uuid.UUID `json:"link_id"`
	LinkName    string    `json:"link_name"`
	ShortHash   string    `json:"short_hash"`
	Channel     string    `json:"channel"`
	TotalClicks int64     `json:"total_clicks"`
	UniqueUsers int64     `json:"unique_users"`
	Revenue     float64   `json:"revenue"`
	Rank        int       `json:"rank"`
}

// ClickTrendPoint represents a single point in click time-series
type ClickTrendPoint struct {
	Date        time.Time `json:"date"`
	Clicks      int64     `json:"clicks"`
	UniqueUsers int64     `json:"unique_users"`
}

// =============================================================================
// ADDITIONAL BRAND PARTNER ANALYTICS RESPONSE DTOs
// =============================================================================

// BrandProductMetric represents a product metric for the brand
type BrandProductMetric struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	ProductType string    `json:"product_type"` // STANDARD, LIMITED
	Status      string    `json:"status"`
	OrderCount  int64     `json:"order_count"`
	UnitsSold   int64     `json:"units_sold"`
	Revenue     float64   `json:"revenue"`
	Rank        int       `json:"rank"`
}

// BrandTopProductsResponse represents the response for brand's top products
type BrandTopProductsResponse struct {
	Products []BrandProductMetric `json:"products"`
	Period   PeriodInfo           `json:"period"`
}

// BrandCampaignMetric represents a campaign metric for the brand
type BrandCampaignMetric struct {
	CampaignID       uuid.UUID  `json:"campaign_id"`
	CampaignName     string     `json:"campaign_name"`
	Status           string     `json:"status"`
	StartDate        *time.Time `json:"start_date"`
	EndDate          *time.Time `json:"end_date"`
	MilestoneCount   int64      `json:"milestone_count"`
	TaskCount        int64      `json:"task_count"`
	CompletedTasks   int64      `json:"completed_tasks"`
	CompletionRate   float64    `json:"completion_rate"`
	ContentCount     int64      `json:"content_count"`
	TotalViews       int64      `json:"total_views"`
	TotalEngagements int64      `json:"total_engagements"`
}

// BrandCampaignsMetricsResponse represents the response for brand's campaign metrics
type BrandCampaignsMetricsResponse struct {
	Campaigns []BrandCampaignMetric `json:"campaigns"`
	Summary   BrandCampaignsSummary `json:"summary"`
	Period    PeriodInfo            `json:"period"`
}

// BrandContentMetric represents content metrics summary for the brand
type BrandContentMetric struct {
	TotalContent   int64   `json:"total_content"`
	PostedContent  int64   `json:"posted_content"`
	TotalViews     int64   `json:"total_views"`
	TotalLikes     int64   `json:"total_likes"`
	TotalComments  int64   `json:"total_comments"`
	TotalShares    int64   `json:"total_shares"`
	EngagementRate float64 `json:"engagement_rate"`
}

// BrandContentMetricsResponse represents the response for brand's content metrics
type BrandContentMetricsResponse struct {
	Metrics BrandContentMetric `json:"metrics"`
	Period  PeriodInfo         `json:"period"`
}

// BrandRevenueTrendPoint represents a single point in brand revenue time-series
// Note: This overrides the previous definition to match service expectations
type BrandRevenueTrendPoint struct {
	Date       time.Time `json:"date"`
	OrderCount int64     `json:"order_count"`
	UnitsSold  int64     `json:"units_sold"`
	Revenue    float64   `json:"revenue"`
}

// BrandRevenueTrendResponse represents the response for brand's revenue trend
type BrandRevenueTrendResponse struct {
	Trend        []BrandRevenueTrendPoint `json:"trend"`
	TotalRevenue float64                  `json:"total_revenue"`
	Period       PeriodInfo               `json:"period"`
}

// BrandAffiliateMetric represents affiliate metrics summary for the brand
type BrandAffiliateMetric struct {
	TotalLinks  int64 `json:"total_links"`
	ActiveLinks int64 `json:"active_links"`
	TotalClicks int64 `json:"total_clicks"`
}

// BrandAffiliateMetricsResponse represents the response for brand's affiliate metrics
type BrandAffiliateMetricsResponse struct {
	Metrics    BrandAffiliateMetric `json:"metrics"`
	ClickTrend []ClickTrendPoint    `json:"click_trend"`
	Period     PeriodInfo           `json:"period"`
}

// BrandContractDetail represents a contract detail for the brand
type BrandContractDetail struct {
	ContractID     uuid.UUID  `json:"contract_id"`
	ContractNumber string     `json:"contract_number"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	TotalValue     float64    `json:"total_value"`
	PaidAmount     float64    `json:"paid_amount"`
	PendingAmount  float64    `json:"pending_amount"`
	StartDate      *time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date"`
	CampaignCount  int64      `json:"campaign_count"`
}

// BrandContractsResponse represents the response for brand's contracts
type BrandContractsResponse struct {
	Contracts []BrandContractDetail   `json:"contracts"`
	Summary   BrandContractsBreakdown `json:"summary"`
}

type BrandProductRating struct {
	ProductID     uuid.UUID `json:"product_id"`
	ProductName   string    `json:"product_name"`
	Type          string    `json:"type"`
	AverageRating float64   `json:"average_rating"`
}

type BrandTopSoldProducts struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	UnitsSold   int64     `json:"total_sold"`
}
