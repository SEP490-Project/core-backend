package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"time"

	"github.com/google/uuid"
)

// ContentStaffAnalyticsRepository defines the interface for content staff analytics data access
type ContentStaffAnalyticsRepository interface {
	// Content Counts
	GetContentCountByStatus(ctx context.Context, status string, startDate, endDate *time.Time) (int64, error)
	GetTotalContentCount(ctx context.Context, startDate, endDate *time.Time) (int64, error)

	// Aggregated Metrics from KPI Metrics / Content Channels
	GetTotalViews(ctx context.Context, startDate, endDate *time.Time) (int64, error)
	GetTotalEngagements(ctx context.Context, startDate, endDate *time.Time) (int64, error) // likes + comments + shares
	GetTotalClicks(ctx context.Context, startDate, endDate *time.Time) (int64, error)      // From affiliate links

	// Platform Metrics
	GetMetricsByPlatform(ctx context.Context, startDate, endDate *time.Time) ([]dtos.PlatformMetricsResult, error)

	// Top Performers
	GetTopContentByViews(ctx context.Context, platform *string, limit int, startDate, endDate *time.Time) ([]dtos.ContentMetricsResult, error)
	GetTopChannelsByEngagement(ctx context.Context, limit int, startDate, endDate *time.Time) ([]dtos.ChannelMetricsResult, error)

	// Trends
	GetEngagementTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.EngagementTrendResult, error)

	// Recent Content
	GetRecentContent(ctx context.Context, limit int) ([]dtos.RecentContentResult, error)

	// Content Status Breakdown
	GetContentStatusBreakdown(ctx context.Context, startDate, endDate *time.Time) ([]dtos.ContentStatusCount, error)

	// Campaign Content Metrics
	GetCampaignContentMetrics(ctx context.Context, campaignID *uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.CampaignContentMetrics, error)
}
