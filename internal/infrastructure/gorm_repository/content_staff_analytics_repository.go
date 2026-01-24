package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
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

// GetPostCountByDateRange returns the count of posted content in a date range
func (r *contentStaffAnalyticsRepository) GetPostCountByDateRange(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select("COUNT(DISTINCT cc.content_id)").
		Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
		Where("cc.published_at >= ?", startDate).
		Where("cc.published_at < ?", endDate)

	if channelID != nil {
		query = query.Where("cc.channel_id = ?", *channelID)
	}

	if err := query.Scan(&count).Error; err != nil {
		zap.L().Error("Failed to get post count by date range", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetTotalViews returns total views gained during the period
// - Website Channels (Incremental Events): SUM of all view events
// - Social Channels (Cumulative Snapshots): delta = latest_value - earliest_value per content_channel
func (r *contentStaffAnalyticsRepository) GetTotalViews(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error) {
	var totalViews int64

	// Build unified query that handles both Website (SUM) and Social (delta) in one pass
	channelFilter := ""
	args := []any{
		enum.KPIReferenceTypeContentChannel,
		enum.KPIValueTypeViews,
		enum.AutoPostStatusPosted,
		startDate, endDate,
		startDate, endDate,
	}

	if channelID != nil {
		channelFilter = "AND cc.channel_id = ?"
		args = append(args, *channelID)
	}

	query := fmt.Sprintf(`
		WITH range_views AS (
			SELECT 
				km.reference_id,
				ch.code,
				km.value,
				ROW_NUMBER() OVER (PARTITION BY km.reference_id ORDER BY km.recorded_date ASC) as rn_first,
				ROW_NUMBER() OVER (PARTITION BY km.reference_id ORDER BY km.recorded_date DESC) as rn_last
			FROM kpi_metrics km
			JOIN content_channels cc ON cc.id = km.reference_id
			JOIN channels ch ON ch.id = cc.channel_id
			WHERE km.reference_type = ?
			  AND km.type = ?
			  AND cc.auto_post_status = ?
			  AND km.recorded_date >= ? AND km.recorded_date < ?
			  AND cc.published_at >= ? AND cc.published_at < ?
			  %s
		),
		per_content_channel AS (
			SELECT 
				reference_id,
				code,
				MAX(CASE WHEN rn_first = 1 THEN value END) as first_val,
				MAX(CASE WHEN rn_last = 1 THEN value END) as last_val,
				SUM(value) as sum_val
			FROM range_views
			GROUP BY reference_id, code
		)
		SELECT COALESCE(SUM(
			CASE 
				-- Website: Incremental events, use SUM
				WHEN code = 'WEBSITE' THEN sum_val
				-- Social: Cumulative snapshots, use delta (last - first)
				ELSE GREATEST(COALESCE(last_val, 0) - COALESCE(first_val, 0), 0)
			END
		), 0)::BIGINT as total
		FROM per_content_channel
	`, channelFilter)

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&totalViews).Error; err != nil {
		zap.L().Error("Failed to get total views from kpi_metrics", zap.Error(err))
		return 0, err
	}
	return totalViews, nil
}

// GetTotalEngagement returns total engagement gained during the period
// Engagement = Likes + Comments + Shares (delta for each)
// All engagement metrics are cumulative snapshots, so we calculate delta = last - first
func (r *contentStaffAnalyticsRepository) GetTotalEngagement(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (int64, error) {
	var totalEngagement int64

	// Build unified query that handles both Website (SUM) and Social (delta) in one pass
	channelFilter := ""
	args := []any{
		enum.KPIReferenceTypeContentChannel,
		enum.KPIValueTypeEngagement,
		enum.AutoPostStatusPosted,
		startDate, endDate,
		startDate, endDate,
	}

	if channelID != nil {
		channelFilter = "AND cc.channel_id = ?"
		args = append(args, *channelID)
	}

	query := fmt.Sprintf(`
		WITH range_views AS (
			SELECT 
				km.reference_id,
				ch.code,
				km.value,
				ROW_NUMBER() OVER (PARTITION BY km.reference_id ORDER BY km.recorded_date ASC) as rn_first,
				ROW_NUMBER() OVER (PARTITION BY km.reference_id ORDER BY km.recorded_date DESC) as rn_last
			FROM kpi_metrics km
			JOIN content_channels cc ON cc.id = km.reference_id
			JOIN channels ch ON ch.id = cc.channel_id
			WHERE km.reference_type = ?
			  AND km.type = ?
			  AND cc.auto_post_status = ?
			  AND km.recorded_date >= ? AND km.recorded_date < ?
			  AND cc.published_at >= ? AND cc.published_at < ?
			  %s
		),
		per_content_channel AS (
			SELECT 
				reference_id,
				code,
				MAX(CASE WHEN rn_first = 1 THEN value END) as first_val,
				MAX(CASE WHEN rn_last = 1 THEN value END) as last_val,
				SUM(value) as sum_val
			FROM range_views
			GROUP BY reference_id, code
		)
		SELECT COALESCE(SUM(GREATEST(COALESCE(last_val, 0) - COALESCE(first_val, 0), 0)), 0)::BIGINT as total
		FROM per_content_channel
	`, channelFilter)

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&totalEngagement).Error; err != nil {
		zap.L().Error("Failed to get total engagement from kpi_metrics", zap.Error(err))
		return 0, err
	}
	return totalEngagement, nil

	// // Get latest ENGAGEMENT (or LIKES+COMMENTS+SHARES) per content_channel
	// subquery := `
	// 	SELECT DISTINCT ON (km.reference_id, km.type) km.reference_id, km.type, km.value
	// 	FROM kpi_metrics km
	// 	JOIN content_channels cc ON cc.id = km.reference_id
	// 	WHERE km.reference_type = ?
	// 	  AND km.type IN ?
	// 	  AND km.recorded_date >= ?
	// 	  AND km.recorded_date < ?
	// 	  AND cc.auto_post_status = ?
	// 	  AND cc.published_at >= ?
	// 	  AND cc.published_at < ?
	// `
	// kpiMetricsType := []enum.KPIValueType{
	// 	enum.KPIValueTypeEngagement,
	// 	// enum.KPIValueTypeLikes,
	// 	// enum.KPIValueTypeComments,
	// 	// enum.KPIValueTypeShares,
	// }
	// args := []any{
	// 	enum.KPIReferenceTypeContentChannel,
	// 	kpiMetricsType,
	// 	startDate,
	// 	endDate,
	// 	enum.AutoPostStatusPosted.String(),
	// 	startDate,
	// 	endDate,
	// }

	// if channelID != nil {
	// 	subquery += " AND cc.channel_id = ?"
	// 	args = append(args, *channelID)
	// }

	// subquery += " ORDER BY km.reference_id, km.type, km.recorded_date DESC"

	// query := r.db.WithContext(ctx).Raw(`
	// 	SELECT COALESCE(SUM(latest.value), 0) as total
	// 	FROM (`+subquery+`) AS latest
	// `, args...)

	// if err := query.Scan(&engagement).Error; err != nil {
	// 	zap.L().Error("Failed to get total engagement from kpi_metrics", zap.Error(err))
	// 	return 0, err
	// }
	// return int64(engagement), nil
	// var totalEngagement int64

	// engagementQuery := `
	//  	WITH
	// `
}

// GetAverageCTR returns CTR using latest CLICK_THROUGH and VIEWS values
func (r *contentStaffAnalyticsRepository) GetAverageCTR(ctx context.Context, startDate, endDate time.Time, channelID *uuid.UUID) (float64, error) {
	// Get total clicks from affiliate links (Incremental data -> SUM)
	var totalClicks float64

	clicksQuery := `
		SELECT COALESCE(SUM(km.value), 0)
		FROM kpi_metrics km
		JOIN affiliate_links al ON al.id = km.reference_id
		WHERE km.reference_type = ?
		  AND km.type = ?
		  AND km.recorded_date >= ?
		  AND km.recorded_date < ?
	`
	clicksArgs := []any{
		enum.KPIReferenceTypeAffiliateLink,
		enum.KPIValueTypeClickThrough,
		startDate,
		endDate,
	}

	if channelID != nil {
		clicksQuery += " AND (al.metadata ->> 'channel_id') = ?"
		clicksArgs = append(clicksArgs, *channelID)
	}

	if err := r.db.WithContext(ctx).Raw(clicksQuery, clicksArgs...).Scan(&totalClicks).Error; err != nil {
		zap.L().Error("Failed to get total clicks from kpi_metrics", zap.Error(err))
		return 0, err
	}

	// Get latest total views (uses fixed GetTotalViews logic)
	totalViews, err := r.GetTotalViews(ctx, startDate, endDate, channelID)
	if err != nil {
		return 0, err
	}

	if totalViews == 0 {
		return 0, nil
	}

	return (totalClicks / float64(totalViews)) * 100, nil
}

// GetPendingContentCount returns count of content in pending statuses
func (r *contentStaffAnalyticsRepository) GetPendingContentCount(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var count int64

	pendingStatuses := []string{
		enum.ContentStatusDraft.String(),
		enum.ContentStatusAwaitStaff.String(),
		enum.ContentStatusAwaitBrand.String(),
	}

	if err := r.db.WithContext(ctx).Table("contents").
		Where("deleted_at IS NULL").
		Where("status IN ?", pendingStatuses).
		Where("created_at >= ?", startDate).
		Where("created_at < ?", endDate).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get pending content count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetChannelMetrics returns the DELTA metrics (metrics gained during the period) for each channel
// Uses pre-aggregated CHANNEL-level metrics from ContentMetricsPollerJob
// For cumulative metrics (Social): delta = latest_value - earliest_value in range
// For incremental metrics (Website views): SUM of values in range
func (r *contentStaffAnalyticsRepository) GetChannelMetrics(ctx context.Context, startDate, endDate time.Time) ([]dtos.ChannelMetricsDTO, error) {
	var results []dtos.ChannelMetricsDTO

	// 1. Get channel info
	// Note: We don't need to fetch 'metrics' column here if we are calculating fresh deltas below
	channelQuery := r.db.WithContext(ctx).Table("channels ch").
		Select(`
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			COALESCE((ch.metrics->'current_mapped'->>'FOLLOWERS')::bigint, 0) as followers_count
		`).
		Where("ch.deleted_at IS NULL")

	var channels []dtos.ChannelMetricsDTO
	if err := channelQuery.Scan(&channels).Error; err != nil {
		zap.L().Error("Failed to get channels", zap.Error(err))
		return nil, err
	}

	// 2. Loop channels
	for i := range channels {
		channelID := channels[i].ChannelID

		// A. Get post count (For NEW posts in this period)
		var postCount int64
		if err := r.db.WithContext(ctx).Table("content_channels cc").
			Where("cc.channel_id = ?", channelID).
			Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
			Where("cc.published_at >= ? AND cc.published_at < ?", startDate, endDate).
			Count(&postCount).Error; err != nil {
			// Log but don't fail entire request
			zap.L().Warn("Failed to get post count", zap.String("channel_id", channelID.String()), zap.Error(err))
		}
		channels[i].PostCount = postCount

		// B. Get Metrics Deltas
		// Unified Query: Handles both Incremental (Website) and Cumulative (Social)
		type MetricDelta struct {
			Type  enum.KPIValueType
			Delta float64
		}
		var metrics []MetricDelta

		deltaQuery := `
			WITH range_metrics AS (
				SELECT
					km.reference_id,
					km.type,
					km.value,
					ch.code, 
					-- Get First and Last value recorded INSIDE the window
					ROW_NUMBER() OVER (PARTITION BY km.reference_id, km.type ORDER BY km.recorded_date ASC) as rn_first,
					ROW_NUMBER() OVER (PARTITION BY km.reference_id, km.type ORDER BY km.recorded_date DESC) as rn_last
				FROM kpi_metrics km
				JOIN content_channels cc ON cc.id = km.reference_id
				JOIN channels ch ON ch.id = cc.channel_id
				WHERE cc.channel_id = ?
				  AND km.reference_type = ?
				  AND cc.auto_post_status = ?
				  AND cc.published_at >= ? AND cc.published_at < ?
				  AND km.recorded_date >= ? AND km.recorded_date < ?
			),
			first_last AS (
				SELECT
					reference_id,
					type,
					code,
					MAX(CASE WHEN rn_first = 1 THEN value END) as first_val,
					MAX(CASE WHEN rn_last = 1 THEN value END) as last_val,
					SUM(value) as sum_val
				FROM range_metrics
				GROUP BY reference_id, type, code
			)
			SELECT
				type,
				SUM(
					CASE
						-- Website Views/UniqueViews are incremental (1 row = 1 view), use SUM
						WHEN (type IN ('VIEWS', 'UNIQUE_VIEWS') AND code = 'WEBSITE') THEN sum_val
						
						-- All other metrics are cumulative snapshots (Total Counter), use Delta
						-- GREATEST(..., 0) ensures if a metric resets/drops, we don't show negative growth
						ELSE GREATEST(last_val - first_val, 0)
					END
				) as delta
			FROM first_last
			GROUP BY type
		`

		// Fixed Arguments: Removed the extra startDate/endDate pair
		if err := r.db.WithContext(ctx).Raw(deltaQuery,
			channelID,
			enum.KPIReferenceTypeContentChannel,
			enum.AutoPostStatusPosted,
			startDate, endDate,
			startDate, endDate,
		).Scan(&metrics).Error; err != nil {
			zap.L().Warn("Failed to get channel delta metrics", zap.String("channel_id", channelID.String()), zap.Error(err))
			continue
		}

		// Map metrics
		for _, m := range metrics {
			val := int64(m.Delta)
			switch m.Type {
			case enum.KPIValueTypeViews:
				channels[i].TotalViews = val
			case enum.KPIValueTypeLikes:
				channels[i].TotalLikes = val
			case enum.KPIValueTypeComments:
				channels[i].TotalComments = val
			case enum.KPIValueTypeShares:
				channels[i].TotalShares = val
			case enum.KPIValueTypeEngagement:
				channels[i].TotalEngagement = val
			case enum.KPIValueTypeReach:
				channels[i].TotalReach = val
			}
		}

		// C. Get Affiliate Clicks (Incremental -> SUM)
		// Note: ensure metadata key matches your DB exactly
		var clicks float64
		err := r.db.WithContext(ctx).Raw(`
			SELECT COALESCE(SUM(km.value), 0)
			FROM kpi_metrics km
			JOIN affiliate_links al ON al.id = km.reference_id
			WHERE ((al.metadata ->> 'channel_id') = ? OR al.channel_id = ?)
			  AND km.reference_type = ?
			  AND km.type = ?
			  AND km.recorded_date >= ?
			  AND km.recorded_date < ?
		`, channelID.String(), channelID.String(),
			enum.KPIReferenceTypeAffiliateLink, enum.KPIValueTypeClickThrough,
			startDate, endDate,
		).Scan(&clicks).Error

		if err != nil {
			zap.L().Warn("Failed to get affiliate clicks", zap.Error(err))
		}

		channels[i].TotalClicks = int64(clicks)

		// D. Calculate CTR
		// Avoid division by zero
		if channels[i].TotalViews > 0 {
			channels[i].AverageCTR = (float64(channels[i].TotalClicks) / float64(channels[i].TotalViews)) * 100
		}

		results = append(results, channels[i])
	}

	return results, nil
}

// GetTopPostForChannel returns the top performing post for a specific channel
func (r *contentStaffAnalyticsRepository) GetTopPostForChannel(ctx context.Context, channelID uuid.UUID, startDate, endDate time.Time) (*dtos.TopPostDTO, error) {
	// Get channel code
	var channelCode string
	if err := r.db.WithContext(ctx).Table("channels").Select("code").Where("id = ?", channelID).Scan(&channelCode).Error; err != nil {
		return nil, err
	}

	// Get content channels for this channel in date range with their metrics
	type ContentWithMetrics struct {
		ContentChannelID uuid.UUID
		ContentID        uuid.UUID
		Title            string
		Views            int64
		Likes            int64
		Comments         int64
		Shares           int64
	}

	var query string
	var args []any

	if channelCode == "WEBSITE" {
		// Website: Views are incremental (SUM), Engagement is cumulative (MAX/Latest)
		// Actually, Engagement for Website is also cumulative snapshots in kpi_metrics (from Poller).
		// Only Views are incremental (from Consumer).

		query = `
			WITH metrics_agg AS (
				SELECT 
					km.reference_id,
					-- 1. Views Type, 2. Likes Type, 3. Comments Type, 4. Shares Type
					SUM(CASE WHEN km.type = ? THEN km.value ELSE 0 END) as views,
					MAX(CASE WHEN km.type = ? THEN km.value ELSE 0 END) as likes,
					MAX(CASE WHEN km.type = ? THEN km.value ELSE 0 END) as comments,
					MAX(CASE WHEN km.type = ? THEN km.value ELSE 0 END) as shares
				FROM kpi_metrics km
				JOIN content_channels cc ON cc.id = km.reference_id
				WHERE cc.channel_id = ?        -- 5. Channel ID
				  AND km.reference_type = ?    -- 6. Ref Type
				  AND cc.auto_post_status = ?  -- 7. Status
				  AND cc.published_at >= ?     -- 8. Start Date
				  AND cc.published_at < ?      -- 9. End Date
				  AND km.recorded_date >= ?    -- 10. Start Date
				  AND km.recorded_date < ?     -- 11. End Date
				GROUP BY km.reference_id
			)
			SELECT 
				cc.id as content_channel_id,
				cc.content_id,
				c.title,
				CAST(COALESCE(ma.views, 0) AS BIGINT) as views,
				CAST(COALESCE(ma.likes, 0) AS BIGINT) as likes,
				CAST(COALESCE(ma.comments, 0) AS BIGINT) as comments,
				CAST(COALESCE(ma.shares, 0) AS BIGINT) as shares
			FROM content_channels cc
			JOIN contents c ON c.id = cc.content_id
			LEFT JOIN metrics_agg ma ON ma.reference_id = cc.id
			WHERE cc.channel_id = ?            -- 12. Channel ID
			  AND cc.auto_post_status = ?      -- 13. Status
			  AND cc.published_at >= ?         -- 14. Start Date
			  AND cc.published_at < ?          -- 15. End Date
			  AND c.deleted_at IS NULL
			ORDER BY (COALESCE(ma.views, 0) + COALESCE(ma.likes, 0) * 2 + COALESCE(ma.comments, 0) * 3 + COALESCE(ma.shares, 0) * 4) DESC
			LIMIT 1
		`
		// Ensure this list matches the comments above EXACTLY
		args = []any{
			enum.KPIValueTypeViews,              // 1
			enum.KPIValueTypeLikes,              // 2
			enum.KPIValueTypeComments,           // 3
			enum.KPIValueTypeShares,             // 4
			channelID,                           // 5
			enum.KPIReferenceTypeContentChannel, // 6
			enum.AutoPostStatusPosted.String(),  // 7
			startDate, endDate,                  // 8, 9
			startDate, endDate, // 10, 11
			channelID,                          // 12
			enum.AutoPostStatusPosted.String(), // 13
			startDate, endDate,                 // 14, 15
		}
	} else {
		// Social: Cumulative Snapshots (DISTINCT ON) for everything
		query = `
			WITH latest_metrics AS (
				SELECT DISTINCT ON (km.reference_id, km.type) 
					km.reference_id, km.type, km.value
				FROM kpi_metrics km
				JOIN content_channels cc ON cc.id = km.reference_id
				WHERE cc.channel_id = ?
				  AND km.reference_type = ?
				  AND cc.auto_post_status = ?
				  AND cc.published_at >= ?
				  AND cc.published_at < ?
				  AND km.recorded_date >= ?
				  AND km.recorded_date < ?
				ORDER BY km.reference_id, km.type, km.recorded_date DESC
			),
			pivoted_metrics AS (
				SELECT 
					reference_id,
					CAST(COALESCE(MAX(CASE WHEN type = ? THEN value END), 0) AS BIGINT) as views,
					CAST(COALESCE(MAX(CASE WHEN type = ? THEN value END), 0) AS BIGINT) as likes,
					CAST(COALESCE(MAX(CASE WHEN type = ? THEN value END), 0) AS BIGINT) as comments,
					CAST(COALESCE(MAX(CASE WHEN type = ? THEN value END), 0) AS BIGINT) as shares
				FROM latest_metrics
				GROUP BY reference_id
			)
			SELECT 
				cc.id as content_channel_id,
				cc.content_id,
				c.title,
				CAST(COALESCE(pm.views, 0) AS BIGINT) as views,
				CAST(COALESCE(pm.likes, 0) AS BIGINT) as likes,
				CAST(COALESCE(pm.comments, 0) AS BIGINT) as comments,
				CAST(COALESCE(pm.shares, 0) AS BIGINT) as shares
			FROM content_channels cc
			JOIN contents c ON c.id = cc.content_id
			LEFT JOIN pivoted_metrics pm ON pm.reference_id = cc.id
			WHERE cc.channel_id = ?
			  AND cc.auto_post_status = ?
			  AND cc.published_at >= ?
			  AND cc.published_at < ?
			  AND c.deleted_at IS NULL
			ORDER BY (COALESCE(pm.views, 0) + COALESCE(pm.likes, 0) * 2 + COALESCE(pm.comments, 0) * 3 + COALESCE(pm.shares, 0) * 4) DESC
			LIMIT 1
		`
		args = []any{
			channelID,
			enum.KPIReferenceTypeContentChannel,
			enum.AutoPostStatusPosted.String(),
			startDate, endDate,
			startDate, endDate,
			enum.KPIValueTypeViews,
			enum.KPIValueTypeLikes,
			enum.KPIValueTypeComments,
			enum.KPIValueTypeShares,
			channelID,
			enum.AutoPostStatusPosted.String(),
			startDate, endDate,
		}
	}

	var result ContentWithMetrics
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error

	if err != nil || result.ContentID == uuid.Nil {
		return nil, err
	}

	return &dtos.TopPostDTO{
		ContentID: result.ContentID,
		Title:     result.Title,
		Views:     result.Views,
		Likes:     result.Likes,
		Comments:  result.Comments,
		Shares:    result.Shares,
	}, nil
}

// GetTrendData returns time series data for trend charts from kpi_metrics
// This returns DELTA values per time bucket to show the growth/change over time
// - Website Views/UniqueViews: Incremental (each row = 1 view), use SUM per bucket
// - All other metrics: Cumulative snapshots, use MAX per bucket then calculate delta with LAG
func (r *contentStaffAnalyticsRepository) GetTrendData(ctx context.Context, startDate, endDate time.Time, granularity constant.TrendGranularity, channelID *uuid.UUID) ([]dtos.TrendDataPointDTO, error) {
	var results []dtos.TrendDataPointDTO

	// 1. Determine Lookback Interval and SQL Bucket logic
	// We need to fetch data from BEFORE the startDate to calculate the delta for the first bucket.
	var lookbackInterval time.Duration
	var timeBucketExpr string

	switch granularity {
	case constant.TrendGranularityWeek:
		timeBucketExpr = "time_bucket('1 week', km.recorded_date)"
		lookbackInterval = 7 * 24 * time.Hour
	case constant.TrendGranularityMonth:
		// NOTE: time_bucket doesn't handle variable '1 month'.
		// For accuracy with calendar months, prefer date_trunc over time_bucket('30 days').
		timeBucketExpr = "date_trunc('month', km.recorded_date)"
		lookbackInterval = 32 * 24 * time.Hour // Safe buffer for a month
	default: // Day
		timeBucketExpr = "time_bucket('1 day', km.recorded_date)"
		lookbackInterval = 24 * time.Hour
	}

	// Adjust the query start date to include the previous bucket
	queryStartDate := startDate.Add(-lookbackInterval)

	query := `
		WITH per_reference_metrics AS (
			SELECT
				` + timeBucketExpr + ` as bucket,
				km.type,
				ch.code,
				km.reference_id,
				SUM(km.value) as sum_val,
				MAX(km.value) as max_val
			FROM kpi_metrics km
			JOIN content_channels cc ON cc.id = km.reference_id
			JOIN channels ch ON ch.id = cc.channel_id
			WHERE km.reference_type = ?
			  AND km.recorded_date >= ?  -- Uses queryStartDate
			  AND km.recorded_date < ?
			  AND cc.auto_post_status = ?
			  AND cc.published_at >= ? AND cc.published_at < ?
	`
	args := []any{
		enum.KPIReferenceTypeContentChannel,
		queryStartDate, // Adjusted start date
		endDate,
		enum.AutoPostStatusPosted.String(),
		startDate, endDate,
	}

	if channelID != nil {
		query += " AND cc.channel_id = ?"
		args = append(args, *channelID)
	}

	query += `
			GROUP BY bucket, km.type, ch.code, km.reference_id
		),
		with_deltas AS (
			SELECT
				bucket,
				type,
				code,
				sum_val,
				-- Calculate Delta
				max_val - LAG(max_val) OVER (
					PARTITION BY reference_id, type 
					ORDER BY bucket
				) as delta_val
			FROM per_reference_metrics
		)
		SELECT
			bucket as date,
			type,
			SUM(
				CASE 
					-- Incremental metrics (WEBSITE views): Use the SUM of the specific bucket
					WHEN (type = ? OR type = ?) AND code = 'WEBSITE' THEN sum_val
					
					-- Cumulative metrics: Use the calculated Delta
					-- If delta is NULL (which happens for the very first lookback bucket), treat as 0
					-- GREATEST handles negative deltas (resets)
					ELSE GREATEST(COALESCE(delta_val, 0), 0)
				END
			) as total
		FROM with_deltas
		-- CRITICAL: Filter out the lookback bucket here so the user only sees requested range
		WHERE bucket >= ? 
		GROUP BY bucket, type
		ORDER BY bucket ASC
	`

	// Add args for CASE Types
	args = append(args, enum.KPIValueTypeViews, enum.KPIValueTypeUniqueViews)
	// Add args for Final Where Clause (Original Start Date)
	args = append(args, startDate)

	type RawTrendData struct {
		Date  time.Time
		Type  enum.KPIValueType
		Total float64
	}
	var rawData []RawTrendData
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawData).Error; err != nil {
		zap.L().Error("Failed to get trend data from kpi_metrics", zap.Error(err))
		return nil, err
	}

	// Pivot the data: group by date, spread metrics into columns
	dateMap := make(map[time.Time]*dtos.TrendDataPointDTO)
	for _, row := range rawData {
		if _, exists := dateMap[row.Date]; !exists {
			dateMap[row.Date] = &dtos.TrendDataPointDTO{Date: row.Date}
		}
		dp := dateMap[row.Date]
		switch row.Type {
		case enum.KPIValueTypeViews:
			dp.Views = int64(row.Total)
		case enum.KPIValueTypeLikes:
			dp.Likes = int64(row.Total)
		case enum.KPIValueTypeComments:
			dp.Comments = int64(row.Total)
		case enum.KPIValueTypeShares:
			dp.Shares = int64(row.Total)
		case enum.KPIValueTypeEngagement:
			dp.Engagements = int64(row.Total)
		}
	}

	// Convert map to slice and calculate engagements if not set
	for _, dp := range dateMap {
		if dp.Engagements == 0 {
			dp.Engagements = dp.Likes + dp.Comments + dp.Shares
		}
		results = append(results, *dp)
	}

	// Sort results by date ascending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Date.Before(results[i].Date) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// GetContentTypeDistribution returns content count by type
func (r *contentStaffAnalyticsRepository) GetContentTypeDistribution(ctx context.Context, startDate, endDate time.Time) ([]dtos.ContentTypeDistributionDTO, error) {
	var results []dtos.ContentTypeDistributionDTO
	var total int64

	// First get total count
	if err := r.db.WithContext(ctx).Table("contents c").
		Joins("JOIN content_channels cc ON cc.content_id = c.id").
		Where("c.deleted_at IS NULL").
		Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
		Where("cc.published_at >= ?", startDate).
		Where("cc.published_at < ?", endDate).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to get total content count for distribution", zap.Error(err))
		return nil, err
	}

	// Then get distribution
	query := r.db.WithContext(ctx).Table("contents c").
		Select(`
			c.type as content_type,
			COUNT(DISTINCT c.id) as count,
			CASE WHEN ? > 0 THEN (COUNT(DISTINCT c.id)::float / ?::float * 100) ELSE 0 END as percentage
		`, total, total).
		Joins("JOIN content_channels cc ON cc.content_id = c.id").
		Where("c.deleted_at IS NULL").
		Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
		Where("cc.published_at >= ?", startDate).
		Where("cc.published_at < ?", endDate).
		Group("c.type").
		Order("count DESC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get content type distribution", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetChannelDistribution returns content count by channel (for pie chart)
func (r *contentStaffAnalyticsRepository) GetChannelDistribution(ctx context.Context, startDate, endDate time.Time) ([]dtos.ChannelDistributionDTO, error) {
	var results []dtos.ChannelDistributionDTO
	var total int64

	// First get total count of posted content channels
	if err := r.db.WithContext(ctx).Table("content_channels cc").
		Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
		Where("cc.published_at >= ?", startDate).
		Where("cc.published_at < ?", endDate).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to get total content count for channel distribution", zap.Error(err))
		return nil, err
	}

	// Then get distribution by channel
	query := r.db.WithContext(ctx).Table("content_channels cc").
		Select(`
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			COUNT(DISTINCT cc.content_id) as count,
			CASE WHEN ? > 0 THEN (COUNT(DISTINCT cc.content_id)::float / ?::float * 100) ELSE 0 END as percentage
		`, total, total).
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Where("cc.auto_post_status = ?", enum.AutoPostStatusPosted.String()).
		Where("cc.published_at >= ?", startDate).
		Where("cc.published_at < ?", endDate).
		Group("ch.id, ch.name, ch.code").
		Order("count DESC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get channel distribution", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetTopContentByPerformance returns top performing content from kpi_metrics
func (r *contentStaffAnalyticsRepository) GetTopContentByPerformance(ctx context.Context, limit int, startDate, endDate time.Time, channelID *uuid.UUID) ([]dtos.ContentPerformanceDTO, error) {
	return r.getContentByPerformance(ctx, limit, startDate, endDate, channelID, "DESC")
}

// GetBottomContentByPerformance returns lowest performing content
func (r *contentStaffAnalyticsRepository) GetBottomContentByPerformance(ctx context.Context, limit int, startDate, endDate time.Time, channelID *uuid.UUID) ([]dtos.ContentPerformanceDTO, error) {
	return r.getContentByPerformance(ctx, limit, startDate, endDate, channelID, "ASC")
}

// getContentByPerformance is a helper method for getting content sorted by performance
// Uses DISTINCT ON to get LATEST metric values (not cumulative sums)
func (r *contentStaffAnalyticsRepository) getContentByPerformance(ctx context.Context, limit int, startDate, endDate time.Time, channelID *uuid.UUID, order string) ([]dtos.ContentPerformanceDTO, error) {
	// Build the SQL query using DISTINCT ON to get latest values
	// This uses a CTE to get the latest value for each content_channel + metric type combination
	channelFilter := ""
	args := []any{
		enum.KPIReferenceTypeContentChannel.String(),
		startDate, endDate,
		enum.AutoPostStatusPosted.String(),
		startDate, endDate,
		enum.KPIReferenceTypeAffiliateLink.String(),
		enum.KPIValueTypeClickThrough.String(),
		startDate, endDate,
	}

	if channelID != nil {
		channelFilter = "AND cc.channel_id = ?"
		// Insert channel ID at appropriate positions
		args = append(args[:6], append([]any{*channelID}, args[6:]...)...)
	}

	query := fmt.Sprintf(`
		WITH latest_metrics AS (
			-- Get latest value for each content_channel + metric type
			SELECT DISTINCT ON (km.reference_id, km.type)
				km.reference_id as content_channel_id,
				km.type,
				km.value
			FROM kpi_metrics km
			WHERE km.reference_type = ?
			  AND km.recorded_date >= ? AND km.recorded_date < ?
			ORDER BY km.reference_id, km.type, km.recorded_date DESC
		),
		pivoted_metrics AS (
			-- Pivot latest metrics into columns
			SELECT 
				content_channel_id,
				COALESCE(MAX(CASE WHEN type = 'VIEWS' THEN value END), 0) as views,
				COALESCE(MAX(CASE WHEN type = 'LIKES' THEN value END), 0) as likes,
				COALESCE(MAX(CASE WHEN type = 'COMMENTS' THEN value END), 0) as comments,
				COALESCE(MAX(CASE WHEN type = 'SHARES' THEN value END), 0) as shares
			FROM latest_metrics
			GROUP BY content_channel_id
		),
		content_info AS (
			-- Get content channel info
			SELECT 
				cc.id as content_channel_id,
				cc.content_id,
				c.title,
				c.type as content_type,
				ch.id as channel_id,
				ch.name as channel_name,
				cc.published_at,
				c.thumbnail_url
			FROM content_channels cc
			JOIN contents c ON c.id = cc.content_id
			JOIN channels ch ON ch.id = cc.channel_id
			WHERE cc.auto_post_status = ?
			  AND cc.published_at >= ? AND cc.published_at < ?
			  AND c.deleted_at IS NULL
			  %s
		),
		latest_clicks AS (
			SELECT SUM(km.value) as clicks,
			       (al.metadata ->> 'content_channel_id')::uuid as content_channel_id
			FROM kpi_metrics km
			JOIN affiliate_links al ON al.id = km.reference_id
			WHERE km.reference_type = ?
			  AND km.type = ?
			  AND km.recorded_date >= ? AND km.recorded_date < ?
			GROUP BY content_channel_id
		)
		SELECT 
			ci.content_id,
			ci.title,
			ci.content_type,
			ci.channel_id,
			ci.channel_name,
			ci.published_at,
			ci.thumbnail_url,
			COALESCE(pm.views, 0)::bigint as views,
			COALESCE(pm.likes, 0)::bigint as likes,
			COALESCE(pm.comments, 0)::bigint as comments,
			COALESCE(pm.shares, 0)::bigint as shares,
			(COALESCE(pm.likes, 0) + COALESCE(pm.comments, 0) + COALESCE(pm.shares, 0))::bigint as engagement,
			CASE 
				WHEN COALESCE(pm.views, 0) > 0 
				THEN (COALESCE(lc.clicks, 0) / pm.views * 100)
				ELSE 0 
			END as ctr,
			(COALESCE(pm.views, 0) + COALESCE(pm.likes, 0) * 2 + COALESCE(pm.comments, 0) * 3 + COALESCE(pm.shares, 0) * 4) as performance_score
		FROM content_info ci
		LEFT JOIN pivoted_metrics pm ON pm.content_channel_id = ci.content_channel_id
		LEFT JOIN latest_clicks lc ON lc.content_channel_id = ci.content_channel_id
		ORDER BY performance_score %s
		LIMIT ?
	`, channelFilter, order)

	args = append(args, limit)

	var results []dtos.ContentPerformanceDTO
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get content by performance", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetScheduledContentCount returns count of scheduled content for a date range
func (r *contentStaffAnalyticsRepository) GetScheduledContentCount(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var count int64

	if err := r.db.WithContext(ctx).Model(new(model.Schedule)).
		Where("deleted_at IS NULL").
		Where("scheduled_at >= ?", startDate).
		Where("scheduled_at < ?", endDate).
		Where("status IN ?", []enum.ScheduleStatus{enum.ScheduleStatusPending, enum.ScheduleStatusProcessing}).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get scheduled content count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetTaskContentDeliverableCount returns expected content count from tasks in date range
func (r *contentStaffAnalyticsRepository) GetTaskContentDeliverableCount(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var count int64

	// Count tasks with type CONTENT that have deadline in the date range
	if err := r.db.WithContext(ctx).Table("tasks t").
		Where("t.deleted_at IS NULL").
		Where("t.type = ?", enum.TaskTypeContent.String()).
		Where("t.deadline >= ?", startDate).
		Where("t.deadline < ?", endDate).
		Where("t.status NOT IN ?", []string{
			enum.TaskStatusCancelled.String(),
		}).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get task content deliverable count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetChannelMappedMetrics returns DELTA kpi_metrics values for a channel during the period
// Uses pre-aggregated CHANNEL-level metrics from ContentMetricsPollerJob
// Falls back to aggregating from CONTENT_CHANNEL level if no CHANNEL-level data exists
// Delta = (last value in period) - (first value in period)
func (r *contentStaffAnalyticsRepository) GetChannelMappedMetrics(ctx context.Context, channelID uuid.UUID, startDate, endDate time.Time) (map[string]float64, error) {
	metrics := make(map[string]float64)

	type MetricAggregate struct {
		Type  string
		Delta float64
	}
	var results []MetricAggregate

	/*
		// First, try to get pre-aggregated CHANNEL-level metrics (from poller job)
		// Use delta logic: (last value in period) - (first value in period)
		channelQuery := `
			WITH range_metrics AS (
				SELECT
					km.type,
					km.value,
					ROW_NUMBER() OVER (PARTITION BY km.type ORDER BY km.recorded_date ASC) as rn_first,
					ROW_NUMBER() OVER (PARTITION BY km.type ORDER BY km.recorded_date DESC) as rn_last
				FROM kpi_metrics km
				WHERE km.reference_id = ?
				  AND km.reference_type = ?
				  AND km.recorded_date >= ?
				  AND km.recorded_date < ?
			),
			first_last AS (
				SELECT
					type,
					MAX(CASE WHEN rn_first = 1 THEN value END) as first_val,
					MAX(CASE WHEN rn_last = 1 THEN value END) as last_val
				FROM range_metrics
				GROUP BY type
			)
			SELECT type, GREATEST(COALESCE(last_val, 0) - COALESCE(first_val, 0), 0) as total
			FROM first_last
		`
		if err := r.db.WithContext(ctx).Raw(channelQuery,
			channelID,
			enum.KPIReferenceTypeChannel.String(),
			startDate, endDate,
		).Scan(&results).Error; err != nil {
			zap.L().Warn("Failed to get channel-level metrics", zap.Error(err))
		}

		// If we found CHANNEL-level metrics, use them
		if len(results) > 0 {
			for _, res := range results {
				metrics[res.Type] = res.Total
			}
			return metrics, nil
		}
	*/

	// Fallback: Aggregate from CONTENT_CHANNEL level (for old data before poller fix)
	// Use delta logic: For each content_channel, calculate (last - first), then SUM across all
	fallbackQuery := `
			WITH range_metrics AS (
				SELECT
					km.reference_id,
					km.type,
					km.value,
					ch.code, 
					ROW_NUMBER() OVER (PARTITION BY km.reference_id, km.type ORDER BY km.recorded_date ASC) as rn_first,
					ROW_NUMBER() OVER (PARTITION BY km.reference_id, km.type ORDER BY km.recorded_date DESC) as rn_last
				FROM kpi_metrics km
					JOIN content_channels cc ON cc.id = km.reference_id
					JOin channels ch ON ch.id = cc.channel_id
				WHERE cc.channel_id = ?
				  AND km.reference_type = ?
				  AND cc.auto_post_status = ?
				  AND km.recorded_date >= ? AND km.recorded_date < ?
				  AND cc.published_at >= ? AND cc.published_at < ?
			),
			first_last AS (
				SELECT
					reference_id,
					type,
					code,
					MAX(CASE WHEN rn_first = 1 THEN value END) as first_val,
					MAX(CASE WHEN rn_last = 1 THEN value END) as last_val,
					SUM(value) as sum_val
				FROM range_metrics
         		GROUP BY reference_id, type, code
			)
			SELECT
				type,
				SUM(
					CASE
						WHEN (type IN ('VIEWS', 'UNIQUE_VIEWS') AND code = 'WEBSITE') THEN sum_val
						ELSE GREATEST(last_val - first_val, 0)
					END
				) as delta
			FROM first_last
			GROUP BY type
		`

	if err := r.db.WithContext(ctx).Raw(fallbackQuery,
		channelID,
		enum.KPIReferenceTypeContentChannel.String(),
		enum.AutoPostStatusPosted.String(),
		startDate, endDate,
		startDate, endDate,
	).Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get channel mapped metrics from content_channel fallback", zap.Error(err))
		return metrics, err
	}

	for _, res := range results {
		metrics[res.Type] = res.Delta
	}

	return metrics, nil
}

// GetChannelFollowers returns the followers count for a channel at a specific time (or latest before)
func (r *contentStaffAnalyticsRepository) GetChannelFollowers(ctx context.Context, channelID uuid.UUID, atTime time.Time) (int64, error) {
	var followers float64

	// Get the latest recorded followers count before or at atTime
	err := r.db.WithContext(ctx).Table("kpi_metrics").
		Select("value").
		Where("reference_id = ?", channelID).
		Where("reference_type = ?", enum.KPIReferenceTypeChannel).
		Where("type = ?", enum.KPIValueTypeFollowers).
		Where("recorded_date <= ?", atTime).
		Order("recorded_date DESC").
		Limit(1).
		Scan(&followers).Error

	if err != nil {
		return 0, err
	}

	return int64(followers), nil
}

// GetAggregatedClicksFromKPIMetrics returns total clicks from affiliate links for a channel in date range
func (r *contentStaffAnalyticsRepository) GetAggregatedClicksFromKPIMetrics(ctx context.Context, channelID *uuid.UUID, startDate, endDate *time.Time) (int64, error) {
	var totalClicks float64
	query := r.db.WithContext(ctx).Table("kpi_metrics km").
		Joins("JOIN affiliate_links al ON al.id = km.reference_id").
		Where("km.reference_type = ?", enum.KPIReferenceTypeAffiliateLink).
		Where("km.type = ?", enum.KPIValueTypeClickThrough).
		Where("km.recorded_date >= ?", *startDate).
		Where("km.recorded_date < ?", *endDate)

	if channelID != nil {
		query = query.Where("(al.channel_id = ? OR al.metadata ->> 'channel_id' = ?)", *channelID, *channelID)
	}

	if err := query.Select("COALESCE(SUM(km.value), 0)").Scan(&totalClicks).Error; err != nil {
		zap.L().Error("Failed to get aggregated clicks from kpi_metrics", zap.Error(err))
		return 0, err
	}
	return int64(totalClicks), nil
}
