package requests

import (
	"time"
)

// SalesDashboardFilter represents the common filter for dashboard endpoints
type SalesDashboardFilter struct {
	FromDateStr *string `form:"from_date" json:"from_date"`
	ToDateStr   *string `form:"to_date" json:"to_date"`
	Limit       int     `form:"limit" json:"limit"`               // Default 5
	PeriodGap   string  `form:"period_gap" json:"period_gap"`     // day, week, month, quarter, year, all
	CompareWith string  `form:"compare_with" json:"compare_with"` // previous day/week/month/quarter/year, and if all (no comparison)
	SortBy      string  `form:"sort_by" json:"sort_by"`           // revenue, units
	SortOrder   string  `form:"sort_order" json:"sort_order"`     // asc, desc

	// Internal uses
	FromDate *time.Time `json:"-"`
	ToDate   *time.Time `json:"-"`
}

// =============================================================================
// REVENUE DETAIL QUERIES - For retrieving orders contributing to revenue metrics
// =============================================================================

// RevenueOrdersFilter represents query parameters for filtering orders by revenue type
type RevenueOrdersFilter struct {
	// Pagination
	Page  int `form:"page" json:"page"`   // Default 1
	Limit int `form:"limit" json:"limit"` // Default 10, Max 100

	// Date Range
	FromDateStr *string `form:"from_date" json:"from_date"` // YYYY-MM-DD
	ToDateStr   *string `form:"to_date" json:"to_date"`     // YYYY-MM-DD

	// Search & Filters
	Search     string `form:"search" json:"search"`           // Search by order ID or customer name
	Status     string `form:"status" json:"status"`           // Filter by status
	CustomerID string `form:"customer_id" json:"customer_id"` // Filter by customer ID

	// Sorting
	SortBy    string `form:"sort_by" json:"sort_by"`       // created_at, total_amount (default: created_at)
	SortOrder string `form:"sort_order" json:"sort_order"` // asc, desc (default: desc)

	// Internal uses
	FromDate *time.Time `json:"-"`
	ToDate   *time.Time `json:"-"`
}
