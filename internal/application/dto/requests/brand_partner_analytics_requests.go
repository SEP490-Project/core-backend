package requests

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// BRAND PARTNER ANALYTICS REQUEST DTOs
// =============================================================================

// BrandPartnerDashboardRequest represents a request for Brand Partner dashboard
// BrandUserID is extracted from the authenticated user's context
type BrandPartnerDashboardRequest struct {
	Year  *int `form:"year" json:"year" validate:"omitempty,min=2000,max=2100" example:"2025"`
	Month *int `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`
}

// GetYearMonth returns year and month (defaults to current if not provided)
func (r *BrandPartnerDashboardRequest) GetYearMonth() (year int, month int) {
	now := time.Now()
	if r.Year != nil {
		year = *r.Year
	} else {
		year = now.Year()
	}
	if r.Month != nil {
		month = *r.Month
	} else {
		month = int(now.Month())
	}
	return year, month
}

// GetDateRange returns start and end dates for the month
func (r *BrandPartnerDashboardRequest) GetDateRange() (start, end time.Time) {
	year, month := r.GetYearMonth()
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0).Add(-time.Second)
	return start, end
}

// BrandContractsOverviewRequest represents a request for brand's contracts overview
type BrandContractsOverviewRequest struct {
	BrandUserID  uuid.UUID  `json:"-"` // Injected from auth context
	ContractType *string    `form:"contract_type" json:"contract_type" binding:"omitempty,oneof=ADVERTISING AFFILIATE AMBASSADOR CO_PRODUCING"`
	StartDate    *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate      *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// BrandCampaignsOverviewRequest represents a request for brand's campaigns overview
type BrandCampaignsOverviewRequest struct {
	BrandUserID uuid.UUID  `json:"-"` // Injected from auth context
	Status      *string    `form:"status" json:"status" binding:"omitempty,oneof=DRAFT ACTIVE IN_PROGRESS PENDING FINISHED CANCELLED"`
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// BrandRevenueSourcesRequest represents a request for brand's revenue breakdown
type BrandRevenueSourcesRequest struct {
	BrandUserID uuid.UUID  `json:"-"` // Injected from auth context
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// BrandUpcomingMilestonesRequest represents a request for brand's upcoming milestones
type BrandUpcomingMilestonesRequest struct {
	BrandUserID uuid.UUID `json:"-"`                                                   // Injected from auth context
	Days        int       `form:"days" json:"days" binding:"omitempty,min=1,max=90"`   // Days ahead to look (default: 30)
	Limit       int       `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Max items to return (default: 10)
}

// GetDays returns the days (defaults to 30)
func (r *BrandUpcomingMilestonesRequest) GetDays() int {
	if r.Days <= 0 {
		return 30
	}
	return r.Days
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandUpcomingMilestonesRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandUpcomingPaymentsRequest represents a request for brand's upcoming payments
type BrandUpcomingPaymentsRequest struct {
	BrandUserID uuid.UUID `json:"-"`                                                   // Injected from auth context
	Days        int       `form:"days" json:"days" binding:"omitempty,min=1,max=90"`   // Days ahead to look (default: 30)
	Limit       int       `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Max items to return (default: 10)
}

// GetDays returns the days (defaults to 30)
func (r *BrandUpcomingPaymentsRequest) GetDays() int {
	if r.Days <= 0 {
		return 30
	}
	return r.Days
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandUpcomingPaymentsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandContentPerformanceRequest represents a request for brand's content performance
type BrandContentPerformanceRequest struct {
	BrandUserID uuid.UUID  `json:"-"` // Injected from auth context
	CampaignID  *uuid.UUID `form:"campaign_id" json:"campaign_id" binding:"omitempty,uuid"`
	Platform    *string    `form:"platform" json:"platform" binding:"omitempty,oneof=FACEBOOK TIKTOK INSTAGRAM YOUTUBE"`
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Limit       int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandContentPerformanceRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandAffiliatePerformanceRequest represents a request for brand's affiliate link performance
type BrandAffiliatePerformanceRequest struct {
	BrandUserID uuid.UUID  `json:"-"` // Injected from auth context
	ContractID  *uuid.UUID `form:"contract_id" json:"contract_id" binding:"omitempty,uuid"`
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Limit       int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandAffiliatePerformanceRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandTopProductsRequest represents a request for brand's top products
type BrandTopProductsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Limit     int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandTopProductsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandCampaignsRequest represents a request for brand's campaign metrics
type BrandCampaignsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Status    *string    `form:"status" json:"status" binding:"omitempty,oneof=DRAFT ACTIVE IN_PROGRESS PENDING FINISHED CANCELLED"`
	Limit     int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandCampaignsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// BrandContentMetricsRequest represents a request for brand's content metrics
type BrandContentMetricsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// BrandRevenueTrendRequest represents a request for brand's revenue trend
type BrandRevenueTrendRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Granularity string     `form:"granularity" json:"granularity" binding:"omitempty,oneof=DAY WEEK MONTH"`
}

// GetGranularity returns the granularity (defaults to DAY)
func (r *BrandRevenueTrendRequest) GetGranularity() string {
	if r.Granularity == "" {
		return "DAY"
	}
	return r.Granularity
}

// BrandAffiliateMetricsRequest represents a request for brand's affiliate metrics
type BrandAffiliateMetricsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// BrandContractsRequest represents a request for brand's contract details
type BrandContractsRequest struct {
	Status *string `form:"status" json:"status" binding:"omitempty,oneof=DRAFT PENDING ACTIVE COMPLETED CANCELLED"`
	Limit  int     `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *BrandContractsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}
