package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ClickEventRepository defines the interface for click event data access
// This repository handles TimescaleDB hypertable-specific queries
type ClickEventRepository interface {
	// Embed generic repository for standard CRUD operations
	GenericRepository[model.ClickEvent]

	// GetRecentClicks retrieves click events since a specific timestamp
	// Used by aggregation job to process new clicks
	GetRecentClicks(ctx context.Context, since time.Time, limit int) ([]model.ClickEvent, error)

	// GetAggregatedClicks returns click counts grouped by affiliate link ID since a specific timestamp
	GetAggregatedClicks(ctx context.Context, since time.Time) (map[uuid.UUID]int, error)

	// GetClicksByTimeRange retrieves click events for a specific affiliate link within a time range
	// Optimized for TimescaleDB chunk exclusion
	GetClicksByTimeRange(ctx context.Context, affiliateLinkID uuid.UUID, startTime, endTime time.Time) ([]model.ClickEvent, error)

	// GetHourlyStats retrieves aggregated hourly click statistics for an affiliate link
	// Uses TimescaleDB time_bucket function for efficient aggregation
	GetHourlyStats(ctx context.Context, affiliateLinkID uuid.UUID, startTime, endTime time.Time) ([]dtos.HourlyClickStats, error)

	// GetDailyStats retrieves aggregated daily click statistics for an affiliate link
	GetDailyStats(ctx context.Context, affiliateLinkID uuid.UUID, startTime, endTime time.Time) ([]dtos.DailyClickStats, error)

	// GetClickCountByAffiliate counts total clicks for a specific affiliate link
	GetClickCountByAffiliate(ctx context.Context, affiliateLinkID uuid.UUID, startTime, endTime time.Time) (int64, error)

	// GetUniqueUserCountByAffiliate counts unique users who clicked an affiliate link
	GetUniqueUserCountByAffiliate(ctx context.Context, affiliateLinkID uuid.UUID, startTime, endTime time.Time) (int64, error)

	// GetClicksByContract retrieves all clicks for affiliate links under a specific contract
	GetClicksByContract(ctx context.Context, contractID uuid.UUID, startTime, endTime time.Time) ([]model.ClickEvent, error)

	// GetClicksByChannel retrieves all clicks for affiliate links in a specific channel
	GetClicksByChannel(ctx context.Context, channelID uuid.UUID, startTime, endTime time.Time) ([]model.ClickEvent, error)

	// GetTopPerformingLinks retrieves top N affiliate links by click count within a time range
	GetTopPerformingLinks(ctx context.Context, startTime, endTime time.Time, limit int) ([]dtos.AffiliateLinkPerformance, error)

	// GetGlobalOverview retrieves global click statistics for the platform
	GetGlobalOverview(ctx context.Context, startTime, endTime time.Time) (dtos.GlobalClickStats, error)

	// GetTopContracts retrieves top N contracts by click performance
	GetTopContracts(ctx context.Context, startTime, endTime time.Time, limit int) ([]dtos.ContractPerformance, error)

	// GetTopChannels retrieves top N channels by click performance
	GetTopChannels(ctx context.Context, startTime, endTime time.Time, limit int) ([]dtos.ChannelPerformance, error)

	// GetGlobalTrendData retrieves daily trend data for the dashboard
	GetGlobalTrendData(ctx context.Context, startTime, endTime time.Time) ([]dtos.DailyClickStats, error)
}
