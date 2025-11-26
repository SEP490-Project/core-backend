package responses

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONTENT STAFF ANALYTICS RESPONSE DTOs
// =============================================================================

// ContentStaffDashboardResponse represents the complete Content Staff dashboard
type ContentStaffDashboardResponse struct {
	Overview               ContentOverviewMetrics `json:"overview"`
	ContentStatusBreakdown ContentStatusBreakdown `json:"content_status_breakdown"`
	PlatformMetrics        []PlatformMetric       `json:"platform_metrics"`
	TopContent             []ContentMetric        `json:"top_content"`
	TopChannels            []ChannelMetric        `json:"top_channels"`
	RecentContent          []RecentContentItem    `json:"recent_content"`
	EngagementTrend        []EngagementTrendPoint `json:"engagement_trend"`
	Period                 PeriodInfo             `json:"period"`
}

// ContentOverviewMetrics represents high-level content metrics
type ContentOverviewMetrics struct {
	TotalContent     int64   `json:"total_content"`     // Total content pieces created
	PostedContent    int64   `json:"posted_content"`    // Content with status POSTED
	PendingContent   int64   `json:"pending_content"`   // Content awaiting approval/posting
	DraftContent     int64   `json:"draft_content"`     // Content in draft
	TotalViews       int64   `json:"total_views"`       // Sum of all content views
	TotalEngagements int64   `json:"total_engagements"` // Sum of likes + comments + shares
	TotalClicks      int64   `json:"total_clicks"`      // From affiliate links
	EngagementRate   float64 `json:"engagement_rate"`   // Engagement rate percentage
	ViewGrowth       float64 `json:"view_growth"`       // Percentage change from previous period
	EngagementGrowth float64 `json:"engagement_growth"` // Percentage change
}

// ContentStatusBreakdown represents content counts by status
type ContentStatusBreakdown struct {
	TotalCount    int64 `json:"total_count"`
	DraftCount    int64 `json:"draft_count"`
	PendingCount  int64 `json:"pending_count"`
	ApprovedCount int64 `json:"approved_count"`
	RejectedCount int64 `json:"rejected_count"`
	PostedCount   int64 `json:"posted_count"`
}

// PlatformMetric represents aggregated metrics for a single platform
type PlatformMetric struct {
	Platform       string  `json:"platform"`      // FACEBOOK, TIKTOK, INSTAGRAM, YOUTUBE
	ContentCount   int64   `json:"content_count"` // Number of contents on this platform
	TotalViews     int64   `json:"total_views"`
	TotalLikes     int64   `json:"total_likes"`
	TotalComments  int64   `json:"total_comments"`
	TotalShares    int64   `json:"total_shares"`
	TotalClicks    int64   `json:"total_clicks"`    // Affiliate link clicks
	EngagementRate float64 `json:"engagement_rate"` // (likes + comments + shares) / views * 100
}

// ContentMetric represents performance metrics for a single content
type ContentMetric struct {
	ContentID      uuid.UUID  `json:"content_id"`
	Title          string     `json:"title"`
	Platform       string     `json:"platform"`
	ChannelName    string     `json:"channel_name"`
	CampaignName   string     `json:"campaign_name"`
	Views          int64      `json:"views"`
	Likes          int64      `json:"likes"`
	Comments       int64      `json:"comments"`
	Shares         int64      `json:"shares"`
	Clicks         int64      `json:"clicks"` // Affiliate link clicks
	EngagementRate float64    `json:"engagement_rate"`
	PostedAt       *time.Time `json:"posted_at"`
	Rank           int        `json:"rank"`
}

// ChannelMetric represents performance metrics for a single channel
type ChannelMetric struct {
	ChannelID        uuid.UUID `json:"channel_id"`
	ChannelName      string    `json:"channel_name"`
	Platform         string    `json:"platform"`
	OwnerName        string    `json:"owner_name"`
	ContentCount     int64     `json:"content_count"`
	TotalViews       int64     `json:"total_views"`
	TotalLikes       int64     `json:"total_likes"`
	TotalComments    int64     `json:"total_comments"`
	TotalShares      int64     `json:"total_shares"`
	TotalEngagements int64     `json:"total_engagements"`
	EngagementRate   float64   `json:"engagement_rate"`
	Rank             int       `json:"rank"`
}

// RecentContentItem represents recently created/updated content
type RecentContentItem struct {
	ContentID    uuid.UUID `json:"content_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	CampaignName string    `json:"campaign_name"`
	CreatorName  string    `json:"creator_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EngagementTrendPoint represents a single point in engagement time-series
type EngagementTrendPoint struct {
	Date        time.Time `json:"date"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Comments    int64     `json:"comments"`
	Shares      int64     `json:"shares"`
	Engagements int64     `json:"engagements"` // Total: likes + comments + shares
}

// CampaignContentMetric represents content metrics for a campaign
type CampaignContentMetric struct {
	CampaignID       uuid.UUID `json:"campaign_id"`
	CampaignName     string    `json:"campaign_name"`
	ContentCount     int64     `json:"content_count"`
	PostedCount      int64     `json:"posted_count"`
	PendingCount     int64     `json:"pending_count"`
	DraftCount       int64     `json:"draft_count"`
	TotalViews       int64     `json:"total_views"`
	TotalEngagements int64     `json:"total_engagements"`
}
