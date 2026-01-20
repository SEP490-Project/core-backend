package responses

import (
	"time"

	"github.com/google/uuid"
)

// ContentDashboardResponse represents the main dashboard response
type ContentDashboardResponse struct {
	// Filter info
	Period DashboardPeriodInfo `json:"period"`

	// Quick stats section
	QuickStats QuickStatsSection `json:"quick_stats"`

	// Channel metrics cards (Website, Facebook, TikTok)
	ChannelMetrics []ChannelMetricsCard `json:"channel_metrics"`

	// Charts section
	Charts ChartsSection `json:"charts"`

	// Content performance lists
	TopContent    []ContentPerformanceItem `json:"top_content"`
	BottomContent []ContentPerformanceItem `json:"bottom_content"`

	// Upcoming scheduled content
	UpcomingSchedule []ScheduledContentItem `json:"upcoming_schedule"`

	// Active alerts for content staff
	Alerts []AlertItem `json:"alerts"`
}

// DashboardPeriodInfo contains information about the selected period for dashboard
type DashboardPeriodInfo struct {
	PresetLabel   string    `json:"preset_label"`  // "This Month", "Last Week", etc.
	CompareLabel  string    `json:"compare_label"` // "vs Last Month"
	CurrentStart  time.Time `json:"current_start"`
	CurrentEnd    time.Time `json:"current_end"`
	PreviousStart time.Time `json:"previous_start"`
	PreviousEnd   time.Time `json:"previous_end"`
}

// QuickStatsSection represents the quick stats at the top of the dashboard
type QuickStatsSection struct {
	PostsThisWeek    MetricWithGrowth       `json:"posts_this_week"`
	TotalViews       MetricWithGrowth       `json:"total_views"`
	TotalEngagement  MetricWithGrowth       `json:"total_engagement"`
	AverageCTR       MetricWithGrowth       `json:"average_ctr"`
	PendingContent   int64                  `json:"pending_content"`
	PostingFrequency PostingFrequencyMetric `json:"posting_frequency"`
}

// MetricWithGrowth represents a metric value with growth indicator
type MetricWithGrowth struct {
	Value         any     `json:"value"`          // int64, float64
	PreviousValue any     `json:"previous_value"` // Value from previous period
	Growth        float64 `json:"growth"`         // Percentage change
	GrowthStatus  string  `json:"growth_status"`  // "up", "down", "stable"
	CompareLabel  string  `json:"compare_label"`  // "vs last month"
}

// PostingFrequencyMetric represents posting frequency status
type PostingFrequencyMetric struct {
	Actual   int64   `json:"actual"`   // Posts actually published
	Expected int64   `json:"expected"` // Expected posts based on schedule
	Ratio    float64 `json:"ratio"`    // actual / expected
	Status   string  `json:"status"`   // "on_track", "behind", "ahead"
	Source   string  `json:"source"`   // "schedule", "tasks", "average" - where expected came from
}

// ChannelMetricsCard represents metrics for a single channel
type ChannelMetricsCard struct {
	ChannelID        uuid.UUID          `json:"channel_id"`
	ChannelName      string             `json:"channel_name"`
	ChannelCode      string             `json:"channel_code"` // "WEBSITE", "FACEBOOK", "TIKTOK"
	PostCount        int64              `json:"post_count"`
	TotalReach       int64              `json:"total_reach"`
	TotalEngagement  int64              `json:"total_engagement"`
	CTR              float64            `json:"ctr"`
	FollowersCount   int64              `json:"followers_count"`           // Channel followers/subscribers
	FollowersTrend   *TrendIndicator    `json:"followers_trend,omitempty"` // Followers change vs previous period
	FetchedMetrics   map[string]any     `json:"fetched_metrics"`           // Raw platform metrics
	MappedMetrics    map[string]float64 `json:"mapped_metrics"`            // Standardized KPI values
	TopPost          *TopPostBrief      `json:"top_post,omitempty"`
	ReachGrowth      float64            `json:"reach_growth"`
	EngagementGrowth float64            `json:"engagement_growth"`
}

// TopPostBrief represents a brief summary of the top post
type TopPostBrief struct {
	ContentID uuid.UUID `json:"content_id"`
	Title     string    `json:"title"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
}

// ChartsSection contains data for all dashboard charts
type ChartsSection struct {
	// Bar chart: Reach & Engagement by Channel
	ReachByChannel []BarChartDataPoint `json:"reach_by_channel"`

	// Line chart: Trend over time
	TrendData []DashboardTimeSeriesPoint `json:"trend_data"`

	// Pie chart: Content distribution by channel (posts per channel)
	ChannelDistribution []PieChartDataPoint `json:"channel_distribution"`

	// Pie chart: Content type distribution (kept for backwards compatibility)
	ContentTypeDistribution []PieChartDataPoint `json:"content_type_distribution"`
}

// BarChartDataPoint represents a single data point in a bar chart
type BarChartDataPoint struct {
	Label      string `json:"label"` // Channel name
	Reach      int64  `json:"reach"`
	Engagement int64  `json:"engagement"`
}

// DashboardTimeSeriesPoint represents a single point in a time series for dashboard
type DashboardTimeSeriesPoint struct {
	Date        time.Time `json:"date"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Comments    int64     `json:"comments"`
	Shares      int64     `json:"shares"`
	Engagements int64     `json:"engagements"` // Total
}

// PieChartDataPoint represents a slice in a pie chart
type PieChartDataPoint struct {
	Label string  `json:"label"` // "Blog Post", "Video"
	Value int64   `json:"value"` // Count
	Ratio float64 `json:"ratio"` // Percentage
}

// ContentPerformanceItem represents a content item in performance rankings
type ContentPerformanceItem struct {
	ContentID        uuid.UUID  `json:"content_id"`
	Title            string     `json:"title"`
	Type             string     `json:"type"` // "POST", "VIDEO"
	ChannelName      string     `json:"channel_name"`
	Views            int64      `json:"views"`
	Engagement       int64      `json:"engagement"`
	CTR              float64    `json:"ctr"`
	PerformanceScore float64    `json:"performance_score"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	Thumbnail        *string    `json:"thumbnail,omitempty"`
	Rank             int        `json:"rank"`
}

// ScheduledContentItem represents a scheduled content item
type ScheduledContentItem struct {
	ScheduleID  uuid.UUID `json:"schedule_id"`
	ContentID   uuid.UUID `json:"content_id"`
	Title       string    `json:"title"`
	ChannelName string    `json:"channel_name"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Status      string    `json:"status"` // "PENDING", "PROCESSING", etc.
	CreatedBy   string    `json:"created_by"`
	CreatedByID uuid.UUID `json:"created_by_id"`
}

// AlertItem represents an alert in the dashboard
type AlertItem struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`     // "WARNING", "ERROR", "INFO"
	Category      string     `json:"category"` // "LOW_CTR", "CONTENT_REJECTED", etc.
	Severity      string     `json:"severity"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	ReferenceType *string    `json:"reference_type,omitempty"`
	ActionURL     *string    `json:"action_url,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	IsRead        bool       `json:"is_read"`
}

// AlertsResponse represents a list of alerts
type AlertsResponse struct {
	Alerts      []AlertItem `json:"alerts"`
	UnreadCount int64       `json:"unread_count"`
	Total       int64       `json:"total"`
}

// ChannelDetailsResponse represents detailed metrics for a specific channel
type ChannelDetailsResponse struct {
	// Last channel metrics sync time
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`

	// Channel information
	Channel ChannelInfo `json:"channel"`

	// Period information
	Period DashboardPeriodInfo `json:"period"`

	// Standardized mapped metrics
	MappedMetrics map[string]float64 `json:"mapped_metrics"`

	// Raw fetched metrics from platform (FB/TikTok only)
	FetchedMetrics map[string]any `json:"fetched_metrics,omitempty"`

	// Followers count and trend
	FollowersCount int64           `json:"followers_count"`
	FollowersTrend *TrendIndicator `json:"followers_trend,omitempty"`

	// Content trend over time (posts count)
	ContentTrend []ContentTrendPoint `json:"content_trend"`

	// Engagement trend over time
	EngagementTrend []DashboardTimeSeriesPoint `json:"engagement_trend"`

	// Top performing content for this channel
	TopContent []ContentPerformanceItem `json:"top_content"`

	// Recent content for this channel
	RecentContent []ChannelRecentContentItem `json:"recent_content"`

	// Affiliate statistics (if channel has affiliate links)
	AffiliateStats *AffiliateStatsResponse `json:"affiliate_stats,omitempty"`
}

// ChannelInfo represents basic channel information
type ChannelInfo struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description *string    `json:"description,omitempty"`
	HomePageURL *string    `json:"home_page_url,omitempty"`
	IsActive    bool       `json:"is_active"`
	TokenInfo   *TokenInfo `json:"token_info,omitempty"`
}

// TokenInfo represents OAuth token information for the channel
type TokenInfo struct {
	AccountName          *string    `json:"account_name,omitempty"`
	ExternalID           *string    `json:"external_id,omitempty"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at,omitempty"`
	LastSyncedAt         *time.Time `json:"last_synced_at,omitempty"`
}

// TrendIndicator represents a trend comparison
type TrendIndicator struct {
	Value      float64 `json:"value"`      // Change amount
	Percentage float64 `json:"percentage"` // Change %
	Direction  string  `json:"direction"`  // "up", "down", "stable"
}

// ContentTrendPoint represents content count at a point in time
type ContentTrendPoint struct {
	Date  time.Time `json:"date"`
	Posts int       `json:"posts"`
}

// ChannelRecentContentItem represents a recent content item for channel details
// (Different from RecentContentItem in content_staff_analytics_responses.go)
type ChannelRecentContentItem struct {
	ContentID   uuid.UUID  `json:"content_id"`
	Title       string     `json:"title"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Views       int64      `json:"views"`
	Engagement  int64      `json:"engagement"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// AffiliateStatsResponse represents affiliate link statistics
type AffiliateStatsResponse struct {
	TotalLinks  int   `json:"total_links"`
	TotalClicks int64 `json:"total_clicks"`
	UniqueUsers int64 `json:"unique_users"`
	CTR         any   `json:"ctr"` // float64 or "N/A"
	HasLinks    bool  `json:"has_links"`
}
