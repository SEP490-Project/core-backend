package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/constant"
	"time"

	"github.com/google/uuid"
)

// ContentStaffAnalyticsRepository defines the interface for content dashboard data access
// This is a fresh implementation - the existing content_staff_analytics_repository will be deprecated
type ContentStaffAnalyticsRepository interface {
	// Quick Stats Queries

	// GetPostCountByDateRange returns the count of posted content in a date range
	GetPostCountByDateRange(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error)

	// GetTotalViews returns total views from content channel metrics
	GetTotalViews(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error)

	// GetTotalEngagement returns total engagement (likes + comments + shares) from metrics
	GetTotalEngagement(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error)

	// GetAverageCTR returns average CTR across all content channels
	GetAverageCTR(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (float64, error)

	// GetPendingContentCount returns count of content in pending statuses
	GetPendingContentCount(ctx context.Context, startDate, endDate time.Time) (int64, error)

	// Channel Metrics Queries

	// GetChannelMetrics returns aggregated metrics for each channel
	GetChannelMetrics(ctx context.Context, startDate, endDate time.Time) ([]dtos.ChannelMetricsDTO, error)

	// GetTopPostForChannel returns the top performing post for a specific channel
	GetTopPostForChannel(ctx context.Context, channelID uuid.UUID, startDate, endDate time.Time) (*dtos.TopPostDTO, error)

	// Charts Queries

	// GetTrendData returns time series data for trend charts
	GetTrendData(ctx context.Context, startDate, endDate time.Time, granularity constant.TrendGranularity, channelID *uuid.UUID) ([]dtos.TrendDataPointDTO, error)

	// GetContentTypeDistribution returns content count by type
	GetContentTypeDistribution(ctx context.Context, startDate, endDate time.Time) ([]dtos.ContentTypeDistributionDTO, error)

	// GetChannelDistribution returns content count by channel (for pie chart)
	GetChannelDistribution(ctx context.Context, startDate, endDate time.Time) ([]dtos.ChannelDistributionDTO, error)

	// Content Performance Queries

	// GetTopContentByPerformance returns top performing content sorted by performance score
	GetTopContentByPerformance(ctx context.Context, limit int, startDate, endDate time.Time, channelID *uuid.UUID) ([]dtos.ContentPerformanceDTO, error)

	// GetBottomContentByPerformance returns lowest performing content
	GetBottomContentByPerformance(ctx context.Context, limit int, startDate, endDate time.Time, channelID *uuid.UUID) ([]dtos.ContentPerformanceDTO, error)

	// Posting Frequency Source Queries

	// GetScheduledContentCount returns count of scheduled content for a date range
	GetScheduledContentCount(ctx context.Context, startDate, endDate time.Time) (int64, error)

	// GetTaskContentDeliverableCount returns expected content count from tasks in date range
	GetTaskContentDeliverableCount(ctx context.Context, startDate, endDate time.Time) (int64, error)

	// Channel Details Queries

	// GetChannelMappedMetrics returns DELTA kpi_metrics values aggregated for a channel
	// Delta = (last value in period) - (first value in period) for each metric type
	GetChannelMappedMetrics(ctx context.Context, channelID uuid.UUID, startDate, endDate time.Time) (map[string]float64, error)

	// GetChannelFollowers returns the followers count for a channel at a specific time (or latest before)
	GetChannelFollowers(ctx context.Context, channelID uuid.UUID, atTime time.Time) (int64, error)

	// GetAggregatedClicksFromKPIMetrics returns total clicks from affiliate links for a channel in date range
	GetAggregatedClicksFromKPIMetrics(ctx context.Context, channelID *uuid.UUID, startDate, endDate *time.Time) (int64, error)
}
