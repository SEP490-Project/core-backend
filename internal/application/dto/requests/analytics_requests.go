package requests

import (
	"time"

	"github.com/google/uuid"
)

// CTRMetricsRequest represents a request for CTR analytics with time range filters
type CTRMetricsRequest struct {
	StartDate *time.Time `json:"start_date" form:"start_date" binding:"omitempty"` // Start date for analytics range
	EndDate   *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`     // End date for analytics range

	// Optional filters
	ContractID  *uuid.UUID `json:"contract_id" form:"contract_id" binding:"omitempty,uuid"` // Filter by contract
	Channel     *string    `json:"channel" form:"channel" binding:"omitempty,oneof=TIKTOK FACEBOOK INSTAGRAM YOUTUBE SHOPEE LAZADA"`
	Granularity *string    `json:"granularity" form:"granularity" binding:"omitempty,oneof=HOUR DAY WEEK MONTH"` // Time bucket size (default: DAY)

	// Pagination
	Page     int `json:"page" form:"page" binding:"omitempty,min=1"`                   // Page number (default: 1)
	PageSize int `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=100"` // Items per page (default: 10, max: 100)
}

// TimeSeriesRequest represents a request for time-series data
type TimeSeriesRequest struct {
	AffiliateLinkID uuid.UUID  `json:"affiliate_link_id" uri:"affiliate_link_id" binding:"required,uuid"`
	StartDate       *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate         *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
	Granularity     string     `json:"granularity" form:"granularity" binding:"omitempty,oneof=HOUR DAY WEEK MONTH"` // Default: DAY
}

// TopPerformersRequest represents a request for top performing affiliate links
type TopPerformersRequest struct {
	StartDate *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate   *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
	SortBy    string     `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=CLICKS CTR ENGAGEMENT"` // Default: CLICKS
	Limit     int        `json:"limit" form:"limit" binding:"omitempty,min=1,max=50"`                    // Default: 10
}

// DashboardRequest represents a request for dashboard metrics (optional time range)
type DashboardRequest struct {
	StartDate *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate   *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
}

// ContractMetricsRequest represents a request for contract-level metrics
type ContractMetricsRequest struct {
	ContractID uuid.UUID  `json:"contract_id" uri:"contract_id" binding:"required,uuid"`
	StartDate  *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate    *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
}

// ChannelMetricsRequest represents a request for channel comparison
type ChannelMetricsRequest struct {
	StartDate *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate   *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
}
