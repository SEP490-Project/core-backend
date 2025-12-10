package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
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
type ContentMetricsPollerJob struct {
	db                  *gorm.DB
	unitOfWork          irepository.UnitOfWork
	contentChannelRepo  irepository.GenericRepository[model.ContentChannel]
	channelRepo         irepository.GenericRepository[model.Channel]
	kpiMetricsRepo      irepository.GenericRepository[model.KPIMetrics]
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

// NewContentMetricsPollerJob creates a new instance of ContentMetricsPollerJob
func NewContentMetricsPollerJob(
	db *gorm.DB,
	unitOfWork irepository.UnitOfWork,
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	channelRepo irepository.GenericRepository[model.Channel],
	kpiMetricsRepo irepository.GenericRepository[model.KPIMetrics],
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
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Panic recovered in ContentMetricsPollerJob, disabling job",
				zap.Any("recover", r))
			j.SetEnabled(false)
		}
	}()

	ctx := context.Background()
	j.lastRunTime = time.Now()

	zap.L().Info("ContentMetricsPollerJob starting...")

	// Get active channels by platform
	fbChannels, tikTokChannels, err := j.getActiveChannels(ctx)
	if err != nil {
		zap.L().Error("Failed to get active channels", zap.Error(err))
		return
	}

	zap.L().Info("Found active channels",
		zap.Int("facebook_channels", len(fbChannels)),
		zap.Int("tiktok_channels", len(tikTokChannels)))

	// Execute all tasks with continue-on-failure
	var wg sync.WaitGroup
	tasks := []struct {
		name string
		fn   func(context.Context)
	}{
		{"Facebook Page Metrics", func(ctx context.Context) { j.fetchFacebookPageMetrics(ctx, fbChannels) }},
		{"Facebook Posts Metrics", func(ctx context.Context) { j.fetchFacebookPostsMetrics(ctx, fbChannels) }},
		{"TikTok User Metrics", func(ctx context.Context) { j.fetchTikTokUserMetrics(ctx, tikTokChannels) }},
		{"TikTok Video List Metrics", func(ctx context.Context) { j.fetchTikTokVideoListMetrics(ctx, tikTokChannels) }},
	}

	for _, task := range tasks {
		wg.Add(1)
		go func(taskName string, taskFn func(context.Context)) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					zap.L().Error("Task panicked, continuing with others",
						zap.String("task", taskName),
						zap.Any("recover", r))
				}
			}()

			zap.L().Debug("Starting task", zap.String("task", taskName))
			taskFn(ctx)
			zap.L().Debug("Completed task", zap.String("task", taskName))
		}(task.name, task.fn)
	}

	wg.Wait()
	zap.L().Info("ContentMetricsPollerJob completed")
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
func (j *ContentMetricsPollerJob) getActiveChannels(ctx context.Context) ([]model.Channel, []model.Channel, error) {
	channels, _, err := j.channelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).
				Where("hashed_access_token IS NOT NULL").
				Where("code IN (?, ?)", "FACEBOOK", "TIKTOK")
		},
		nil, 0, 0)

	if err != nil {
		return nil, nil, err
	}

	var fbChannels, tikTokChannels []model.Channel
	for _, ch := range channels {
		switch ch.Code {
		case "FACEBOOK":
			fbChannels = append(fbChannels, ch)
		case "TIKTOK":
			tikTokChannels = append(tikTokChannels, ch)
		}
	}

	return fbChannels, tikTokChannels, nil
}

// waitForFBRateLimit waits for Facebook rate limiter before making API call
func (j *ContentMetricsPollerJob) waitForFBRateLimit(ctx context.Context) error {
	return j.fbRateLimiter.Wait(ctx)
}

// persistChannelMetrics updates Channel.Metrics and inserts into kpi_metrics
func (j *ContentMetricsPollerJob) persistChannelMetrics(
	ctx context.Context,
	channel *model.Channel,
	rawMetrics map[string]any,
	mappedMetrics map[string]float64,
) error {
	// Build metrics JSON for Channel.Metrics
	metricsData := map[string]any{
		"current_fetched": rawMetrics,
		"current_mapped":  mappedMetrics,
		"last_updated_at": time.Now().Format(time.RFC3339),
	}

	metricsJSON, err := json.Marshal(metricsData)
	if err != nil {
		return fmt.Errorf("failed to marshal channel metrics: %w", err)
	}

	// Update channel metrics
	channel.Metrics = datatypes.JSON(metricsJSON)
	if err := j.channelRepo.Update(ctx, channel); err != nil {
		return fmt.Errorf("failed to update channel metrics: %w", err)
	}

	// Insert into kpi_metrics table
	return j.persistKPIMetrics(ctx, channel.ID, enum.KPIReferenceTypeChannel, mappedMetrics)
}

// persistContentChannelMetrics updates ContentChannel.Metrics and inserts into kpi_metrics
func (j *ContentMetricsPollerJob) persistContentChannelMetrics(
	ctx context.Context,
	contentChannel *model.ContentChannel,
	rawMetrics map[string]any,
	mappedMetrics map[string]float64,
) error {
	// Get existing metrics to preserve history
	existingMetrics, _ := contentChannel.GetMetrics()
	if existingMetrics == nil {
		existingMetrics = &model.ContentChannelMetrics{}
	}

	// Move current to last
	existingMetrics.LastFetched = existingMetrics.CurrentFetched
	existingMetrics.LastMapped = existingMetrics.CurrentMapped

	// Set new current values
	existingMetrics.CurrentFetched = rawMetrics
	existingMetrics.CurrentMapped = mappedMetrics

	metricsJSON, err := json.Marshal(existingMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal content channel metrics: %w", err)
	}

	contentChannel.Metrics = datatypes.JSON(metricsJSON)
	if err := j.contentChannelRepo.Update(ctx, contentChannel); err != nil {
		return fmt.Errorf("failed to update content channel metrics: %w", err)
	}

	// Insert into kpi_metrics table
	return j.persistKPIMetrics(ctx, contentChannel.ID, enum.KPIReferenceTypeContentChannel, mappedMetrics)
}

// persistKPIMetrics inserts metrics into kpi_metrics table
func (j *ContentMetricsPollerJob) persistKPIMetrics(
	ctx context.Context,
	referenceID uuid.UUID,
	referenceType enum.KPIReferenceType,
	metrics map[string]float64,
) error {
	now := time.Now()

	for metricName, metricValue := range metrics {
		// Convert string metric name to KPIValueType enum
		kpiType := enum.KPIValueType(metricName)
		if !kpiType.IsValid() {
			zap.L().Warn("Invalid KPI metric type, skipping",
				zap.String("metric_name", metricName))
			continue
		}

		kpiMetric := &model.KPIMetrics{
			ReferenceID:   referenceID,
			ReferenceType: referenceType,
			Type:          kpiType,
			Value:         metricValue,
			RecordedDate:  now,
		}

		if err := j.kpiMetricsRepo.Add(ctx, kpiMetric); err != nil {
			zap.L().Warn("Failed to insert KPI metric",
				zap.String("reference_id", referenceID.String()),
				zap.String("metric_name", metricName),
				zap.Error(err))
			// Continue with other metrics even if one fails
		}
	}

	return nil
}

// endregion

// region: ======== Facebook Metrics Methods ========

// fetchFacebookPageMetrics fetches page-level metrics for Facebook channels
func (j *ContentMetricsPollerJob) fetchFacebookPageMetrics(ctx context.Context, channels []model.Channel) {
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

		// For channel-level metrics, we use a simpler mapping
		// Followers is the key metric for channels
		mappedMetrics := map[string]float64{
			enum.KPIValueTypeFollowers.String(): float64(pageInfo.FollowersCount),
		}

		// Persist metrics
		if err := j.persistChannelMetrics(ctx, channel, rawMetrics, mappedMetrics); err != nil {
			zap.L().Error("Failed to persist Facebook page metrics",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		zap.L().Info("Facebook page metrics persisted",
			zap.String("channel_name", channel.Name),
			zap.Int("fan_count", pageInfo.FanCount),
			zap.Int("followers_count", pageInfo.FollowersCount))
	}
}

// fetchFacebookPostsMetrics fetches post metrics from Facebook pages
func (j *ContentMetricsPollerJob) fetchFacebookPostsMetrics(ctx context.Context, channels []model.Channel) {
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

		// Fetch posts with metrics using pagination
		fields := "id,created_time,message,reactions.limit(0).summary(total_count),comments.limit(0).summary(total_count),shares,attachments{media_type,target}"
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
				j.processPostMetrics(ctx, channel, accessToken, &post)
				postsProcessed++
			}

			// Check for next page
			if postsResp.Paging == nil || postsResp.Paging.Next == nil {
				break
			}
			cursor = &postsResp.Paging.Cursors.After
		}

		zap.L().Info("Facebook posts metrics processed",
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
) {
	// Find matching ContentChannel by ExternalPostID
	contentChannels, _, err := j.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("channel_id = ?", channel.ID).
				Where("external_post_id = ?", post.ID)
		},
		nil, 1, 1)

	if err != nil || len(contentChannels) == 0 {
		// Post not tracked in our system, skip
		return
	}

	cc := &contentChannels[0]

	// Extract metrics from post response
	rawMetrics := map[string]any{
		"post_id": post.ID,
	}

	reactions := 0
	comments := 0
	shares := 0

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

	// Map to KPI metrics using helper
	mappedMetrics := make(map[string]float64)

	// Reactions -> Likes + Engagement
	if reactions > 0 {
		maps.Copy(mappedMetrics, helper.MapFacebookMetricsToKPIField("post_reactions_by_type_total", float64(reactions)))
	}

	// Comments -> Comments + Engagement
	if comments > 0 {
		mappedMetrics[enum.KPIValueTypeComments.String()] = float64(comments)
		if existing, ok := mappedMetrics[enum.KPIValueTypeEngagement.String()]; ok {
			mappedMetrics[enum.KPIValueTypeEngagement.String()] = existing + float64(comments)
		} else {
			mappedMetrics[enum.KPIValueTypeEngagement.String()] = float64(comments)
		}
	}

	// Shares -> Shares + Engagement
	if shares > 0 {
		mappedMetrics[enum.KPIValueTypeShares.String()] = float64(shares)
		if existing, ok := mappedMetrics[enum.KPIValueTypeEngagement.String()]; ok {
			mappedMetrics[enum.KPIValueTypeEngagement.String()] = existing + float64(shares)
		} else {
			mappedMetrics[enum.KPIValueTypeEngagement.String()] = float64(shares)
		}
	}

	// If video post, fetch video insights
	if post.Attachments != nil && len(post.Attachments.Data) > 0 {
		for _, attachment := range post.Attachments.Data {
			if attachment.MediaType == "video" && attachment.Target != nil {
				j.fetchAndMergeVideoInsights(ctx, accessToken, attachment.Target.ID, rawMetrics, mappedMetrics)
			}
		}
	}

	// Persist metrics
	if err := j.persistContentChannelMetrics(ctx, cc, rawMetrics, mappedMetrics); err != nil {
		zap.L().Error("Failed to persist Facebook post metrics",
			zap.String("post_id", post.ID),
			zap.Error(err))
	}
}

// fetchAndMergeVideoInsights fetches video insights and merges into existing metrics
func (j *ContentMetricsPollerJob) fetchAndMergeVideoInsights(
	ctx context.Context,
	accessToken string,
	videoID string,
	rawMetrics map[string]any,
	mappedMetrics map[string]float64,
) {
	// Wait for rate limit
	if err := j.waitForFBRateLimit(ctx); err != nil {
		return
	}

	metrics := []string{"total_video_views", "total_video_impressions"}
	insights, err := j.facebookProxy.GetVideoInsights(ctx, videoID, accessToken, metrics, dtos.FacebookInsightsPeriodLifetime)
	if err != nil {
		zap.L().Warn("Failed to fetch video insights",
			zap.String("video_id", videoID),
			zap.Error(err))
		return
	}

	for _, data := range insights.Data {
		if len(data.Values) > 0 {
			if value, ok := data.Values[0].Value.(float64); ok {
				rawMetrics[data.Name] = value

				// Map video metrics
				for k, v := range helper.MapFacebookMetricsToKPIField(data.Name, value) {
					if existing, exists := mappedMetrics[k]; exists {
						mappedMetrics[k] = existing + v
					} else {
						mappedMetrics[k] = v
					}
				}
			}
		}
	}
}

// endregion

// region: ======== TikTok Metrics Methods ========

// fetchTikTokUserMetrics fetches user-level metrics for TikTok channels
func (j *ContentMetricsPollerJob) fetchTikTokUserMetrics(ctx context.Context, channels []model.Channel) {
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

		// Persist metrics
		if err := j.persistChannelMetrics(ctx, channel, rawMetrics, mappedMetrics); err != nil {
			zap.L().Error("Failed to persist TikTok user metrics",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
		}

		zap.L().Info("TikTok user metrics persisted",
			zap.String("channel_name", channel.Name),
			zap.Int64("followers", utils.DerefPtr(userProfile.Data.User.FollowerCount, 0)))
	}
}

// fetchTikTokVideoListMetrics fetches video list and metrics for TikTok channels
func (j *ContentMetricsPollerJob) fetchTikTokVideoListMetrics(ctx context.Context, channels []model.Channel) {
	for i := range channels {
		channel := &channels[i]

		accessToken, err := j.tiktokSocialService.GetTikTokAccessToken(ctx)
		if err != nil {
			zap.L().Error("Failed to get TikTok access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			continue
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
				j.processTikTokVideoMetrics(ctx, channel, &video)
				videosProcessed++
			}

			// Check for more videos
			if !videosResp.Data.HasMore {
				break
			}
			cursor = &videosResp.Data.Cursor
		}

		zap.L().Info("TikTok videos metrics processed",
			zap.String("channel_name", channel.Name),
			zap.Int("videos_processed", videosProcessed))
	}
}

// processTikTokVideoMetrics processes metrics for a single TikTok video
func (j *ContentMetricsPollerJob) processTikTokVideoMetrics(
	ctx context.Context,
	channel *model.Channel,
	video *dtos.TikTokVideoItem,
) {
	// Find matching ContentChannel by ExternalPostID
	contentChannels, _, err := j.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("channel_id = ?", channel.ID).
				Where("external_post_id = ?", video.ID)
		},
		nil, 1, 1)

	if err != nil || len(contentChannels) == 0 {
		// Video not tracked in our system, skip
		return
	}

	cc := &contentChannels[0]

	// Extract metrics
	rawMetrics := map[string]any{
		"video_id":      video.ID,
		"view_count":    video.ViewCount,
		"like_count":    video.LikeCount,
		"comment_count": video.CommentCount,
		"share_count":   video.ShareCount,
	}

	// Map to KPI metrics using helper
	mappedMetrics := make(map[string]float64)

	maps.Copy(mappedMetrics, helper.MapTikTokMetricsToKPIField("view_count", float64(video.ViewCount)))
	for k, v := range helper.MapTikTokMetricsToKPIField("like_count", float64(video.LikeCount)) {
		if existing, exists := mappedMetrics[k]; exists {
			mappedMetrics[k] = existing + v
		} else {
			mappedMetrics[k] = v
		}
	}
	for k, v := range helper.MapTikTokMetricsToKPIField("comment_count", float64(video.CommentCount)) {
		if existing, exists := mappedMetrics[k]; exists {
			mappedMetrics[k] = existing + v
		} else {
			mappedMetrics[k] = v
		}
	}
	for k, v := range helper.MapTikTokMetricsToKPIField("share_count", float64(video.ShareCount)) {
		if existing, exists := mappedMetrics[k]; exists {
			mappedMetrics[k] = existing + v
		} else {
			mappedMetrics[k] = v
		}
	}

	// Persist metrics
	if err := j.persistContentChannelMetrics(ctx, cc, rawMetrics, mappedMetrics); err != nil {
		zap.L().Error("Failed to persist TikTok video metrics",
			zap.String("video_id", video.ID),
			zap.Error(err))
	}
}

// endregion
