package requests

import "time"

// =============================================================================
// ADMIN ANALYTICS REQUEST DTOs
// =============================================================================

// AdminDashboardRequest represents a request for Admin dashboard
type AdminDashboardRequest struct {
	Year  *int `form:"year" json:"year" validate:"omitempty,min=2000,max=2100" example:"2025"`
	Month *int `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`
}

// GetYearMonth returns year and month (defaults to current if not provided)
func (r *AdminDashboardRequest) GetYearMonth() (year int, month int) {
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
func (r *AdminDashboardRequest) GetDateRange() (start, end time.Time) {
	year, month := r.GetYearMonth()
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0).Add(-time.Second)
	return start, end
}

// UsersOverviewRequest represents a request for users overview
type UsersOverviewRequest struct {
	Role      *string    `form:"role" json:"role" binding:"omitempty,oneof=ADMIN MARKETING_STAFF SALES_STAFF CONTENT_STAFF BRAND_PARTNER CUSTOMER"`
	StartDate *time.Time `form:"start_date" json:"start_date" binding:"omitempty"` // Filter by registration date
	EndDate   *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
}

// PlatformRevenueRequest represents a request for platform-wide revenue
type PlatformRevenueRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Granularity string     `form:"granularity" json:"granularity" binding:"omitempty,oneof=DAY WEEK MONTH"` // Default: MONTH
}

// GetGranularity returns the granularity (defaults to MONTH)
func (r *PlatformRevenueRequest) GetGranularity() string {
	if r.Granularity == "" {
		return "MONTH"
	}
	return r.Granularity
}

// SystemHealthRequest represents a request for system health metrics
type SystemHealthRequest struct {
	// No parameters needed - returns current system state
}

// UserGrowthRequest represents a request for user growth over time
type UserGrowthRequest struct {
	StartDate   *time.Time `form:"start_date" json:"start_date" binding:"omitempty"`
	EndDate     *time.Time `form:"end_date" json:"end_date" binding:"omitempty"`
	Granularity string     `form:"granularity" json:"granularity" binding:"omitempty,oneof=DAY WEEK MONTH"` // Default: MONTH
	Role        *string    `form:"role" json:"role" binding:"omitempty"`                                    // Filter by role
}

// GetGranularity returns the granularity (defaults to MONTH)
func (r *UserGrowthRequest) GetGranularity() string {
	if r.Granularity == "" {
		return "MONTH"
	}
	return r.Granularity
}
