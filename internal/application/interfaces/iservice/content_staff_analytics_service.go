package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// ContentStaffAnalyticsService defines the interface for Content Staff analytics operations
type ContentStaffAnalyticsService interface {
	// GetDashboard returns the complete Content Staff dashboard with all metrics
	GetDashboard(ctx context.Context, req *requests.ContentStaffDashboardRequest) (*responses.ContentStaffDashboardResponse, error)

	// GetContentStatusBreakdown returns content counts by status
	GetContentStatusBreakdown(ctx context.Context, req *requests.ContentStatusRequest) (*responses.ContentStatusBreakdown, error)

	// GetMetricsByPlatform returns metrics grouped by platform (Facebook, TikTok, etc.)
	GetMetricsByPlatform(ctx context.Context, req *requests.PlatformMetricsRequest) ([]responses.PlatformMetric, error)

	// GetTopContent returns top performing content by views
	GetTopContent(ctx context.Context, req *requests.TopContentRequest) ([]responses.ContentMetric, error)

	// GetTopChannels returns top performing channels by engagement
	GetTopChannels(ctx context.Context, req *requests.TopChannelsRequest) ([]responses.ChannelMetric, error)

	// GetEngagementTrend returns engagement metrics over time
	GetEngagementTrend(ctx context.Context, req *requests.EngagementTrendRequest) ([]responses.EngagementTrendPoint, error)

	// GetCampaignContentMetrics returns content metrics by campaign
	GetCampaignContentMetrics(ctx context.Context, req *requests.CampaignContentRequest) ([]responses.CampaignContentMetric, error)
}
