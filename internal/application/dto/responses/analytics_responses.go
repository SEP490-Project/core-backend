package responses

import (
	"time"

	"github.com/google/uuid"
)

// ContractMetricsResponse represents analytics metrics for a contract
type ContractMetricsResponse struct {
	ContractID   uuid.UUID             `json:"contract_id"`
	ContractName string                `json:"contract_name"`
	BrandName    string                `json:"brand_name"`
	TotalClicks  int64                 `json:"total_clicks"`
	UniqueUsers  int64                 `json:"unique_users"`
	CTR          float64               `json:"ctr"` // Click-through rate
	TopChannels  []ChannelMetricItem   `json:"top_channels"`
	TopLinks     []AffiliateLinkMetric `json:"top_links"`
	Period       PeriodInfo            `json:"period"`
}

// ChannelMetricsResponse represents analytics metrics grouped by channel
type ChannelMetricsResponse struct {
	Channels []ChannelMetricItem `json:"channels"`
	Period   PeriodInfo          `json:"period"`
}

// ChannelMetricItem represents metrics for a single channel
type ChannelMetricItem struct {
	Channel      string  `json:"channel"` // e.g., "TIKTOK", "FACEBOOK"
	TotalClicks  int64   `json:"total_clicks"`
	UniqueUsers  int64   `json:"unique_users"`
	CTR          float64 `json:"ctr"`
	LinkCount    int     `json:"link_count"`    // Number of affiliate links in this channel
	PercentTotal float64 `json:"percent_total"` // Percentage of total clicks
}

// TimeSeriesDataResponse represents time-series data for a single affiliate link
type TimeSeriesDataResponse struct {
	AffiliateLinkID uuid.UUID         `json:"affiliate_link_id"`
	LinkName        string            `json:"link_name"`
	TrackingURL     string            `json:"tracking_url"`
	Channel         string            `json:"channel"`
	DataPoints      []TimeSeriesPoint `json:"data_points"`
	Granularity     string            `json:"granularity"` // e.g., "HOUR", "DAY", "WEEK"
	Period          PeriodInfo        `json:"period"`
}

// TimeSeriesPoint represents a single data point in time series
type TimeSeriesPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Clicks      int64     `json:"clicks"`
	UniqueUsers int64     `json:"unique_users"`
}

// TopPerformerResponse represents top performing affiliate links
type TopPerformerResponse struct {
	TopLinks []AffiliateLinkMetric `json:"top_links"`
	SortBy   string                `json:"sort_by"` // e.g., "CLICKS", "CTR"
	Period   PeriodInfo            `json:"period"`
}

// AffiliateLinkMetric represents metrics for a single affiliate link
type AffiliateLinkMetric struct {
	AffiliateLinkID uuid.UUID `json:"affiliate_link_id"`
	LinkName        string    `json:"link_name"`
	ShortHash       string    `json:"short_hash"`
	TrackingURL     string    `json:"tracking_url"`
	Channel         string    `json:"channel"`
	TotalClicks     int64     `json:"total_clicks"`
	UniqueUsers     int64     `json:"unique_users"`
	CTR             float64   `json:"ctr"`
	Rank            int       `json:"rank,omitempty"` // Position in top performers
}

// DashboardMetricsResponse represents overall dashboard metrics
type DashboardMetricsResponse struct {
	Overview       OverviewMetrics            `json:"overview"`
	TopContracts   []ContractAnalyticsSummary `json:"top_contracts"`
	TopChannels    []ChannelMetricItem        `json:"top_channels"`
	RecentActivity []RecentActivityItem       `json:"recent_activity"`
	TrendData      []TrendDataPoint           `json:"trend_data"` // Last 7 days by default
	Period         PeriodInfo                 `json:"period"`
}

// OverviewMetrics represents high-level overview metrics
type OverviewMetrics struct {
	TotalClicks     int64   `json:"total_clicks"`
	UniqueUsers     int64   `json:"unique_users"`
	TotalLinks      int     `json:"total_links"`
	ActiveContracts int     `json:"active_contracts"`
	AverageCTR      float64 `json:"average_ctr"`
	ClickGrowth     float64 `json:"click_growth"` // Percentage change from previous period
	UserGrowth      float64 `json:"user_growth"`  // Percentage change from previous period
}

// ContractAnalyticsSummary represents summary metrics for a contract (analytics-specific)
type ContractAnalyticsSummary struct {
	ContractID   uuid.UUID `json:"contract_id"`
	ContractName string    `json:"contract_name"`
	BrandName    string    `json:"brand_name"`
	TotalClicks  int64     `json:"total_clicks"`
	UniqueUsers  int64     `json:"unique_users"`
	CTR          float64   `json:"ctr"`
	Rank         int       `json:"rank"`
}

// RecentActivityItem represents a recent click event (aggregated)
type RecentActivityItem struct {
	AffiliateLinkID uuid.UUID `json:"affiliate_link_id"`
	LinkName        string    `json:"link_name"`
	Channel         string    `json:"channel"`
	ClickCount      int64     `json:"click_count"`
	Timestamp       time.Time `json:"timestamp"`
}

// TrendDataPoint represents aggregated trend data for a time period
type TrendDataPoint struct {
	Date        time.Time `json:"date"`
	Clicks      int64     `json:"clicks"`
	UniqueUsers int64     `json:"unique_users"`
}

// PeriodInfo represents the time period for the analytics
type PeriodInfo struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
