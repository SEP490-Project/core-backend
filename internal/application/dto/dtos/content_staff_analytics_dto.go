package dtos

import (
	"time"

	"github.com/google/uuid"
)

// PlatformMetricsResult represents platform metrics query result
type PlatformMetricsResult struct {
	Platform       string
	ContentCount   int64
	TotalViews     int64
	TotalLikes     int64
	TotalComments  int64
	TotalShares    int64
	TotalClicks    int64
	EngagementRate float64
}

// ContentMetricsResult represents content metrics query result
type ContentMetricsResult struct {
	ContentID      uuid.UUID
	Title          string
	Platform       string
	ChannelName    string
	CampaignName   string
	Views          int64
	Likes          int64
	Comments       int64
	Shares         int64
	Clicks         int64
	EngagementRate float64
	PostedAt       *time.Time
}

// ChannelMetricsResult represents channel metrics query result
type ChannelMetricsResult struct {
	ChannelID        uuid.UUID
	ChannelName      string
	Platform         string
	OwnerName        string
	FollowerCount    int64
	ContentCount     int64
	TotalViews       int64
	TotalLikes       int64
	TotalComments    int64
	TotalShares      int64
	TotalEngagements int64
	EngagementRate   float64
}

// EngagementTrendResult represents engagement trend query result
type EngagementTrendResult struct {
	Date        time.Time
	Views       int64
	Likes       int64
	Comments    int64
	Shares      int64
	Clicks      int64
	Engagements int64
}

// RecentContentResult represents recent content query result
type RecentContentResult struct {
	ContentID    uuid.UUID
	Title        string
	Status       string
	CampaignName string
	CreatorName  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ContentStatusCount represents content count by status
type ContentStatusCount struct {
	Status string
	Count  int64
}

// CampaignContentMetrics represents content metrics by campaign
type CampaignContentMetrics struct {
	CampaignID       uuid.UUID
	CampaignName     string
	ContentCount     int64
	PostedCount      int64
	PendingCount     int64
	DraftCount       int64
	TotalViews       int64
	TotalEngagements int64
}
