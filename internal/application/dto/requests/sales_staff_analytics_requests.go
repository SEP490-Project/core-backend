package requests

import (
	"time"
)

// SalesDashboardFilter represents the common filter for dashboard endpoints
type SalesDashboardFilter struct {
	FromDateStr *string `form:"from_date" json:"from_date"`
	ToDateStr   *string `form:"to_date" json:"to_date"`
	Limit       int     `form:"limit" json:"limit"`               // Default 5
	PeriodGap   string  `form:"period_gap" json:"period_gap"`     // day, week, month, quarter, year
	CompareWith string  `form:"compare_with" json:"compare_with"` // previous day/week/month/quarter/year

	// Internal uses
	FromDate *time.Time `json:"-"`
	ToDate   *time.Time `json:"-"`
}
