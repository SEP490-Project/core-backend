package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SocialMetricsPollerJob struct {
	db                 *gorm.DB
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	kpiMetricsRepo     irepository.GenericRepository[model.KPIMetrics]
	channelService     iservice.ChannelService
	facebookProxy      iproxies.FacebookProxy
	tiktokProxy        iproxies.TikTokProxy
	cronScheduler      *cron.Cron
	lastRunTime        time.Time
	cronExpr           string
	enabled            bool
	entryID            cron.EntryID
}

func NewSocialMetricsPollerJob(
	db *gorm.DB,
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	kpiMetricsRepo irepository.GenericRepository[model.KPIMetrics],
	channelService iservice.ChannelService,
	facebookProxy iproxies.FacebookProxy,
	tiktokProxy iproxies.TikTokProxy,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	cronExpr := adminConfig.SocialMetricsPollerIntervalCronExpr
	if cronExpr == "" {
		cronExpr = "* */30 * * * *" // Default to 1 hour
	}

	return &SocialMetricsPollerJob{
		db:                 db,
		contentChannelRepo: contentChannelRepo,
		kpiMetricsRepo:     kpiMetricsRepo,
		channelService:     channelService,
		facebookProxy:      facebookProxy,
		tiktokProxy:        tiktokProxy,
		cronScheduler:      cronScheduler,
		cronExpr:           cronExpr,
		enabled:            adminConfig.SocialMetricsPollerEnabled,
	}
}

// GetLastRunTime implements CronJob.
func (j *SocialMetricsPollerJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// Initialize implements CronJob.
func (j *SocialMetricsPollerJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("SocialMetricsPollerJob is disabled via admin config")
		return nil
	}
	zap.L().Debug("Initializing SocialMetricsPollerJob...")

	entryID, err := j.cronScheduler.AddFunc(j.cronExpr, j.Run)
	if err != nil {
		return fmt.Errorf("failed to schedule SocialMetricsPollerJob: %w", err)
	}
	j.entryID = entryID
	zap.L().Info("SocialMetricsPollerJob scheduled", zap.String("cron_expr", j.cronExpr))
	return nil
}

// Stop implements CronJob.
func (j *SocialMetricsPollerJob) Stop() {
	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		zap.L().Info("SocialMetricsPollerJob stopped")
	}
}

// IsEnabled implements CronJob.
func (j *SocialMetricsPollerJob) IsEnabled() bool {
	return j.enabled
}

// Restart implements CronJob.
func (j *SocialMetricsPollerJob) Restart(adminConfig *config.AdminConfig) error {
	j.Stop()
	j.cronExpr = adminConfig.SocialMetricsPollerIntervalCronExpr
	j.enabled = adminConfig.SocialMetricsPollerEnabled
	return j.Initialize()
}

// SetEnabled implements CronJob.
func (j *SocialMetricsPollerJob) SetEnabled(enabled bool) {
	j.enabled = enabled
}

// Run executes the job logic
func (j *SocialMetricsPollerJob) Run() {
	ctx := context.Background()
	zap.L().Info("Starting SocialMetricsPollerJob execution")

	// 1. Fetch active content channels
	channels, err := j.fetchActiveContentChannels(ctx)
	if err != nil {
		zap.L().Error("Failed to fetch active content channels", zap.Error(err))
		return
	}

	if len(channels) == 0 {
		zap.L().Info("No active content channels to process")
		return
	}

	// 2. Process each channel (fetch metrics)
	affectedCampaignIDs := make(map[uuid.UUID]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency to avoid rate limits
	sem := make(chan struct{}, 5) // Max 5 concurrent requests

	for i := range channels {
		cc := &channels[i]
		wg.Add(1)
		go func(cc *model.ContentChannel) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			updated, err := j.processContentChannel(ctx, cc)
			if err != nil {
				zap.L().Error("Failed to process content channel", zap.String("id", cc.ID.String()), zap.Error(err))
				return
			}

			if updated {
				campaignID, err := j.getCampaignID(ctx, cc.ContentID)
				if err == nil {
					mu.Lock()
					affectedCampaignIDs[campaignID] = true
					mu.Unlock()
				}
			}
		}(cc)
	}
	wg.Wait()

	// 3. Aggregate metrics for affected campaigns
	for campaignID := range affectedCampaignIDs {
		if err := j.aggregateCampaignMetrics(ctx, campaignID); err != nil {
			zap.L().Error("Failed to aggregate campaign metrics", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		}
	}

	j.lastRunTime = time.Now()
	zap.L().Info("SocialMetricsPollerJob execution completed")
}

func (j *SocialMetricsPollerJob) fetchActiveContentChannels(ctx context.Context) ([]model.ContentChannel, error) {
	var channels []model.ContentChannel
	err := j.db.WithContext(ctx).
		Preload("Channel").
		Where("auto_post_status = ?", enum.AutoPostStatusPosted).
		Where("external_post_id IS NOT NULL").
		Find(&channels).Error
	return channels, err
}

func (j *SocialMetricsPollerJob) processContentChannel(ctx context.Context, cc *model.ContentChannel) (bool, error) {
	if cc.Channel == nil {
		return false, fmt.Errorf("channel is nil for content channel %s", cc.ID)
	}

	// Get access token
	accessToken, err := j.channelService.GetDecryptedToken(ctx, cc.Channel.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get access token: %w", err)
	}

	var newMetrics map[string]float64

	// Determine platform and fetch metrics
	// Check Channel Name or Code prefix
	if strings.HasPrefix(strings.ToUpper(cc.Channel.Code), "FB") || strings.HasPrefix(strings.ToUpper(cc.Channel.Name), "FACEBOOK") {
		newMetrics, err = j.fetchFacebookMetrics(ctx, cc, accessToken)
	} else if strings.HasPrefix(strings.ToUpper(cc.Channel.Code), "TT") || strings.HasPrefix(strings.ToUpper(cc.Channel.Name), "TIKTOK") {
		newMetrics, err = j.fetchTikTokMetrics(ctx, cc, accessToken)
	} else {
		// Try to guess from ExternalPostType
		if cc.ExternalPostType != nil {
			if *cc.ExternalPostType == enum.ExternalPostTypeVideo {
				zap.L().Warn("Unknown platform for content channel, skipping",
					zap.String("content_channel_id", cc.ID.String()),
					zap.String("platform", string(cc.Channel.Code)))
			}
		}
		return false, nil // Unknown platform
	}

	if err != nil {
		return false, err
	}

	if newMetrics == nil {
		return false, nil
	}

	// Update Metrics in DB
	return j.updateContentChannelMetrics(ctx, cc, newMetrics)
}

func (j *SocialMetricsPollerJob) fetchFacebookMetrics(ctx context.Context, cc *model.ContentChannel, accessToken string) (map[string]float64, error) {
	if cc.ExternalPostID == nil {
		return nil, nil
	}

	metrics := make(map[string]float64)
	var err error

	// Determine content type and fetch appropriate metrics
	// Default to Post metrics if type is unknown or standard post
	isVideo := false
	if cc.ExternalPostType != nil {
		switch *cc.ExternalPostType {
		case enum.ExternalPostTypeVideo, enum.ExternalPostTypeLongVideo:
			isVideo = true
		}
	}

	if isVideo {
		// Video/Reel Metrics
		videoMetrics := []string{
			"post_video_likes_by_reaction_type",
			"post_video_avg_time_watched",
			"post_video_social_actions",
			"post_video_view_time",
			"post_impressions_unique",
			"blue_reels_play_count",
			"fb_reels_total_plays",
			"fb_reels_replay_count",
			"post_video_followers",
			// "post_video_retention_graph", // Graph data, not scalar, skipping for now
		}

		var resp *dtos.FacebookVideoInsightsResponse
		resp, err = j.facebookProxy.GetVideoInsights(ctx, *cc.ExternalPostID, accessToken, videoMetrics, dtos.FacebookInsightsPeriodLifetime)
		if err == nil && resp != nil {
			j.processFacebookMetricData(resp.Data, metrics)
		}
	} else {
		// Standard Post Metrics
		postMetrics := []string{
			"post_clicks",
			"post_reactions_by_type_total",
			"post_media_view",
		}

		var resp *dtos.FacebookPostMetricsResponse
		resp, err = j.facebookProxy.GetPostMetrics(ctx, *cc.ExternalPostID, accessToken, postMetrics, dtos.FacebookInsightsPeriodLifetime)
		if err == nil && resp != nil {
			j.processFacebookMetricData(resp.Data, metrics)
		}
	}

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (j *SocialMetricsPollerJob) processFacebookMetricData(data []dtos.FacebookMetricData, metrics map[string]float64) {
	for _, d := range data {
		if len(d.Values) > 0 {
			val := d.Values[0].Value
			switch v := val.(type) {
			case float64:
				metrics[d.Name] = v
			case int:
				metrics[d.Name] = float64(v)
			case map[string]any:
				// Sum up values if it's a breakdown (e.g. reactions)
				var total float64
				for _, subVal := range v {
					if f, ok := subVal.(float64); ok {
						total += f
					}
				}
				metrics[d.Name] = total
			}
		}
	}
}

func (j *SocialMetricsPollerJob) fetchTikTokMetrics(ctx context.Context, cc *model.ContentChannel, accessToken string) (map[string]float64, error) {
	if cc.ExternalPostID == nil {
		return nil, nil
	}

	metricsResp, err := j.tiktokProxy.GetVideoMetrics(ctx, *cc.ExternalPostID, accessToken)
	if err != nil {
		return nil, err
	}

	if len(metricsResp.Data.Videos) == 0 {
		return nil, fmt.Errorf("video not found")
	}

	video := metricsResp.Data.Videos[0]
	metrics := make(map[string]float64)
	metrics["view_count"] = float64(video.ViewCount)
	metrics["like_count"] = float64(video.LikeCount)
	metrics["comment_count"] = float64(video.CommentCount)
	metrics["share_count"] = float64(video.ShareCount)

	return metrics, nil
}

func (j *SocialMetricsPollerJob) updateContentChannelMetrics(ctx context.Context, cc *model.ContentChannel, newMetrics map[string]float64) (bool, error) {
	// Parse existing metrics
	var currentMetrics model.ContentChannelMetrics
	if len(cc.Metrics) > 0 {
		if err := json.Unmarshal(cc.Metrics, &currentMetrics); err != nil {
			// If invalid, start fresh
			currentMetrics = model.ContentChannelMetrics{}
		}
	}

	// Update LastFetched with previous Current
	currentMetrics.LastFetched = currentMetrics.Current
	currentMetrics.Current = newMetrics

	// Marshal back
	metricsJSON, err := json.Marshal(currentMetrics)
	if err != nil {
		return false, err
	}

	// Update DB
	cc.Metrics = datatypes.JSON(metricsJSON)
	if err := j.contentChannelRepo.Update(ctx, cc); err != nil {
		return false, err
	}

	return true, nil
}

func (j *SocialMetricsPollerJob) getCampaignID(ctx context.Context, contentID uuid.UUID) (uuid.UUID, error) {
	// Content -> Task -> Milestone -> Campaign
	var content model.Content
	if err := j.db.WithContext(ctx).
		Preload("Task.Milestone").
		Where("id = ?", contentID).
		First(&content).Error; err != nil {
		return uuid.Nil, err
	}

	if content.Task == nil || content.Task.Milestone == nil {
		return uuid.Nil, fmt.Errorf("content %s not linked to campaign", contentID)
	}

	return content.Task.Milestone.CampaignID, nil
}

func (j *SocialMetricsPollerJob) aggregateCampaignMetrics(ctx context.Context, campaignID uuid.UUID) error {
	// 1. Get all contents for this campaign
	// Query: Campaign -> Milestone -> Task -> Content -> ContentChannel

	type Result struct {
		Metrics datatypes.JSON
	}

	var results []Result
	err := j.db.WithContext(ctx).
		Table("content_channels").
		Joins("JOIN contents ON content_channels.content_id = contents.id").
		Joins("JOIN tasks ON contents.task_id = tasks.id").
		Joins("JOIN milestones ON tasks.milestone_id = milestones.id").
		Where("milestones.campaign_id = ?", campaignID).
		Select("content_channels.metrics").
		Scan(&results).Error

	if err != nil {
		return err
	}

	// 2. Sum up metrics
	totals := make(map[string]float64)
	for _, res := range results {
		if len(res.Metrics) == 0 {
			continue
		}
		var cm model.ContentChannelMetrics
		if err := json.Unmarshal(res.Metrics, &cm); err == nil {
			for k, v := range cm.Current {
				totals[k] += v
			}
		}
	}

	// 3. Map to KPI types and insert
	// Mapping:
	// Facebook: post_reactions_like_total -> LIKES, post_clicks -> CTR (needs calculation?), post_impressions -> IMPRESSIONS
	// TikTok: like_count -> LIKES, view_count -> IMPRESSIONS (approx), share_count -> SHARES, comment_count -> COMMENTS

	kpiMap := map[enum.KPIValueType]float64{
		enum.KPIValueTypeLikes:       0,
		enum.KPIValueTypeComments:    0,
		enum.KPIValueTypeShares:      0,
		enum.KPIValueTypeImpressions: 0,
		enum.KPIValueTypeReach:       0,
		enum.KPIValueTypeEngagement:  0,
	}

	for k, v := range totals {
		switch k {
		case "post_reactions_like_total", "like_count", "post_reactions_love_total", "post_reactions_wow_total", "post_reactions_haha_total":
			kpiMap[enum.KPIValueTypeLikes] += v
			kpiMap[enum.KPIValueTypeEngagement] += v
		case "post_comments", "comment_count": // Facebook doesn't give comments count in the list I used?
			// I used "post_post_engagements" for page, but for post I used reactions.
			// Facebook post metrics: post_clicks, post_impressions, reactions.
			// I should add "post_comments" to Facebook metrics list if available?
			// Or just map what I have.
			kpiMap[enum.KPIValueTypeComments] += v
			kpiMap[enum.KPIValueTypeEngagement] += v
		case "post_shares", "share_count": // Facebook shares?
			kpiMap[enum.KPIValueTypeShares] += v
			kpiMap[enum.KPIValueTypeEngagement] += v
		case "post_impressions", "view_count":
			kpiMap[enum.KPIValueTypeImpressions] += v
		case "post_impressions_unique":
			kpiMap[enum.KPIValueTypeReach] += v
		case "post_clicks":
			// Clicks contribute to engagement?
			kpiMap[enum.KPIValueTypeEngagement] += v
		}
	}

	// Insert into kpi_metrics
	now := time.Now()
	for kpiType, value := range kpiMap {
		if value > 0 {
			metric := &model.KPIMetrics{
				ReferenceID:   campaignID,
				ReferenceType: enum.KPIReferenceTypeCampaign,
				Type:          kpiType,
				Value:         value,
				RecordedDate:  now,
			}
			if err := j.kpiMetricsRepo.Add(ctx, metric); err != nil {
				zap.L().Error("Failed to add KPI metric", zap.Error(err))
			}
		}
	}

	return nil
}
