package requests

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONTENT STAFF ANALYTICS REQUEST DTOs
// =============================================================================

// ContentStaffDashboardRequest represents a request for Content Staff dashboard
type ContentStaffDashboardRequest struct {
	Year  *int `form:"year" json:"year" validate:"omitempty,min=2000,max=2100" example:"2025"`
	Month *int `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`
}

// GetYearMonth returns year and month (defaults to current if not provided)
func (r *ContentStaffDashboardRequest) GetYearMonth() (year int, month int) {
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
func (r *ContentStaffDashboardRequest) GetDateRange() (start, end time.Time) {
	year, month := r.GetYearMonth()
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0).Add(-time.Second)
	return start, end
}

// ContentOverviewRequest represents a request for content overview
type ContentOverviewRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Platform  *string    `form:"platform" json:"platform" binding:"omitempty,oneof=FACEBOOK TIKTOK INSTAGRAM YOUTUBE"` // Filter by platform
}

// ContentPerformanceRequest represents a request for content performance metrics
type ContentPerformanceRequest struct {
	StartDate  *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate    *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Platform   *string    `form:"platform" json:"platform" binding:"omitempty,oneof=FACEBOOK TIKTOK INSTAGRAM YOUTUBE"`
	CampaignID *uuid.UUID `form:"campaign_id" json:"campaign_id" binding:"omitempty,uuid"` // Filter by campaign
	ContractID *uuid.UUID `form:"contract_id" json:"contract_id" binding:"omitempty,uuid"` // Filter by contract
	Limit      int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=100"`    // For top performers, default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *ContentPerformanceRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// ChannelPerformanceRequest represents a request for channel-level metrics
type ChannelPerformanceRequest struct {
	ChannelID *uuid.UUID `form:"channel_id" json:"channel_id" uri:"channel_id" binding:"omitempty,uuid"`
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// TopContentRequest represents a request for top performing content
type TopContentRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Platform  *string    `form:"platform" json:"platform" binding:"omitempty,oneof=FACEBOOK TIKTOK INSTAGRAM YOUTUBE"`
	SortBy    string     `form:"sort_by" json:"sort_by" binding:"omitempty,oneof=VIEWS LIKES COMMENTS SHARES ENGAGEMENT"` // Default: VIEWS
	Limit     int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"`                                     // Default: 10
}

// GetSortBy returns the sort field (defaults to VIEWS)
func (r *TopContentRequest) GetSortBy() string {
	if r.SortBy == "" {
		return "VIEWS"
	}
	return r.SortBy
}

// GetLimit returns the limit (defaults to 10)
func (r *TopContentRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// ContentStatusOverviewRequest represents a request for content status counts
type ContentStatusOverviewRequest struct {
	StartDate  *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate    *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	CampaignID *uuid.UUID `form:"campaign_id" json:"campaign_id" binding:"omitempty,uuid"`
}

// ContentStatusRequest represents a request for content status breakdown
type ContentStatusRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// PlatformMetricsRequest represents a request for platform metrics
type PlatformMetricsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// TopChannelsRequest represents a request for top channels
type TopChannelsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Platform  *string    `form:"platform" json:"platform" binding:"omitempty,oneof=FACEBOOK TIKTOK INSTAGRAM YOUTUBE"`
	Limit     int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"`
}

// GetLimit returns the limit (defaults to 10)
func (r *TopChannelsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// EngagementTrendRequest represents a request for engagement trend
type EngagementTrendRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Granularity string     `form:"granularity" json:"granularity" binding:"omitempty,oneof=DAY WEEK MONTH"`
}

// GetGranularity returns the granularity (defaults to DAY)
func (r *EngagementTrendRequest) GetGranularity() string {
	if r.Granularity == "" {
		return "DAY"
	}
	return r.Granularity
}

// CampaignContentRequest represents a request for campaign content metrics
type CampaignContentRequest struct {
	StartDate  *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate    *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	CampaignID *uuid.UUID `form:"campaign_id" json:"campaign_id" binding:"omitempty,uuid"`
	Limit      int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"`
}

// GetLimit returns the limit (defaults to 10)
func (r *CampaignContentRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}
