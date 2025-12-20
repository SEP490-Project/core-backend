package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// clickEventRepository implements ClickEventRepository interface with TimescaleDB support
type clickEventRepository struct {
	irepository.GenericRepository[model.ClickEvent]
	db *gorm.DB
}

// NewClickEventRepository creates a new instance of ClickEventRepository
func NewClickEventRepository(db *gorm.DB) irepository.ClickEventRepository {
	return &clickEventRepository{
		GenericRepository: NewGenericRepository[model.ClickEvent](db),
		db:                db,
	}
}

// GetRecentClicks retrieves click events since a specific timestamp
func (r *clickEventRepository) GetRecentClicks(ctx context.Context, since time.Time, limit int) ([]model.ClickEvent, error) {
	var events []model.ClickEvent

	query := r.db.WithContext(ctx).
		Where("clicked_at >= ?", since).
		Order("clicked_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&events).Error
	return events, err
}

// GetAggregatedClicks returns click counts grouped by affiliate link ID since a specific timestamp
func (r *clickEventRepository) GetAggregatedClicks(ctx context.Context, since time.Time) (map[uuid.UUID]int, error) {
	type Result struct {
		AffiliateLinkID uuid.UUID
		Count           int
	}
	var results []Result

	err := r.db.WithContext(ctx).
		Model(&model.ClickEvent{}).
		Where("clicked_at >= ?", since).
		Select("affiliate_link_id, COUNT(*) as count").
		Group("affiliate_link_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	aggregates := make(map[uuid.UUID]int)
	for _, res := range results {
		aggregates[res.AffiliateLinkID] = res.Count
	}
	return aggregates, nil
}

// GetClicksByTimeRange retrieves click events for a specific affiliate link within a time range
// Optimized for TimescaleDB chunk exclusion
func (r *clickEventRepository) GetClicksByTimeRange(
	ctx context.Context,
	affiliateLinkID uuid.UUID,
	startTime, endTime time.Time,
) ([]model.ClickEvent, error) {
	var events []model.ClickEvent

	// TimescaleDB automatically scans only relevant chunks based on clicked_at range
	err := r.db.WithContext(ctx).
		Where("affiliate_link_id = ?", affiliateLinkID).
		Where("clicked_at >= ?", startTime).
		Where("clicked_at < ?", endTime).
		Order("clicked_at DESC").
		Find(&events).Error

	return events, err
}

// GetHourlyStats retrieves aggregated hourly click statistics using TimescaleDB time_bucket
func (r *clickEventRepository) GetHourlyStats(
	ctx context.Context,
	affiliateLinkID uuid.UUID,
	startTime, endTime time.Time,
) ([]dtos.HourlyClickStats, error) {
	var stats []dtos.HourlyClickStats

	// Use TimescaleDB's time_bucket function for efficient aggregation
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			time_bucket('1 hour', clicked_at) AS hour,
			COUNT(*) AS total_clicks,
			COUNT(DISTINCT COALESCE(user_id::text, ip_address::text)) AS unique_users,
			COUNT(DISTINCT session_id) FILTER (WHERE session_id IS NOT NULL) AS unique_sessions
		FROM click_events
		WHERE affiliate_link_id = ?
			AND clicked_at >= ?
			AND clicked_at < ?
		GROUP BY hour
		ORDER BY hour DESC
	`, affiliateLinkID, startTime, endTime).Scan(&stats).Error

	return stats, err
}

// GetDailyStats retrieves aggregated daily click statistics
func (r *clickEventRepository) GetDailyStats(
	ctx context.Context,
	affiliateLinkID uuid.UUID,
	startTime, endTime time.Time,
) ([]dtos.DailyClickStats, error) {
	var stats []dtos.DailyClickStats

	// Use TimescaleDB's time_bucket for daily aggregation
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			time_bucket('1 day', clicked_at) AS date,
			COUNT(*) AS total_clicks,
			COUNT(DISTINCT COALESCE(user_id::text, ip_address::text)) AS unique_users,
			COUNT(DISTINCT session_id) FILTER (WHERE session_id IS NOT NULL) AS unique_sessions
		FROM click_events
		WHERE affiliate_link_id = ?
			AND clicked_at >= ?
			AND clicked_at < ?
		GROUP BY date
		ORDER BY date DESC
	`, affiliateLinkID, startTime, endTime).Scan(&stats).Error

	return stats, err
}

// GetClickCountByAffiliate counts total clicks for a specific affiliate link
func (r *clickEventRepository) GetClickCountByAffiliate(
	ctx context.Context,
	affiliateLinkID uuid.UUID,
	startTime, endTime time.Time,
) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.ClickEvent{}).
		Where("affiliate_link_id = ?", affiliateLinkID).
		Where("clicked_at >= ?", startTime).
		Where("clicked_at < ?", endTime).
		Count(&count).Error

	return count, err
}

// GetUniqueUserCountByAffiliate counts unique users who clicked an affiliate link
func (r *clickEventRepository) GetUniqueUserCountByAffiliate(
	ctx context.Context,
	affiliateLinkID uuid.UUID,
	startTime, endTime time.Time,
) (int64, error) {
	var count int64

	// Count distinct user_id OR ip_address (for anonymous users)
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT COALESCE(user_id::text, ip_address::text))
		FROM click_events
		WHERE affiliate_link_id = ?
			AND clicked_at >= ?
			AND clicked_at < ?
	`, affiliateLinkID, startTime, endTime).Scan(&count).Error

	return count, err
}

// GetClicksByContract retrieves all clicks for affiliate links under a specific contract
func (r *clickEventRepository) GetClicksByContract(
	ctx context.Context,
	contractID uuid.UUID,
	startTime, endTime time.Time,
) ([]model.ClickEvent, error) {
	var events []model.ClickEvent

	// Join with affiliate_links to filter by contract_id
	err := r.db.WithContext(ctx).
		Table("click_events ce").
		Select("ce.*").
		Joins("INNER JOIN affiliate_links al ON ce.affiliate_link_id = al.id").
		Where("al.contract_id = ?", contractID).
		Where("ce.clicked_at >= ?", startTime).
		Where("ce.clicked_at < ?", endTime).
		Order("ce.clicked_at DESC").
		Scan(&events).Error

	return events, err
}

// GetClicksByChannel retrieves all clicks for affiliate links in a specific channel
func (r *clickEventRepository) GetClicksByChannel(
	ctx context.Context,
	channelID uuid.UUID,
	startTime, endTime time.Time,
) ([]model.ClickEvent, error) {
	var events []model.ClickEvent

	// Join with affiliate_links to filter by channel_id
	err := r.db.WithContext(ctx).
		Table("click_events ce").
		Select("ce.*").
		Joins("INNER JOIN affiliate_links al ON ce.affiliate_link_id = al.id").
		Where("al.channel_id = ?", channelID).
		Where("ce.clicked_at >= ?", startTime).
		Where("ce.clicked_at < ?", endTime).
		Order("ce.clicked_at DESC").
		Scan(&events).Error

	return events, err
}

// GetTopPerformingLinks retrieves top N affiliate links by click count within a time range
func (r *clickEventRepository) GetTopPerformingLinks(
	ctx context.Context,
	startTime, endTime time.Time,
	limit int,
) ([]dtos.AffiliateLinkPerformance, error) {
	var performance []dtos.AffiliateLinkPerformance

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			al.id AS affiliate_link_id,
			al.hash,
			COUNT(ce.id) AS total_clicks,
			COUNT(DISTINCT COALESCE(ce.user_id::text, ce.ip_address::text)) AS unique_users,
			COALESCE(c.title, 'Unknown') AS content_title,
			COALESCE(ch.name, 'Unknown') AS channel_name
		FROM click_events ce
		INNER JOIN affiliate_links al ON ce.affiliate_link_id = al.id
		LEFT JOIN contents c ON al.content_id = c.id
		LEFT JOIN channels ch ON al.channel_id = ch.id
		WHERE ce.clicked_at >= ? AND ce.clicked_at < ?
		GROUP BY al.id, al.hash, c.title, ch.name
		ORDER BY total_clicks DESC
		LIMIT ?
	`, startTime, endTime, limit).Scan(&performance).Error

	return performance, err
}

// GetGlobalOverview retrieves global click statistics for the platform
func (r *clickEventRepository) GetGlobalOverview(
	ctx context.Context,
	startTime, endTime time.Time,
) (dtos.GlobalClickStats, error) {
	var stats dtos.GlobalClickStats

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			COUNT(*) AS total_clicks,
			COUNT(DISTINCT COALESCE(user_id::text, ip_address::text)) AS unique_users,
			COUNT(DISTINCT session_id) FILTER (WHERE session_id IS NOT NULL) AS unique_sessions
		FROM click_events
		WHERE clicked_at >= ? AND clicked_at < ?
	`, startTime, endTime).Scan(&stats).Error

	return stats, err
}

// GetTopContracts retrieves top N contracts by click performance
func (r *clickEventRepository) GetTopContracts(
	ctx context.Context,
	startTime, endTime time.Time,
	limit int,
) ([]dtos.ContractPerformance, error) {
	var performance []dtos.ContractPerformance

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			al.contract_id,
			COALESCE(c.contract_number, 'Unknown') AS contract_name,
			COUNT(ce.id) AS total_clicks,
			COUNT(DISTINCT COALESCE(ce.user_id::text, ce.ip_address::text)) AS unique_users
		FROM click_events ce
		INNER JOIN affiliate_links al ON ce.affiliate_link_id = al.id
		LEFT JOIN contracts c ON al.contract_id = c.id
		WHERE ce.clicked_at >= ? AND ce.clicked_at < ?
			AND al.contract_id IS NOT NULL
		GROUP BY al.contract_id, c.contract_number
		ORDER BY total_clicks DESC
		LIMIT ?
	`, startTime, endTime, limit).Scan(&performance).Error

	return performance, err
}

// GetTopChannels retrieves top N channels by click performance
func (r *clickEventRepository) GetTopChannels(
	ctx context.Context,
	startTime, endTime time.Time,
	limit int,
) ([]dtos.ChannelPerformance, error) {
	var performance []dtos.ChannelPerformance

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			al.channel_id,
			COALESCE(ch.name, 'Unknown') AS channel_name,
			COUNT(ce.id) AS total_clicks,
			COUNT(DISTINCT COALESCE(ce.user_id::text, ce.ip_address::text)) AS unique_users
		FROM click_events ce
		INNER JOIN affiliate_links al ON ce.affiliate_link_id = al.id
		LEFT JOIN channels ch ON al.channel_id = ch.id
		WHERE ce.clicked_at >= ? AND ce.clicked_at < ?
			AND al.channel_id IS NOT NULL
		GROUP BY al.channel_id, ch.name
		ORDER BY total_clicks DESC
		LIMIT ?
	`, startTime, endTime, limit).Scan(&performance).Error

	return performance, err
}

// GetGlobalTrendData retrieves daily trend data for the dashboard
func (r *clickEventRepository) GetGlobalTrendData(
	ctx context.Context,
	startTime, endTime time.Time,
) ([]dtos.DailyClickStats, error) {
	var stats []dtos.DailyClickStats

	// Use TimescaleDB's time_bucket for daily aggregation
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			time_bucket('1 day', clicked_at) AS date,
			COUNT(*) AS total_clicks,
			COUNT(DISTINCT COALESCE(user_id::text, ip_address::text)) AS unique_users,
			COUNT(DISTINCT session_id) FILTER (WHERE session_id IS NOT NULL) AS unique_sessions
		FROM click_events
		WHERE clicked_at >= ? AND clicked_at < ?
		GROUP BY date
		ORDER BY date ASC
	`, startTime, endTime).Scan(&stats).Error

	return stats, err
}
