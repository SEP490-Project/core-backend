package requests

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// SALES STAFF ANALYTICS REQUEST DTOs
// =============================================================================

// SalesStaffDashboardRequest represents a request for Sales Staff dashboard
type SalesStaffDashboardRequest struct {
	Year  *int `form:"year" json:"year" validate:"omitempty,min=2000,max=2100" example:"2025"`
	Month *int `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`
}

// GetYearMonth returns year and month (defaults to current if not provided)
func (r *SalesStaffDashboardRequest) GetYearMonth() (year int, month int) {
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
func (r *SalesStaffDashboardRequest) GetDateRange() (start, end time.Time) {
	year, month := r.GetYearMonth()
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0).Add(-time.Second)
	return start, end
}

// OrdersOverviewRequest represents a request for orders overview
type OrdersOverviewRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	OrderType *string    `form:"order_type" json:"order_type" binding:"omitempty,oneof=STANDARD LIMITED"` // Filter by order type
}

// PreOrdersOverviewRequest represents a request for pre-orders overview
type PreOrdersOverviewRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// TopBrandsRequest represents a request for top brands by revenue
type TopBrandsRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Limit     int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"` // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *TopBrandsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// TopProductsRequest represents a request for top products by revenue
type TopProductsRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	ProductType *string    `form:"product_type" json:"product_type" binding:"omitempty,oneof=STANDARD LIMITED"` // Filter by product type
	Limit       int        `form:"limit" json:"limit" binding:"omitempty,min=1,max=50"`                         // Default: 10
}

// GetLimit returns the limit (defaults to 10)
func (r *TopProductsRequest) GetLimit() int {
	if r.Limit <= 0 {
		return 10
	}
	return r.Limit
}

// RevenueGrowthRequest represents a request for revenue growth time-series
type RevenueGrowthRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Granularity string     `form:"granularity" json:"granularity" binding:"omitempty,oneof=DAY WEEK MONTH"` // Default: DAY
}

// GetGranularity returns the granularity (defaults to DAY)
func (r *RevenueGrowthRequest) GetGranularity() string {
	if r.Granularity == "" {
		return "DAY"
	}
	return r.Granularity
}

// RevenueBySourceRequest represents a request for revenue breakdown by source
type RevenueBySourceRequest struct {
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// PaymentStatusRequest represents a request for payment status overview
type PaymentStatusRequest struct {
	StartDate  *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate    *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	ContractID *uuid.UUID `form:"contract_id" json:"contract_id" binding:"omitempty,uuid"` // Optional: filter by contract
}
