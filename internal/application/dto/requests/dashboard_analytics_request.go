package requests

import (
	"core-backend/internal/domain/constant"
	"time"

	"github.com/google/uuid"
)

// DashboardFilterRequest is the unified filter for Marketing and Brand Partner dashboards
type DashboardFilterRequest struct {
	// Period preset (takes precedence over custom dates)
	Period *string `form:"period" validate:"omitempty,oneof=TODAY YESTERDAY THIS_WEEK LAST_WEEK THIS_MONTH LAST_MONTH THIS_QUARTER LAST_QUARTER THIS_YEAR LAST_YEAR LAST_7_DAYS LAST_30_DAYS CUSTOM"`

	// Custom date range (used when Period is CUSTOM or not provided)
	FromDate *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate   *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`

	// Chart granularity for time-series charts
	TrendGranularity *string `form:"trend_granularity" validate:"omitempty,oneof=HOUR DAY WEEK MONTH"`

	// Limit for top-N queries
	Limit *int `form:"limit" validate:"omitempty,min=1,max=50"`

	// For Brand Partner only (extracted from JWT, not query param)
	BrandUserID *uuid.UUID `form:"-"` // Set by handler from auth context
}

// GetDateRanges returns current period and previous period for comparison
func (r *DashboardFilterRequest) GetDateRanges() (current constant.DateRange, previous constant.DateRange) {
	preset := constant.PeriodPresetThisMonth // Default

	if r.Period != nil {
		preset = constant.DashboardPeriodPreset(*r.Period)
	}

	var customStart, customEnd *time.Time

	if preset == constant.PeriodPresetCustom && r.FromDate != nil && r.ToDate != nil {
		from, _ := time.Parse("2006-01-02", *r.FromDate)
		to, _ := time.Parse("2006-01-02", *r.ToDate)
		customStart = &from
		customEnd = &to
	}

	return constant.GetDateRangeForPreset(preset, customStart, customEnd)
}

// GetPreset returns the preset or default
func (r *DashboardFilterRequest) GetPreset() constant.DashboardPeriodPreset {
	if r.Period != nil {
		return constant.DashboardPeriodPreset(*r.Period)
	}
	return constant.PeriodPresetThisMonth
}

// GetPresetLabel returns a human-readable label for the preset
func (r *DashboardFilterRequest) GetPresetLabel() string {
	return r.GetPreset().GetLabel()
}

// GetCompareLabel returns a label describing what the current period is compared against
func (r *DashboardFilterRequest) GetCompareLabel() string {
	return r.GetPreset().GetCompareLabel()
}

// GetTrendGranularity returns the trend granularity with default
func (r *DashboardFilterRequest) GetTrendGranularity() constant.TrendGranularity {
	if r.TrendGranularity != nil {
		return constant.TrendGranularity(*r.TrendGranularity)
	}
	return constant.TrendGranularityDay // Default
}

// GetLimit returns the limit with default
func (r *DashboardFilterRequest) GetLimit() int {
	if r.Limit != nil {
		return *r.Limit
	}
	return 5
}

// GetPeriodInfo builds a PeriodInfo response struct from the filter
func (r *DashboardFilterRequest) GetPeriodInfo() PeriodInfo {
	current, previous := r.GetDateRanges()
	return PeriodInfo{
		StartDate:       current.Start,
		EndDate:         current.End,
		PresetLabel:     r.GetPresetLabel(),
		ComparisonLabel: r.GetCompareLabel(),
		PreviousStart:   previous.Start,
		PreviousEnd:     previous.End,
	}
}

// PeriodInfo represents the period information included in responses
type PeriodInfo struct {
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	PresetLabel     string    `json:"preset_label"`     // e.g., "This Month"
	ComparisonLabel string    `json:"comparison_label"` // e.g., "vs Last Month"
	PreviousStart   time.Time `json:"previous_start"`
	PreviousEnd     time.Time `json:"previous_end"`
}
