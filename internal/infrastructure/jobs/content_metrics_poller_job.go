package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/gorountine"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ContentMetricsPollerJob fetches metrics for content channels from social platforms
// This job replaces the old SocialMetricsPollerJob with:
// - Channel-level metrics (page followers, fan count, etc.)
// - Paginated post fetching from Facebook
// - TikTok video list metrics
// - Rate limiting for Facebook API
// - Continue-on-failure error handling
// - Bulk KPI metrics insertion for efficiency
// - Pre-fetched content channels to avoid N+1 queries
// - Dynamic worker pool for video insights fetching
type ContentMetricsPollerJob struct {
	db                  *gorm.DB
	unitOfWork          irepository.UnitOfWork
	contentChannelRepo  irepository.GenericRepository[model.ContentChannel]
	channelRepo         irepository.GenericRepository[model.Channel]
	kpiMetricsRepo      irepository.KPIMetricsRepository
	contentCommentRepo  irepository.GenericRepository[model.ContentComment]
	channelService      iservice.ChannelService
	tiktokSocialService iservice.TikTokSocialService
	facebookProxy       iproxies.FacebookProxy
	tiktokProxy         iproxies.TikTokProxy
	cronScheduler       *cron.Cron
	cronExpr            string
	enabled             bool
	entryID             cron.EntryID
	lastRunTime         time.Time

	// Rate limiting for Facebook API (10 req/s, burst 20)
	fbRateLimiter *rate.Limiter
}

// metricsCollector aggregates KPI metrics from all goroutines for bulk insertion
type metricsCollector struct {
	mu      sync.Mutex
	metrics []*model.KPIMetrics

	// Channel updates to batch
	channelUpdates        map[uuid.UUID]*channelMetricsUpdate
	contentChannelUpdates map[uuid.UUID]*contentChannelMetricsUpdate

	// Aggregated metrics from posts per channel (reactions, comments, shares)
	// This is used to compute page-level aggregated engagement metrics
	channelAggregatedMetrics map[uuid.UUID]*channelAggregatedMetrics
}

// channelAggregatedMetrics holds aggregated metrics from all posts for a channel
type channelAggregatedMetrics struct {
	totalReactions int64
	totalComments  int64
	totalShares    int64
	totalViews     int64
	postsCount     int
}

type channelMetricsUpdate struct {
	channel       *model.Channel
	rawMetrics    map[string]any
	mappedMetrics map[string]float64
}

type contentChannelMetricsUpdate struct {
	contentChannel *model.ContentChannel
	rawMetrics     map[string]any
	mappedMetrics  map[enum.KPIValueType]float64
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		metrics:                  make([]*model.KPIMetrics, 0, 1000),
		channelUpdates:           make(map[uuid.UUID]*channelMetricsUpdate),
		contentChannelUpdates:    make(map[uuid.UUID]*contentChannelMetricsUpdate),
		channelAggregatedMetrics: make(map[uuid.UUID]*channelAggregatedMetrics),
	}
}

// addChannelUpdate adds a channel metrics update to the collector (thread-safe)
func (c *metricsCollector) addChannelUpdate(channel *model.Channel, rawMetrics map[string]any, mappedMetrics map[string]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channelUpdates[channel.ID] = &channelMetricsUpdate{
		channel:       channel,
		rawMetrics:    rawMetrics,
		mappedMetrics: mappedMetrics,
	}

	// Also add KPI metrics
	now := time.Now()
	for metricName, metricValue := range mappedMetrics {
		kpiType := enum.KPIValueType(metricName)
		if !kpiType.IsValid() {
			continue
		}
		c.metrics = append(c.metrics, &model.KPIMetrics{
			ReferenceID:   channel.ID,
			ReferenceType: enum.KPIReferenceTypeChannel,
			Type:          kpiType,
			Value:         metricValue,
			RecordedDate:  now,
		})
	}
}

// addContentChannelUpdate adds a content channel metrics update (thread-safe)
func (c *metricsCollector) addContentChannelUpdate(cc *model.ContentChannel, rawMetrics map[string]any, mappedMetrics map[enum.KPIValueType]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.contentChannelUpdates[cc.ID] = &contentChannelMetricsUpdate{
		contentChannel: cc,
		rawMetrics:     rawMetrics,
		mappedMetrics:  mappedMetrics,
	}

	// Also add KPI metrics
	now := time.Now()
	for metricName, metricValue := range mappedMetrics {
		kpiType := enum.KPIValueType(metricName)
		if !kpiType.IsValid() {
			continue
		}
		c.metrics = append(c.metrics, &model.KPIMetrics{
			ReferenceID:   cc.ID,
			ReferenceType: enum.KPIReferenceTypeContentChannel,
			Type:          kpiType,
			Value:         metricValue,
			RecordedDate:  now,
		})
	}
}

// getMetrics returns all collected KPI metrics
func (c *metricsCollector) getMetrics() []*model.KPIMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}

// getChannelUpdates returns all channel updates
func (c *metricsCollector) getChannelUpdates() map[uuid.UUID]*channelMetricsUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelUpdates
}

// getContentChannelUpdates returns all content channel updates
func (c *metricsCollector) getContentChannelUpdates() map[uuid.UUID]*contentChannelMetricsUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.contentChannelUpdates
}

// aggregatePostMetricsToChannel adds post metrics to the channel's aggregated totals (thread-safe)
func (c *metricsCollector) aggregatePostMetricsToChannel(channelID uuid.UUID, reactions, comments, shares, views int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.channelAggregatedMetrics[channelID]; !exists {
		c.channelAggregatedMetrics[channelID] = &channelAggregatedMetrics{}
	}

	agg := c.channelAggregatedMetrics[channelID]
	agg.totalReactions += int64(reactions)
	agg.totalComments += int64(comments)
	agg.totalShares += int64(shares)
	agg.totalViews += int64(views)
	agg.postsCount++
}

// getChannelAggregatedMetrics returns aggregated metrics for all channels
func (c *metricsCollector) getChannelAggregatedMetrics() map[uuid.UUID]*channelAggregatedMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelAggregatedMetrics
}

// NewContentMetricsPollerJob creates a new instance of ContentMetricsPollerJob
func NewContentMetricsPollerJob(
	db *gorm.DB,
	unitOfWork irepository.UnitOfWork,
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	channelRepo irepository.GenericRepository[model.Channel],
	kpiMetricsRepo irepository.KPIMetricsRepository,
	contentCommentRepo irepository.GenericRepository[model.ContentComment],
	channelService iservice.ChannelService,
	tiktokSocialService iservice.TikTokSocialService,
	facebookProxy iproxies.FacebookProxy,
	tiktokProxy iproxies.TikTokProxy,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	cronExpr := adminConfig.ContentMetricsPollerCronExpr
	if cronExpr == "" {
		cronExpr = "0 */30 * * * *" // Default to every 30 minutes
	}

	return &ContentMetricsPollerJob{
		db:                  db,
		unitOfWork:          unitOfWork,
		contentChannelRepo:  contentChannelRepo,
		channelRepo:         channelRepo,
		kpiMetricsRepo:      kpiMetricsRepo,
		contentCommentRepo:  contentCommentRepo,
		channelService:      channelService,
		tiktokSocialService: tiktokSocialService,
		facebookProxy:       facebookProxy,
		tiktokProxy:         tiktokProxy,
		cronScheduler:       cronScheduler,
		cronExpr:            cronExpr,
		enabled:             adminConfig.ContentMetricsPollerEnabled,
		lastRunTime:         time.Now(),
		// Allow 10 requests per second with burst of 20
		fbRateLimiter: rate.NewLimiter(rate.Limit(10), 20),
	}
}

// Initialize implements CronJob
func (j *ContentMetricsPollerJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("ContentMetricsPollerJob is disabled via admin config")
		return nil
	}

	zap.L().Debug("Initializing ContentMetricsPollerJob...")

	entryID, err := j.cronScheduler.AddFunc(j.cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})
	if err != nil {
		zap.L().Error("Failed to schedule ContentMetricsPollerJob", zap.Error(err))
		return fmt.Errorf("failed to schedule content metrics poller job: %w", err)
	}

	j.entryID = entryID
	zap.L().Info("ContentMetricsPollerJob scheduled", zap.String("cron_expr", j.cronExpr))
	return nil
}

// Run implements CronJob - main entry point for job execution
func (j *ContentMetricsPollerJob) Run() {
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Panic recovered in ContentMetricsPollerJob, disabling job",
				zap.Duration("elapsed_time", time.Since(startTime)),
				zap.Any("recover", r))
			j.SetEnabled(false)
		}
	}()

	ctx := context.Background()
	j.lastRunTime = time.Now()

	zap.L().Info("ContentMetricsPollerJob starting...", zap.Time("start_time", startTime))

	// Get active channels by platform
	// fbChannels, tikTokChannels, err := j.getActiveChannels(ctx)
	channels, err := j.getActiveChannels(ctx)
	if err != nil {
		zap.L().Error("Failed to get active channels", zap.Error(err))
		return
	}
	fbChannels := utils.FilterSlice(channels, func(c model.Channel) bool {
		return c.Code == constant.ChannelCodeFacebook
	})
	tikTokChannels := utils.FilterSlice(channels, func(c model.Channel) bool {
		return c.Code == constant.ChannelCodeTikTok
	})
	websiteChannels := utils.FilterSlice(channels, func(c model.Channel) bool {
		return c.Code == constant.ChannelCodeWebsite
	})

	zap.L().Info("Found active channels",
		zap.Int("channels", len(channels)))

	// Pre-fetch content channels for all channels to avoid N+1 queries
	contentChannelMap := j.prefetchContentChannels(ctx, channels)

	// Create centralized metrics collector
	collector := newMetricsCollector()

	// Create worker pool for video insights fetching (5 concurrent workers)
	videoInsightsPool := gorountine.NewWorkerPool(ctx, 5)
	videoInsightsPool.Start()

	// Execute all tasks with continue-on-failure
	var wg sync.WaitGroup
	tasks := []struct {
		name string
		fn   func(context.Context, *metricsCollector, map[uuid.UUID]map[string]*model.ContentChannel, *gorountine.WorkerPool)
	}{
		{"Facebook Page Metrics", func(ctx context.Context, c *metricsCollector, ccMap map[uuid.UUID]map[string]*model.ContentChannel, pool *gorountine.WorkerPool) {
			j.fetchFacebookPageMetrics(ctx, fbChannels, c)
		}},
		{"Facebook Posts Metrics", func(ctx context.Context, c *metricsCollector, ccMap map[uuid.UUID]map[string]*model.ContentChannel, pool *gorountine.WorkerPool) {
			j.fetchFacebookPostsMetrics(ctx, fbChannels, c, ccMap, pool)
		}},
		{"TikTok User Metrics", func(ctx context.Context, c *metricsCollector, ccMap map[uuid.UUID]map[string]*model.ContentChannel, pool *gorountine.WorkerPool) {
			j.fetchTikTokUserMetrics(ctx, tikTokChannels, c)
		}},
		{"TikTok Video List Metrics", func(ctx context.Context, c *metricsCollector, ccMap map[uuid.UUID]map[string]*model.ContentChannel, pool *gorountine.WorkerPool) {
			j.fetchTikTokVideoListMetrics(ctx, tikTokChannels, c, ccMap)
		}},
		{"Website Metrics", func(ctx context.Context, c *metricsCollector, ccMap map[uuid.UUID]map[string]*model.ContentChannel, pool *gorountine.WorkerPool) {
			j.processWebsiteMetrics(ctx, websiteChannels, c, ccMap)
		}},
	}

	for _, task := range tasks {
		wg.Add(1)
		go func(taskName string, taskFn func(context.Context, *metricsCollector, map[uuid.UUID]map[string]*model.ContentChannel, *gorountine.WorkerPool)) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					zap.L().Error("Task panicked, continuing with others",
						zap.String("task", taskName),
						zap.Any("recover", r))
				}
			}()

			zap.L().Debug("Starting task", zap.String("task", taskName))
			taskFn(ctx, collector, contentChannelMap, videoInsightsPool)
			zap.L().Debug("Completed task", zap.String("task", taskName))
		}(task.name, task.fn)
	}

	wg.Wait()

	// Close worker pool and wait for all video insights tasks to complete
	videoInsightsPool.Close()
	videoInsightsPool.Wait()

	if videoInsightsPool.HasErrors() {
		zap.L().Warn("Some video insights tasks failed",
			zap.Int("error_count", videoInsightsPool.ErrorCount()))
	}

	// Bulk persist all collected data
	j.persistCollectedData(ctx, collector)

	zap.L().Info("ContentMetricsPollerJob completed", zap.Duration("elapsed_time", time.Since(startTime)))
}

// prefetchContentChannels loads all content channels for given channels in one query
// Returns map[channelID][externalPostID]*ContentChannel for O(1) lookup
func (j *ContentMetricsPollerJob) prefetchContentChannels(
	ctx context.Context, channels []model.Channel,
) map[uuid.UUID]map[string]*model.ContentChannel {
	result := make(map[uuid.UUID]map[string]*model.ContentChannel)

	// Collect all channel IDs
	channelIDs := make([]uuid.UUID, 0, len(channels))
	for _, ch := range channels {
		channelIDs = append(channelIDs, ch.ID)
	}
	if len(channelIDs) == 0 {
		return result
	}

	// Fetch all content channels for these channels in one query
	contentChannels, _, err := j.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("channel_id IN ?", channelIDs).
				Where("external_post_id IS NOT NULL").
				Where("external_post_id != ''")
		},
		nil, 0, 0) // No pagination, get all

	if err != nil {
		zap.L().Error("Failed to prefetch content channels", zap.Error(err))
		return result
	}

	// Build lookup map
	for i := range contentChannels {
		cc := &contentChannels[i]
		if _, exists := result[cc.ChannelID]; !exists {
			result[cc.ChannelID] = make(map[string]*model.ContentChannel)
		}
		if cc.ExternalPostID != nil {
			result[cc.ChannelID][*cc.ExternalPostID] = cc
		}
	}

	zap.L().Debug("Prefetched content channels",
		zap.Int("total_channels", len(channelIDs)),
		zap.Int("content_channels", len(contentChannels)))

	return result
}

// persistCollectedData bulk inserts KPI metrics and updates channels
func (j *ContentMetricsPollerJob) persistCollectedData(ctx context.Context, collector *metricsCollector) {
	startTime := time.Now()

	// 1. Merge aggregated post metrics into channel updates
	j.mergeAggregatedMetricsIntoChannels(collector)

	// 2. Bulk insert KPI metrics
	metrics := collector.getMetrics()
	if len(metrics) > 0 {
		rowsAffected, err := j.kpiMetricsRepo.BulkAdd(ctx, metrics, 100)
		if err != nil {
			zap.L().Error("Failed to bulk insert KPI metrics", zap.Error(err))
		} else {
			zap.L().Info("Bulk inserted KPI metrics",
				zap.Int64("rows_affected", rowsAffected),
				zap.Int("total_metrics", len(metrics)))
		}
	}

	// 3. Update channels with metrics JSONB
	channelUpdates := collector.getChannelUpdates()
	for _, update := range channelUpdates {
		metricsData := map[string]any{
			"current_fetched": update.rawMetrics,
			"current_mapped":  update.mappedMetrics,
			"last_updated_at": time.Now().Format(time.RFC3339),
		}
		metricsJSON, err := json.Marshal(metricsData)
		if err != nil {
			continue
		}
		update.channel.Metrics = datatypes.JSON(metricsJSON)
		if err := j.channelRepo.Update(ctx, update.channel); err != nil {
			zap.L().Warn("Failed to update channel metrics",
				zap.String("channel_id", update.channel.ID.String()),
				zap.Error(err))
		}
	}

	// 4. Update content channels with metrics JSONB
	contentChannelUpdates := collector.getContentChannelUpdates()
	for _, update := range contentChannelUpdates {
		existingMetrics, _ := update.contentChannel.GetMetrics()
		if existingMetrics == nil {
			existingMetrics = &model.ContentChannelMetrics{}
		}

		// Move current to last
		existingMetrics.LastFetched = existingMetrics.CurrentFetched
		existingMetrics.LastMapped = existingMetrics.CurrentMapped
		existingMetrics.CurrentFetched = update.rawMetrics
		existingMetrics.CurrentMapped = update.mappedMetrics

		update.contentChannel.Metrics = existingMetrics
		if err := j.contentChannelRepo.Update(ctx, update.contentChannel); err != nil {
			zap.L().Warn("Failed to update content channel metrics",
				zap.String("content_channel_id", update.contentChannel.ID.String()),
				zap.Error(err))
		}
	}

	zap.L().Info("Persisted collected data",
		zap.Int("kpi_metrics", len(metrics)),
		zap.Int("channel_updates", len(channelUpdates)),
		zap.Int("content_channel_updates", len(contentChannelUpdates)),
		zap.Duration("duration", time.Since(startTime)))
}

// IsEnabled implements CronJob
func (j *ContentMetricsPollerJob) IsEnabled() bool {
	return j.enabled
}

// SetEnabled implements CronJob
func (j *ContentMetricsPollerJob) SetEnabled(enabled bool) {
	j.enabled = enabled
}

// GetLastRunTime implements CronJob
func (j *ContentMetricsPollerJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// Restart implements CronJob
func (j *ContentMetricsPollerJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting ContentMetricsPollerJob due to config change")

	j.enabled = adminConfig.ContentMetricsPollerEnabled
	j.cronExpr = adminConfig.ContentMetricsPollerCronExpr

	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		j.entryID = 0
	}

	return j.Initialize()
}

// region: ======== Helper Methods ========

// getActiveChannels returns active Facebook and TikTok channels
func (j *ContentMetricsPollerJob) getActiveChannels(ctx context.Context) ([]model.Channel, error) {
	channels, _, err := j.channelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).
				Where("(hashed_access_token IS NOT NULL) OR (code = ?)", "WEBSITE")
		},
		nil, 0, 0)
	if err != nil {
		return nil, err
	}

	return channels, nil
}

// waitForFBRateLimit waits for Facebook rate limiter before making API call
func (j *ContentMetricsPollerJob) waitForFBRateLimit(ctx context.Context) error {
	return j.fbRateLimiter.Wait(ctx)
}

// mergeAggregatedMetricsIntoChannels merges post-level aggregated metrics into channel updates
// This is called after all posts are processed to compute page-level totals
func (j *ContentMetricsPollerJob) mergeAggregatedMetricsIntoChannels(collector *metricsCollector) {
	aggregatedMetrics := collector.getChannelAggregatedMetrics()
	channelUpdates := collector.getChannelUpdates()

	for channelID, agg := range aggregatedMetrics {
		update, exists := channelUpdates[channelID]
		if !exists {
			// No channel update exists (shouldn't happen if page metrics were fetched)
			continue
		}

		// Add aggregated raw metrics
		update.rawMetrics["total_reactions"] = agg.totalReactions
		update.rawMetrics["total_comments"] = agg.totalComments
		update.rawMetrics["total_shares"] = agg.totalShares
		update.rawMetrics["total_views"] = agg.totalViews
		update.rawMetrics["posts_count"] = agg.postsCount

		// Add aggregated mapped metrics (for KPI)
		totalEngagement := float64(agg.totalReactions + agg.totalComments + agg.totalShares)
		if agg.totalReactions > 0 {
			update.mappedMetrics[enum.KPIValueTypeLikes.String()] = float64(agg.totalReactions)
		}
		if agg.totalComments > 0 {
			update.mappedMetrics[enum.KPIValueTypeComments.String()] = float64(agg.totalComments)
		}
		if agg.totalShares > 0 {
			update.mappedMetrics[enum.KPIValueTypeShares.String()] = float64(agg.totalShares)
		}
		if agg.totalViews > 0 {
			update.mappedMetrics[enum.KPIValueTypeReach.String()] = float64(agg.totalViews)
			update.mappedMetrics[enum.KPIValueTypeViews.String()] = float64(agg.totalViews)
			update.mappedMetrics[enum.KPIValueTypeUniqueViews.String()] = float64(agg.totalViews)
		}
		if totalEngagement > 0 {
			update.mappedMetrics[enum.KPIValueTypeEngagement.String()] = totalEngagement
		}

		zap.L().Debug("Merged aggregated metrics into channel",
			zap.String("channel_id", channelID.String()),
			zap.Int64("total_reactions", agg.totalReactions),
			zap.Int64("total_comments", agg.totalComments),
			zap.Int64("total_shares", agg.totalShares),
			zap.Int64("total_views", agg.totalViews),
			zap.Int("posts_count", agg.postsCount))
	}
}

// endregion

// region: ======== Facebook Metrics Methods ========

// fetchFacebookPageMetrics fetches page-level metrics for Facebook channels
func (j *ContentMetricsPollerJob) fetchFacebookPageMetrics(ctx context.Context, channels []model.Channel, collector *metricsCollector) {
	for i := range channels {
		channel := &channels[i]

		accessToken, err := j.channelService.GetDecryptedToken(ctx, channel.Name)
		if err != nil {
			zap.L().Error("Failed to decrypt Facebook access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		if channel.ExternalID == nil {
			zap.L().Warn("Facebook channel has no external_id (page_id)",
				zap.String("channel_name", channel.Name))
			continue
		}

		// Wait for rate limit
		if err = j.waitForFBRateLimit(ctx); err != nil {
			zap.L().Error("Rate limit wait failed", zap.Error(err))
			continue
		}

		// Fetch page info with metrics
		fields := []string{"id", "name", "fan_count", "followers_count"}
		pageInfo, err := j.facebookProxy.GetPageInfo(ctx, *channel.ExternalID, accessToken, fields)
		if err != nil {
			zap.L().Error("Failed to fetch Facebook page info",
				zap.String("channel_name", channel.Name),
				zap.String("page_id", *channel.ExternalID),
				zap.Error(err))
			continue
		}

		// Map metrics
		rawMetrics := map[string]any{
			"fan_count":       pageInfo.FanCount,
			"followers_count": pageInfo.FollowersCount,
		}

		mappedMetrics := map[string]float64{
			enum.KPIValueTypeFollowers.String(): float64(pageInfo.FollowersCount),
		}

		// Add to collector (not persisting directly)
		collector.addChannelUpdate(channel, rawMetrics, mappedMetrics)

		zap.L().Debug("Collected Facebook page metrics",
			zap.String("channel_name", channel.Name),
			zap.Int("followers_count", pageInfo.FollowersCount))
	}
}

// fetchFacebookPostsMetrics fetches post metrics from Facebook pages
func (j *ContentMetricsPollerJob) fetchFacebookPostsMetrics(
	ctx context.Context,
	channels []model.Channel,
	collector *metricsCollector,
	contentChannelMap map[uuid.UUID]map[string]*model.ContentChannel,
	videoInsightsPool *gorountine.WorkerPool,
) {
	for i := range channels {
		channel := &channels[i]

		accessToken, err := j.channelService.GetDecryptedToken(ctx, channel.Name)
		if err != nil {
			zap.L().Error("Failed to decrypt Facebook access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		if channel.ExternalID == nil {
			continue
		}

		// Get pre-fetched content channels for this channel
		ccMap := contentChannelMap[channel.ID]
		if ccMap == nil {
			ccMap = make(map[string]*model.ContentChannel)
		}

		// Fetch posts with metrics using pagination
		// fields := "id,created_time,message,reactions.limit(0).summary(total_count),comments.limit(0).summary(total_count),shares,attachments{media_type,target}"
		fields := "id,created_time,reactions.limit(0).summary(total_count),comments.limit(0).summary(total_count),shares,attachments{media_type,target},insights.metric(post_impressions_unique).period(lifetime)"
		var cursor *string

		postsProcessed := 0
		maxPosts := 100 // Limit to avoid excessive API calls

		for postsProcessed < maxPosts {
			// Wait for rate limit
			if err := j.waitForFBRateLimit(ctx); err != nil {
				zap.L().Error("Rate limit wait failed", zap.Error(err))
				break
			}

			postsResp, err := j.facebookProxy.GetPagePosts(ctx, *channel.ExternalID, accessToken, fields, cursor)
			if err != nil {
				zap.L().Error("Failed to fetch Facebook page posts",
					zap.String("channel_name", channel.Name),
					zap.Error(err))
				break
			}

			if len(postsResp.Data) == 0 {
				break
			}

			// Process each post
			for _, post := range postsResp.Data {
				j.processPostMetrics(ctx, channel, accessToken, &post, collector, ccMap, videoInsightsPool)
				postsProcessed++
			}

			// Check for next page
			if postsResp.Paging == nil || postsResp.Paging.Next == nil {
				break
			}
			cursor = &postsResp.Paging.Cursors.After
		}

		zap.L().Debug("Processed Facebook posts",
			zap.String("channel_name", channel.Name),
			zap.Int("posts_processed", postsProcessed))
	}
}

// processPostMetrics processes metrics for a single Facebook post
func (j *ContentMetricsPollerJob) processPostMetrics(
	ctx context.Context,
	channel *model.Channel,
	accessToken string,
	post *dtos.FacebookPagePost,
	collector *metricsCollector,
	ccMap map[string]*model.ContentChannel,
	videoInsightsPool *gorountine.WorkerPool,
) {
	// O(1) lookup from pre-fetched map instead of database query
	cc, exists := ccMap[post.ID]
	var firstAttachment *dtos.FacebookAttachment
	if post.Attachments != nil && len(post.Attachments.Data) > 0 {
		firstAttachment = &post.Attachments.Data[0]
	}
	if !exists {
		if firstAttachment == nil || (firstAttachment.MediaType != "photo" && firstAttachment.MediaType != "video") {
			// Post not tracked in our system, skip
			return
		}
		oldExternalPostID := firstAttachment.Target.ID
		temp, ok := ccMap[oldExternalPostID]
		if !ok {
			oldExternalPostID = fmt.Sprintf("%s_%s", utils.DerefPtr(channel.ExternalID, "null"), firstAttachment.Target.ID)
			temp, ok = ccMap[oldExternalPostID]
		}

		if ok {
			var baseURL string
			switch channel.Code {
			case constant.ChannelCodeFacebook:
				baseURL = "https://www.facebook.com/"
			case constant.ChannelCodeTikTok:
				baseURL = "https://www.tiktok.com/@"
			}

			if err := j.updateExternalPostIDForContentChannel(ctx, temp, baseURL, post.ID, firstAttachment.MediaType); err != nil {
				zap.L().Warn("Failed to update external_post_id for content channel",
					zap.String("content_channel_id", temp.ID.String()),
					zap.String("old_external_post_id", oldExternalPostID),
					zap.String("new_external_post_id", post.ID),
					zap.Error(err))
			}

			cc = temp
			exists = true
		} else {
			// Post not tracked in our system even after best effort to find it, skip
			return
		}
	}

	// Extract metrics from post response
	rawMetrics := map[string]any{
		"post_id": post.ID,
	}

	reactions := 0
	comments := 0
	shares := 0
	views := 0

	if post.Reactions != nil {
		reactions = post.Reactions.Summary.TotalCount
		rawMetrics["reactions"] = reactions
	}
	if post.Comments != nil {
		comments = post.Comments.Summary.TotalCount
		rawMetrics["comments"] = comments
	}
	if post.Shares != nil {
		shares = post.Shares.Count
		rawMetrics["shares"] = shares
	}
	if post.Insights != nil && len(post.Insights.Data) > 0 {
		firstData := post.Insights.Data[0]
		if len(firstData.Values) > 0 {
			if value, ok := firstData.Values[0].Value.(int); ok {
				views = value
				rawMetrics["views"] = value
			}
		}
	}

	// Map to KPI metrics using helper
	mappedMetrics := map[enum.KPIValueType]float64{
		enum.KPIValueTypeViews:       0,
		enum.KPIValueTypeUniqueViews: 0,
		enum.KPIValueTypeComments:    0,
		enum.KPIValueTypeLikes:       0,
		enum.KPIValueTypeShares:      0,
		enum.KPIValueTypeEngagement:  0,
	}

	// Reactions -> Likes + Engagement
	if reactions > 0 {
		maps.Copy(mappedMetrics, helper.MapFacebookMetricsToKPIField("post_reactions_by_type_total", float64(reactions)))
	}

	// Comments -> Comments + Engagement
	if comments > 0 {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeComments:   float64(comments),
			enum.KPIValueTypeEngagement: float64(comments),
		})
	}

	// Shares -> Shares + Engagement
	if shares > 0 {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeShares:     float64(shares),
			enum.KPIValueTypeEngagement: float64(shares),
		})
	}

	if views > 0 { // Views -> Views + UniqueViews + Reach
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{
			enum.KPIValueTypeViews:       float64(views),
			enum.KPIValueTypeUniqueViews: float64(views),
			enum.KPIValueTypeReach:       float64(views),
		})
	}

	// If video post, submit video insights fetch to worker pool
	if post.Attachments != nil && len(post.Attachments.Data) > 0 {
		for _, attachment := range post.Attachments.Data {
			if attachment.MediaType == "video" && attachment.Target != nil {
				// Capture variables for closure
				videoID := attachment.Target.ID
				token := accessToken
				rawM := rawMetrics
				mappedM := mappedMetrics
				contentChannel := cc
				col := collector
				chID := channel.ID
				r, c, s := reactions, comments, shares

				// Submit to worker pool for async processing
				videoInsightsPool.Submit(func(ctx context.Context) error {
					views = j.fetchAndMergeVideoInsightsAsync(ctx, token, videoID, rawM, mappedM)
					// After video insights are fetched, add to collector
					col.addContentChannelUpdate(contentChannel, rawM, mappedM)
					// Aggregate post metrics with video views to channel level
					col.aggregatePostMetricsToChannel(chID, r, c, s, views)
					return nil
				})
				return // Return early, collector will be updated by worker
			}
		}
	}

	// For non-video posts, add to collector directly
	collector.addContentChannelUpdate(cc, rawMetrics, mappedMetrics)

	// Aggregate post metrics to channel level (for page-level totals)
	// Views will be 0 for non-video posts
	collector.aggregatePostMetricsToChannel(channel.ID, reactions, comments, shares, views)
}

// fetchAndMergeVideoInsightsAsync fetches video insights (for use in worker pool)
// Returns the total video views for aggregation
func (j *ContentMetricsPollerJob) fetchAndMergeVideoInsightsAsync(
	ctx context.Context,
	accessToken string,
	videoID string,
	rawMetrics map[string]any,
	mappedMetrics map[enum.KPIValueType]float64,
) int {
	// Wait for rate limit
	if err := j.waitForFBRateLimit(ctx); err != nil {
		return 0
	}

	metrics := []string{"total_video_views", "total_video_impressions"}
	insights, err := j.facebookProxy.GetVideoInsights(ctx, videoID, accessToken, metrics, dtos.FacebookInsightsPeriodLifetime)
	if err != nil {
		zap.L().Warn("Failed to fetch video insights",
			zap.String("video_id", videoID),
			zap.Error(err))
		return 0
	}

	var totalViews int
	for _, data := range insights.Data {
		if len(data.Values) > 0 {
			if value, ok := data.Values[0].Value.(float64); ok {
				rawMetrics[data.Name] = value

				// Track views for aggregation
				if data.Name == "total_video_views" {
					totalViews = int(value)
				}

				// Map video metrics
				for k, v := range helper.MapFacebookMetricsToKPIField(data.Name, value) {
					utils.AddValueToMap(mappedMetrics, k, v)
				}
			}
		}
	}

	return totalViews
}

// endregion

// region: ======== TikTok Metrics Methods ========

// fetchTikTokUserMetrics fetches user-level metrics for TikTok channels
func (j *ContentMetricsPollerJob) fetchTikTokUserMetrics(ctx context.Context, channels []model.Channel, collector *metricsCollector) {
	for i := range channels {
		channel := &channels[i]

		accessToken, err := j.tiktokSocialService.GetTikTokAccessToken(ctx)
		if err != nil {
			zap.L().Error("Failed to get TikTok access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		// Fetch user profile with stats
		userProfile, err := j.tiktokProxy.GetSystemUserProfile(ctx, accessToken)
		if err != nil {
			zap.L().Error("Failed to fetch TikTok user profile",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		// Extract metrics
		rawMetrics := map[string]any{
			"follower_count":  utils.DerefPtr(userProfile.Data.User.FollowerCount, 0),
			"following_count": utils.DerefPtr(userProfile.Data.User.FollowingCount, 0),
			"likes_count":     utils.DerefPtr(userProfile.Data.User.LikesCount, 0),
			"video_count":     utils.DerefPtr(userProfile.Data.User.VideoCount, 0),
		}

		mappedMetrics := map[string]float64{
			enum.KPIValueTypeFollowers.String(): float64(utils.DerefPtr(userProfile.Data.User.FollowerCount, 0)),
			enum.KPIValueTypeLikes.String():     float64(utils.DerefPtr(userProfile.Data.User.LikesCount, 0)),
		}

		// Add to collector (not persisting directly)
		collector.addChannelUpdate(channel, rawMetrics, mappedMetrics)

		zap.L().Debug("Collected TikTok user metrics",
			zap.String("channel_name", channel.Name),
			zap.Int64("followers", utils.DerefPtr(userProfile.Data.User.FollowerCount, 0)))
	}
}

// fetchTikTokVideoListMetrics fetches video list and metrics for TikTok channels
func (j *ContentMetricsPollerJob) fetchTikTokVideoListMetrics(
	ctx context.Context,
	channels []model.Channel,
	collector *metricsCollector,
	contentChannelMap map[uuid.UUID]map[string]*model.ContentChannel,
) {
	for i := range channels {
		channel := &channels[i]

		accessToken, err := j.tiktokSocialService.GetTikTokAccessToken(ctx)
		if err != nil {
			zap.L().Error("Failed to get TikTok access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		// Get pre-fetched content channels for this channel
		ccMap := contentChannelMap[channel.ID]
		if ccMap == nil {
			ccMap = make(map[string]*model.ContentChannel)
		}

		// Fetch video list with pagination
		var cursor *int64
		videosProcessed := 0
		maxVideos := 100 // Limit to avoid excessive API calls

		for videosProcessed < maxVideos {
			videosResp, err := j.tiktokProxy.GetUserVideoList(ctx, accessToken, 20, cursor)
			if err != nil {
				zap.L().Error("Failed to fetch TikTok video list",
					zap.String("channel_name", channel.Name),
					zap.Error(err))
				break
			}

			if len(videosResp.Data.Videos) == 0 {
				break
			}

			// Process each video
			for _, video := range videosResp.Data.Videos {
				j.processTikTokVideoMetrics(ctx, channel, &video, collector, ccMap)
				videosProcessed++
			}

			// Check for more videos
			if !videosResp.Data.HasMore {
				break
			}
			cursor = &videosResp.Data.Cursor
		}

		zap.L().Debug("Processed TikTok videos",
			zap.String("channel_name", channel.Name),
			zap.Int("videos_processed", videosProcessed))
	}
}

// processTikTokVideoMetrics processes metrics for a single TikTok video
func (j *ContentMetricsPollerJob) processTikTokVideoMetrics(
	_ context.Context,
	channel *model.Channel,
	video *dtos.TikTokVideoItem,
	collector *metricsCollector,
	ccMap map[string]*model.ContentChannel,
) {
	// O(1) lookup from pre-fetched map instead of database query
	cc, exists := ccMap[video.ID]
	if !exists {
		// Video not tracked in our system, skip
		return
	}

	// Extract metrics
	rawMetrics := map[string]any{
		"video_id":      video.ID,
		"view_count":    video.ViewCount,
		"like_count":    video.LikeCount,
		"comment_count": video.CommentCount,
		"share_count":   video.ShareCount,
	}

	// Map to KPI metrics using helper
	mappedMetrics := make(map[enum.KPIValueType]float64)

	maps.Copy(mappedMetrics, helper.MapTikTokMetricsToKPIField("view_count", float64(video.ViewCount)))
	for k, v := range helper.MapTikTokMetricsToKPIField("like_count", float64(video.LikeCount)) {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{k: v})
	}
	for k, v := range helper.MapTikTokMetricsToKPIField("comment_count", float64(video.CommentCount)) {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{k: v})
	}
	for k, v := range helper.MapTikTokMetricsToKPIField("share_count", float64(video.ShareCount)) {
		utils.AddValuesToMap(mappedMetrics, map[enum.KPIValueType]float64{k: v})
	}

	// Add to collector (not persisting directly)
	collector.addContentChannelUpdate(cc, rawMetrics, mappedMetrics)

	// Aggregate post metrics to channel level
	collector.aggregatePostMetricsToChannel(channel.ID, int(video.LikeCount), int(video.CommentCount), int(video.ShareCount), int(video.ViewCount))
}

// endregion

// region: ======== Website Metrics Methods ========

// processWebsiteMetrics aggregates metrics for website channels from local database
// Website metrics come from:
// - kpi_metrics table (VIEWS, UNIQUE_VIEWS) populated by ContentViewConsumer
// - content_comments table for comment counts
// - ContentChannel.Metrics for stored reaction data
func (j *ContentMetricsPollerJob) processWebsiteMetrics(
	ctx context.Context,
	websiteChannels []model.Channel,
	collector *metricsCollector,
	contentChannelMap map[uuid.UUID]map[string]*model.ContentChannel,
) {
	if len(websiteChannels) == 0 {
		zap.L().Debug("No website channels to process")
		return
	}

	for _, channel := range websiteChannels {
		j.processWebsiteChannel(ctx, channel, collector, contentChannelMap)
	}
}

// processWebsiteChannel aggregates metrics for a single website channel
func (j *ContentMetricsPollerJob) processWebsiteChannel(
	ctx context.Context,
	channel model.Channel,
	collector *metricsCollector,
	contentChannelMap map[uuid.UUID]map[string]*model.ContentChannel,
) {
	zap.L().Debug("Processing website channel metrics", zap.String("channel_id", channel.ID.String()))

	// Get content channels for this website channel
	ccMap, exists := contentChannelMap[channel.ID]
	if !exists || len(ccMap) == 0 {
		// Fetch directly if not in prefetch map (website uses ContentID as key, not ExternalPostID)
		contentChannels, _, err := j.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("channel_id = ?", channel.ID)
		}, nil, 0, 0)
		if err != nil {
			zap.L().Error("Failed to get content channels for website",
				zap.String("channel_id", channel.ID.String()),
				zap.Error(err))
			return
		}
		ccMap = make(map[string]*model.ContentChannel)
		for i := range contentChannels {
			ccMap[contentChannels[i].ContentID.String()] = &contentChannels[i]
		}
	}

	if len(ccMap) == 0 {
		zap.L().Debug("No content channels for website", zap.String("channel_id", channel.ID.String()))
		return
	}

	// Aggregate channel-level metrics
	var totalViews, totalUniqueViews, totalComments, totalLikes, totalShares int64

	for _, cc := range ccMap {
		views, uniqueViews, comments, likes, shares := j.getWebsiteContentMetrics(ctx, cc)
		totalViews += views
		totalUniqueViews += uniqueViews
		totalComments += comments
		totalLikes += likes
		totalShares += shares

		// Update content channel metrics
		// For WEBSITE, Views and UniqueViews are inserted into kpi_metrics by ContentViewConsumer incrementally.
		// We should NOT insert them again here to avoid double counting or incorrect snapshots.
		// However, Likes, Comments, Shares are NOT inserted into kpi_metrics by ContentEngagementService,
		// so we MUST insert them here.
		mappedMetrics := map[enum.KPIValueType]float64{
			// enum.KPIValueTypeReach:       float64(uniqueViews), // Handled by consumer
			// enum.KPIValueTypeViews:       float64(views),       // Handled by consumer
			// enum.KPIValueTypeUniqueViews: float64(uniqueViews), // Handled by consumer
			enum.KPIValueTypeComments:   float64(comments),
			enum.KPIValueTypeLikes:      float64(likes),
			enum.KPIValueTypeShares:     float64(shares),
			enum.KPIValueTypeEngagement: float64(likes + comments + shares),
		}

		rawMetrics := map[string]any{
			"views":        views,
			"unique_views": uniqueViews,
			"comments":     comments,
			"likes":        likes,
			"shares":       shares,
		}

		collector.addContentChannelUpdate(cc, rawMetrics, mappedMetrics)
	}

	// Update channel-level metrics
	rawChannelMetrics := map[string]any{
		"views":            totalViews,
		"unique_views":     totalUniqueViews,
		"comments":         totalComments,
		"likes":            totalLikes,
		"shares":           totalShares,
		"content_count":    len(ccMap),
		"platform":         "website",
		"last_aggregation": time.Now().Format(time.RFC3339),
	}

	mappedChannelMetrics := map[string]float64{
		string(enum.KPIValueTypeReach):       float64(totalUniqueViews),
		string(enum.KPIValueTypeViews):       float64(totalViews),
		string(enum.KPIValueTypeUniqueViews): float64(totalUniqueViews),
		string(enum.KPIValueTypeComments):    float64(totalComments),
		string(enum.KPIValueTypeLikes):       float64(totalLikes),
		string(enum.KPIValueTypeShares):      float64(totalShares),
		string(enum.KPIValueTypeEngagement):  float64(totalLikes + totalComments + totalShares),
	}

	collector.addChannelUpdate(&channel, rawChannelMetrics, mappedChannelMetrics)

	zap.L().Debug("Processed website channel metrics",
		zap.String("channel_id", channel.ID.String()),
		zap.Int64("total_views", totalViews),
		zap.Int64("total_comments", totalComments),
		zap.Int("content_count", len(ccMap)))
}

// getWebsiteContentMetrics retrieves metrics for a single website content channel
// from stored Metrics (likes/shares/views/comments) which are maintained by consumers
func (j *ContentMetricsPollerJob) getWebsiteContentMetrics(
	_ context.Context,
	cc *model.ContentChannel,
) (views, uniqueViews, comments, likes, shares int64) {
	if cc.Metrics == nil {
		return 0, 0, 0, 0, 0
	}

	// Use CurrentMapped for all metrics as they are updated by consumers/services
	if cc.Metrics.CurrentMapped != nil {
		if v, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeViews]; ok {
			views = int64(v)
		}
		if uv, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeUniqueViews]; ok {
			uniqueViews = int64(uv)
		}
		if c, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeComments]; ok {
			comments = int64(c)
		}
		if l, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeLikes]; ok {
			likes = int64(l)
		}
		if s, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeShares]; ok {
			shares = int64(s)
		}
	}

	// Fallback/Supplement from CurrentFetched for website-specific format if needed
	// (Though CurrentMapped should be the source of truth now)
	if cc.Metrics.CurrentFetched != nil {
		// Ensure views are synced if missing in Mapped
		if views == 0 {
			if v, ok := cc.Metrics.CurrentFetched["views_count"]; ok {
				views = castToInt64(v)
			}
		}

		// Ensure shares are synced
		if shares == 0 {
			if s, ok := cc.Metrics.CurrentFetched["shares_count"]; ok {
				shares = castToInt64(s)
			}
		}

		// Ensure likes from reaction summary
		if likes == 0 {
			if summary, ok := cc.Metrics.CurrentFetched["reaction_summary"].(map[string]any); ok {
				var totalLikes int64
				for _, count := range summary {
					totalLikes += castToInt64(count)
				}
				likes = totalLikes
			}
		}
	}

	return
}

func castToInt64(v any) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}

func (j *ContentMetricsPollerJob) updateExternalPostIDForContentChannel(ctx context.Context, cc *model.ContentChannel, baseURL, newExternalPostID, attachmentMediaType string) error {
	var (
		externalPostID   = newExternalPostID
		externalPostURL  = utils.PtrOrNil(fmt.Sprintf("%s/%s", baseURL, newExternalPostID))
		externalPostType = cc.ExternalPostType
		metadata         = cc.Metadata
	)
	if metadata == nil {
		metadata = &model.ContentChannelMetadata{}
	}

	var allowedExternalPostTypes []enum.ExternalPostType
	switch attachmentMediaType {
	case "video":
		metadata.VideoID = cc.ExternalPostID
		allowedExternalPostTypes = []enum.ExternalPostType{enum.ExternalPostTypeVideo, enum.ExternalPostTypeLongVideo}
	case "photo":
		if cc.ExternalPostID != nil {
			newPhotoIDs := make([]string, 0)
			if metadata != nil && len(metadata.PhotoIDs) > 0 {
				newPhotoIDs = append(newPhotoIDs, metadata.PhotoIDs...)
			}
			newPhotoIDs = append(newPhotoIDs, *cc.ExternalPostID)
			metadata.PhotoIDs = newPhotoIDs
		}
		allowedExternalPostTypes = []enum.ExternalPostType{enum.ExternalPostTypeMultiImage, enum.ExternalPostTypeSingleImage}
	}

	if externalPostType != nil && !utils.ContainsSlice(allowedExternalPostTypes, *externalPostType) {
		switch attachmentMediaType {
		case "video":
			externalPostType = utils.PtrOrNil(enum.ExternalPostTypeVideo)
		case "photo":
			externalPostType = utils.PtrOrNil(enum.ExternalPostTypeMultiImage)
		default:
			externalPostType = utils.PtrOrNil(enum.ExternalPostTypeText)
		}
	} else if cc.ExternalPostType == nil {
		externalPostType = utils.PtrOrNil(enum.ExternalPostTypeText)
	}

	return helper.WithTransaction(ctx, j.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err := uow.ContentChannels().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", cc.ID)
		}, map[string]any{
			"external_post_id":   externalPostID,
			"external_post_url":  externalPostURL,
			"external_post_type": externalPostType,
			"metadata":           metadata,
		}); err != nil {
			zap.L().Error("Failed to update content channel external post ID",
				zap.String("content_channel_id", cc.ID.String()),
				zap.Error(err))
			return err
		}

		return nil
	})
}

// endregion
