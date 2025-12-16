package constant

import "core-backend/internal/domain/enum"

// Content status groups for dashboard metrics

// ValidPostedContentStatus - Content that has been published
var ValidPostedContentStatus = []enum.ContentStatus{
	enum.ContentStatusPosted,
}

// ValidPendingContentStatus - Content awaiting action
var ValidPendingContentStatus = []enum.ContentStatus{
	enum.ContentStatusAwaitStaff,
	enum.ContentStatusAwaitBrand,
}

// ValidDraftContentStatus - Content in draft state
var ValidDraftContentStatus = []enum.ContentStatus{
	enum.ContentStatusDraft,
}

// ValidApprovedContentStatus - Content approved but not yet posted
var ValidApprovedContentStatus = []enum.ContentStatus{
	enum.ContentStatusApproved,
}

// ValidRejectedContentStatus - Content that was rejected
var ValidRejectedContentStatus = []enum.ContentStatus{
	enum.ContentStatusRejected,
}

// ValidActiveContentStatuses - All non-terminal content statuses
var ValidActiveContentStatuses = []enum.ContentStatus{
	enum.ContentStatusDraft,
	enum.ContentStatusAwaitStaff,
	enum.ContentStatusAwaitBrand,
	enum.ContentStatusApproved,
}

// Auto post status groups for channel metrics

// ValidPublishedAutoPostStatus - Channel posts that are published
var ValidPublishedAutoPostStatus = []enum.AutoPostStatus{
	enum.AutoPostStatusPosted,
}

// ValidPendingAutoPostStatus - Channel posts pending/in progress
var ValidPendingAutoPostStatus = []enum.AutoPostStatus{
	enum.AutoPostStatusPending,
	enum.AutoPostStatusInProgress,
}

// ValidFailedAutoPostStatus - Channel posts that failed
var ValidFailedAutoPostStatus = []enum.AutoPostStatus{
	enum.AutoPostStatusFailed,
}

// Alert thresholds (defaults, can be overridden by AdminConfig)
const (
	// DefaultLowCTRThreshold - CTR below this percentage triggers alert (2%)
	DefaultLowCTRThreshold float64 = 2.0

	// DefaultLowEngagementThreshold - Engagement rate below this triggers alert (1%)
	DefaultLowEngagementThreshold float64 = 1.0

	// DefaultDeadlineWarningDays - Days before deadline to generate warning
	DefaultDeadlineWarningDays int = 3

	// DefaultPendingApprovalHours - Hours content can be pending before alert
	DefaultPendingApprovalHours int = 24

	// DefaultMaxScheduleRetries - Maximum retry attempts for scheduled publishing
	DefaultMaxScheduleRetries int = 3

	// DefaultAlertExpiryDays - Days until alerts auto-expire
	DefaultAlertExpiryDays int = 30
)

// Channel codes
const (
	ChannelCodeWebsite  string = "WEBSITE"
	ChannelCodeFacebook string = "FACEBOOK"
	ChannelCodeTikTok   string = "TIKTOK"
)

// TrendGranularity options for time-series charts
type TrendGranularity string

const (
	TrendGranularityHour  TrendGranularity = "HOUR"
	TrendGranularityDay   TrendGranularity = "DAY"
	TrendGranularityWeek  TrendGranularity = "WEEK"
	TrendGranularityMonth TrendGranularity = "MONTH"
)

// IsValid checks if the granularity is valid
func (g TrendGranularity) IsValid() bool {
	switch g {
	case TrendGranularityHour, TrendGranularityDay, TrendGranularityWeek, TrendGranularityMonth:
		return true
	}
	return false
}

// GetPostgreSQLInterval returns the PostgreSQL interval string for time_bucket
func (g TrendGranularity) GetPostgreSQLInterval() string {
	intervals := map[TrendGranularity]string{
		TrendGranularityHour:  "1 hour",
		TrendGranularityDay:   "1 day",
		TrendGranularityWeek:  "1 week",
		TrendGranularityMonth: "1 month",
	}
	return intervals[g]
}

// Performance score weights for content ranking
const (
	ViewsWeight      float64 = 0.3
	EngagementWeight float64 = 0.5
	CTRWeight        float64 = 0.2
	CTRMultiplier    float64 = 20.0 // Multiply CTR to normalize against views/engagement
)

// CalculatePerformanceScore calculates a composite performance score
func CalculatePerformanceScore(views, engagement int64, ctr float64) float64 {
	return float64(views)*ViewsWeight +
		float64(engagement)*EngagementWeight +
		ctr*CTRMultiplier*CTRWeight
}
