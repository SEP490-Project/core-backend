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
