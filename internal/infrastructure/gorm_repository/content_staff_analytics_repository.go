package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentStaffAnalyticsRepository struct {
	db *gorm.DB
}

// NewContentStaffAnalyticsRepository creates a new content staff analytics repository
func NewContentStaffAnalyticsRepository(db *gorm.DB) irepository.ContentStaffAnalyticsRepository {
	return &contentStaffAnalyticsRepository{db: db}
}

// GetContentCountByStatus returns the count of content by status
func (r *contentStaffAnalyticsRepository) GetContentCountByStatus(ctx context.Context, status string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("contents").Where("deleted_at IS NULL")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get content count by status", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetTotalContentCount returns the total count of content
func (r *contentStaffAnalyticsRepository) GetTotalContentCount(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	return r.GetContentCountByStatus(ctx, "", startDate, endDate)
}

// GetTotalViews returns total views from content channels metrics JSONB
func (r *contentStaffAnalyticsRepository) GetTotalViews(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	var views int64
	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select("COALESCE(SUM((cc.metrics->>'views')::int), 0)").
		Where("cc.metrics IS NOT NULL")

	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	if err := query.Scan(&views).Error; err != nil {
		zap.L().Error("Failed to get total views", zap.Error(err))
		return 0, err
	}
	return views, nil
}

// GetTotalEngagements returns total engagements (likes + comments + shares) from metrics JSONB
func (r *contentStaffAnalyticsRepository) GetTotalEngagements(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	var engagements int64
	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select("COALESCE(SUM(COALESCE((cc.metrics->>'likes')::int, 0) + COALESCE((cc.metrics->>'comments')::int, 0) + COALESCE((cc.metrics->>'shares')::int, 0)), 0)").
		Where("cc.metrics IS NOT NULL")

	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	if err := query.Scan(&engagements).Error; err != nil {
		zap.L().Error("Failed to get total engagements", zap.Error(err))
		return 0, err
	}
	return engagements, nil
}

// GetTotalClicks returns total clicks from affiliate links
func (r *contentStaffAnalyticsRepository) GetTotalClicks(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	var clicks int64
	query := r.db.WithContext(ctx).Table("click_events ce").
		Select("COUNT(*)")

	if startDate != nil {
		query = query.Where("ce.clicked_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("ce.clicked_at <= ?", *endDate)
	}

	if err := query.Scan(&clicks).Error; err != nil {
		zap.L().Error("Failed to get total clicks", zap.Error(err))
		return 0, err
	}
	return clicks, nil
}

// GetMetricsByPlatform returns metrics aggregated by platform (channel name)
func (r *contentStaffAnalyticsRepository) GetMetricsByPlatform(ctx context.Context, startDate, endDate *time.Time) ([]dtos.PlatformMetricsResult, error) {
	var results []dtos.PlatformMetricsResult

	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select(`
			ch.name as platform,
			COUNT(DISTINCT cc.content_id) as content_count,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as total_views,
			COALESCE(SUM((cc.metrics->>'likes')::int), 0) as total_likes,
			COALESCE(SUM((cc.metrics->>'comments')::int), 0) as total_comments,
			COALESCE(SUM((cc.metrics->>'shares')::int), 0) as total_shares,
			0 as total_clicks,
			CASE WHEN SUM((cc.metrics->>'views')::int) > 0 
				THEN (SUM(COALESCE((cc.metrics->>'likes')::int, 0) + COALESCE((cc.metrics->>'comments')::int, 0) + COALESCE((cc.metrics->>'shares')::int, 0))::float / SUM((cc.metrics->>'views')::int)::float * 100)
				ELSE 0 
			END as engagement_rate
		`).
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Where("cc.metrics IS NOT NULL")

	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	query = query.Group("ch.name").Order("total_views DESC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get metrics by platform", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetTopContentByViews returns top content by views
func (r *contentStaffAnalyticsRepository) GetTopContentByViews(ctx context.Context, platform *string, limit int, startDate, endDate *time.Time) ([]dtos.ContentMetricsResult, error) {
	var results []dtos.ContentMetricsResult

	postedStatus := enum.AutoPostStatusPosted.String()

	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select(`
			c.id as content_id,
			c.title,
			ch.name as platform,
			ch.name as channel_name,
			cmp.name as campaign_name,
			COALESCE((cc.metrics->>'views')::int, 0) as views,
			COALESCE((cc.metrics->>'likes')::int, 0) as likes,
			COALESCE((cc.metrics->>'comments')::int, 0) as comments,
			COALESCE((cc.metrics->>'shares')::int, 0) as shares,
			0 as clicks,
			CASE WHEN (cc.metrics->>'views')::int > 0 
				THEN ((COALESCE((cc.metrics->>'likes')::int, 0) + COALESCE((cc.metrics->>'comments')::int, 0) + COALESCE((cc.metrics->>'shares')::int, 0))::float / (cc.metrics->>'views')::int::float * 100)
				ELSE 0 
			END as engagement_rate,
			cc.published_at as posted_at
		`).
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Joins("LEFT JOIN tasks t ON t.id = c.task_id").
		Joins("LEFT JOIN milestones m ON m.id = t.milestone_id").
		Joins("LEFT JOIN campaigns cmp ON cmp.id = m.campaign_id").
		Where("cc.metrics IS NOT NULL").
		Where("cc.auto_post_status = ?", postedStatus)

	if platform != nil && *platform != "" {
		query = query.Where("ch.name = ?", *platform)
	}
	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	query = query.Order("views DESC").Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get top content by views", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetTopChannelsByEngagement returns top channels by engagement
func (r *contentStaffAnalyticsRepository) GetTopChannelsByEngagement(ctx context.Context, limit int, startDate, endDate *time.Time) ([]dtos.ChannelMetricsResult, error) {
	var results []dtos.ChannelMetricsResult

	postedStatus := enum.AutoPostStatusPosted.String()

	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select(`
			ch.id as channel_id,
			ch.name as channel_name,
			ch.name as platform,
			'' as owner_name,
			COUNT(DISTINCT cc.content_id) as content_count,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as total_views,
			COALESCE(SUM((cc.metrics->>'likes')::int), 0) as total_likes,
			COALESCE(SUM((cc.metrics->>'comments')::int), 0) as total_comments,
			COALESCE(SUM((cc.metrics->>'shares')::int), 0) as total_shares,
			COALESCE(SUM((cc.metrics->>'likes')::int) + SUM((cc.metrics->>'comments')::int) + SUM((cc.metrics->>'shares')::int), 0) as total_engagements,
			CASE WHEN SUM((cc.metrics->>'views')::int) > 0 
				THEN (SUM(COALESCE((cc.metrics->>'likes')::int, 0) + COALESCE((cc.metrics->>'comments')::int, 0) + COALESCE((cc.metrics->>'shares')::int, 0))::float / SUM((cc.metrics->>'views')::int)::float * 100)
				ELSE 0 
			END as engagement_rate
		`).
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Where("cc.metrics IS NOT NULL").
		Where("cc.auto_post_status = ?", postedStatus)

	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	query = query.Group("ch.id, ch.name").
		Order("total_engagements DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get top channels by engagement", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetEngagementTrend returns engagement trend over time
func (r *contentStaffAnalyticsRepository) GetEngagementTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.EngagementTrendResult, error) {
	var results []dtos.EngagementTrendResult

	timeBucket := "date_trunc('day', cc.published_at)"
	switch granularity {
	case "WEEK":
		timeBucket = "date_trunc('week', cc.published_at)"
	case "MONTH":
		timeBucket = "date_trunc('month', cc.published_at)"
	}

	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select(`
			` + timeBucket + ` as date,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as views,
			COALESCE(SUM((cc.metrics->>'likes')::int), 0) as likes,
			COALESCE(SUM((cc.metrics->>'comments')::int), 0) as comments,
			COALESCE(SUM((cc.metrics->>'shares')::int), 0) as shares,
			COALESCE(SUM((cc.metrics->>'likes')::int) + SUM((cc.metrics->>'comments')::int) + SUM((cc.metrics->>'shares')::int), 0) as total_engagements,
			CASE WHEN SUM((cc.metrics->>'views')::int) > 0 
				THEN (SUM(COALESCE((cc.metrics->>'likes')::int, 0) + COALESCE((cc.metrics->>'comments')::int, 0) + COALESCE((cc.metrics->>'shares')::int, 0))::float / SUM((cc.metrics->>'views')::int)::float * 100)
				ELSE 0 
			END as engagement_rate
		`).
		Where("cc.metrics IS NOT NULL").
		Where("cc.published_at IS NOT NULL")

	if startDate != nil {
		query = query.Where("cc.published_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cc.published_at <= ?", *endDate)
	}

	query = query.Group(timeBucket).Order("date ASC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get engagement trend", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetRecentContent returns recent content
func (r *contentStaffAnalyticsRepository) GetRecentContent(ctx context.Context, limit int) ([]dtos.RecentContentResult, error) {
	var results []dtos.RecentContentResult

	query := r.db.WithContext(ctx).Table("contents c").
		Select(`
			c.id as content_id,
			c.title,
			c.status,
			cmp.name as campaign_name,
			'' as creator_name,
			c.created_at,
			c.updated_at
		`).
		Joins("LEFT JOIN tasks t ON t.id = c.task_id").
		Joins("LEFT JOIN milestones m ON m.id = t.milestone_id").
		Joins("LEFT JOIN campaigns cmp ON cmp.id = m.campaign_id").
		Where("c.deleted_at IS NULL").
		Order("c.created_at DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get recent content", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetContentStatusBreakdown returns content counts by status
func (r *contentStaffAnalyticsRepository) GetContentStatusBreakdown(ctx context.Context, startDate, endDate *time.Time) ([]dtos.ContentStatusCount, error) {
	var results []dtos.ContentStatusCount

	query := r.db.WithContext(ctx).Table("contents").
		Select("status, COUNT(*) as count").
		Where("deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	query = query.Group("status")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get content status breakdown", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetCampaignContentMetrics returns content metrics by campaign
func (r *contentStaffAnalyticsRepository) GetCampaignContentMetrics(ctx context.Context, campaignID *uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.CampaignContentMetrics, error) {
	var results []dtos.CampaignContentMetrics

	// Use enum values for content statuses
	postedStatus := enum.ContentStatusPosted.String()
	awaitStaffStatus := enum.ContentStatusAwaitStaff.String()
	awaitBrandStatus := enum.ContentStatusAwaitBrand.String()
	draftStatus := enum.ContentStatusDraft.String()

	query := r.db.WithContext(ctx).Table("campaigns cmp").
		Select(`
			cmp.id as campaign_id,
			cmp.name as campaign_name,
			COUNT(DISTINCT c.id) as content_count,
			SUM(CASE WHEN c.status = ? THEN 1 ELSE 0 END) as posted_count,
			SUM(CASE WHEN c.status IN (?, ?) THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN c.status = ? THEN 1 ELSE 0 END) as draft_count,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as total_views,
			COALESCE(SUM((cc.metrics->>'likes')::int) + SUM((cc.metrics->>'comments')::int) + SUM((cc.metrics->>'shares')::int), 0) as total_engagements
		`, postedStatus, awaitStaffStatus, awaitBrandStatus, draftStatus).
		Joins("JOIN milestones m ON m.campaign_id = cmp.id").
		Joins("LEFT JOIN tasks t ON t.milestone_id = m.id").
		Joins("LEFT JOIN contents c ON c.task_id = t.id").
		Joins("LEFT JOIN content_channels cc ON cc.content_id = c.id").
		Where("cmp.deleted_at IS NULL")

	if campaignID != nil {
		query = query.Where("cmp.id = ?", *campaignID)
	}
	if startDate != nil {
		query = query.Where("cmp.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cmp.created_at <= ?", *endDate)
	}

	query = query.Group("cmp.id, cmp.name").
		Order("total_views DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get campaign content metrics", zap.Error(err))
		return nil, err
	}
	return results, nil
}
