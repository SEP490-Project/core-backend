package requests

import (
	"core-backend/internal/domain/constant"
	"time"
)

// ContentDashboardFilterRequest represents the filter parameters for the content dashboard
type ContentDashboardFilterRequest struct {
	// Period preset (takes precedence over custom dates)
	Period *string `form:"period" validate:"omitempty,oneof=TODAY YESTERDAY THIS_WEEK LAST_WEEK THIS_MONTH LAST_MONTH THIS_QUARTER LAST_QUARTER THIS_YEAR LAST_YEAR LAST_7_DAYS LAST_30_DAYS CUSTOM"`

	// Custom date range (used when preset is CUSTOM or not provided)
	FromDate *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate   *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`

	// Optional filters
	ChannelID   *string `form:"channel_id" validate:"omitempty,uuid"`
	ContentType *string `form:"content_type" validate:"omitempty,oneof=POST VIDEO"`
	CampaignID  *string `form:"campaign_id" validate:"omitempty,uuid"`

	// Chart granularity
	TrendGranularity *string `form:"trend_granularity" validate:"omitempty,oneof=HOUR DAY WEEK MONTH"`

	// Content list limits
	TopContentLimit    *int `form:"top_content_limit" validate:"omitempty,min=1,max=50"`
	BottomContentLimit *int `form:"bottom_content_limit" validate:"omitempty,min=1,max=50"`
}

// GetDateRanges returns current period and previous period for comparison
func (r *ContentDashboardFilterRequest) GetDateRanges() (current constant.DateRange, previous constant.DateRange) {
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
func (r *ContentDashboardFilterRequest) GetPreset() constant.DashboardPeriodPreset {
	if r.Period != nil {
		return constant.DashboardPeriodPreset(*r.Period)
	}
	return constant.PeriodPresetThisMonth
}

// GetPresetLabel returns a human-readable label for the preset
func (r *ContentDashboardFilterRequest) GetPresetLabel() string {
	return r.GetPreset().GetLabel()
}

// GetCompareLabel returns a label describing what the current period is compared against
func (r *ContentDashboardFilterRequest) GetCompareLabel() string {
	return r.GetPreset().GetCompareLabel()
}

// GetTrendGranularity returns the trend granularity with default
func (r *ContentDashboardFilterRequest) GetTrendGranularity() constant.TrendGranularity {
	if r.TrendGranularity != nil {
		return constant.TrendGranularity(*r.TrendGranularity)
	}
	return constant.TrendGranularityDay // Default
}

// GetTopContentLimit returns the top content limit with default
func (r *ContentDashboardFilterRequest) GetTopContentLimit() int {
	if r.TopContentLimit != nil {
		return *r.TopContentLimit
	}
	return 5
}

// GetBottomContentLimit returns the bottom content limit with default
func (r *ContentDashboardFilterRequest) GetBottomContentLimit() int {
	if r.BottomContentLimit != nil {
		return *r.BottomContentLimit
	}
	return 5
}

// ChannelDetailsRequest represents the filter parameters for channel details
type ChannelDetailsRequest struct {
	// Period preset
	Period *string `form:"period" validate:"omitempty,oneof=TODAY YESTERDAY THIS_WEEK LAST_WEEK THIS_MONTH LAST_MONTH THIS_QUARTER LAST_QUARTER THIS_YEAR LAST_YEAR LAST_7_DAYS LAST_30_DAYS CUSTOM"`

	// Custom date range (used when preset is CUSTOM)
	FromDate *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate   *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`

	// Chart granularity
	TrendGranularity *string `form:"trend_granularity" validate:"omitempty,oneof=HOUR DAY WEEK MONTH"`

	// Content list limits
	TopContentLimit    *int `form:"top_content_limit" validate:"omitempty,min=1,max=50"`
	RecentContentLimit *int `form:"recent_content_limit" validate:"omitempty,min=1,max=50"`
}

// GetDateRanges returns current period and previous period for comparison
func (r *ChannelDetailsRequest) GetDateRanges() (current constant.DateRange, previous constant.DateRange) {
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

// GetTrendGranularity returns the trend granularity with default
func (r *ChannelDetailsRequest) GetTrendGranularity() constant.TrendGranularity {
	if r.TrendGranularity != nil {
		return constant.TrendGranularity(*r.TrendGranularity)
	}
	return constant.TrendGranularityDay
}

// GetTopContentLimit returns the top content limit with default
func (r *ChannelDetailsRequest) GetTopContentLimit() int {
	if r.TopContentLimit != nil {
		return *r.TopContentLimit
	}
	return 10
}

// GetRecentContentLimit returns the recent content limit with default
func (r *ChannelDetailsRequest) GetRecentContentLimit() int {
	if r.RecentContentLimit != nil {
		return *r.RecentContentLimit
	}
	return 10
}

// GetPresetLabel returns a human-readable label for the preset
func (r *ChannelDetailsRequest) GetPresetLabel() string {
	preset := constant.PeriodPresetThisMonth
	if r.Period != nil {
		preset = constant.DashboardPeriodPreset(*r.Period)
	}
	return preset.GetLabel()
}

// GetCompareLabel returns a label describing what the current period is compared against
func (r *ChannelDetailsRequest) GetCompareLabel() string {
	preset := constant.PeriodPresetThisMonth
	if r.Period != nil {
		preset = constant.DashboardPeriodPreset(*r.Period)
	}
	return preset.GetCompareLabel()
}
