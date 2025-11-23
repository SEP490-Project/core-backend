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
	"core-backend/pkg/utils"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TikTokStatusPollerJob struct {
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	contentRepo        irepository.GenericRepository[model.Content]
	channelRepo        irepository.GenericRepository[model.Channel]
	tiktokProxy        iproxies.TikTokProxy
	channelService     iservice.ChannelService
	uow                irepository.UnitOfWork
	cronScheduler      *cron.Cron
	intervalSeconds    int
	enabled            bool
	lastRunTime        time.Time
}

func NewTikTokStatusPollerJob(
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	contentRepo irepository.GenericRepository[model.Content],
	channelRepo irepository.GenericRepository[model.Channel],
	tiktokProxy iproxies.TikTokProxy,
	channelService iservice.ChannelService,
	uow irepository.UnitOfWork,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	intervalSeconds := adminConfig.TikTokStatusPollerIntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 30 // Default to 30 seconds if not set
	}

	return &TikTokStatusPollerJob{
		contentChannelRepo: contentChannelRepo,
		contentRepo:        contentRepo,
		channelRepo:        channelRepo,
		tiktokProxy:        tiktokProxy,
		channelService:     channelService,
		uow:                uow,
		cronScheduler:      cronScheduler,
		intervalSeconds:    intervalSeconds,
		enabled:            adminConfig.TikTokStatusPollerEnabled,
		lastRunTime:        time.Now(),
	}
}

// Initialize implements CronJob.
func (j *TikTokStatusPollerJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("TikTok Status Poller Job is disabled via admin config")
		return nil
	}

	zap.L().Debug("Initializing TikTok Status Poller Job...")

	// Generate cron expression (e.g., "*/30 * * * *" for every 30 seconds)
	cronExpr := fmt.Sprintf("*/%d * * * *", j.intervalSeconds)
	zap.L().Info("Scheduling TikTok Status Poller Job",
		zap.String("cron_expression", cronExpr),
		zap.Int("interval_seconds", j.intervalSeconds))

	// Schedule the job
	_, err := j.cronScheduler.AddFunc(cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})

	if err != nil {
		zap.L().Error("Failed to schedule TikTok Status Poller Job", zap.Error(err))
		return fmt.Errorf("failed to schedule tiktok status poller job: %w", err)
	}

	return nil
}

// Run implements CronJob.
func (j *TikTokStatusPollerJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	j.lastRunTime = time.Now()

	zap.L().Debug("TikTok Status Poller Job starting...")

	// Query ContentChannels where:
	// - Channel code is TIKTOK
	// - AutoPostStatus is PENDING
	// - ExternalPostID contains publish_id (indicates async upload initiated)
	contentChannels, _, err := j.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.
				Joins("JOIN channels ON channels.id = content_channels.channel_id").
				Where("channels.code = ?", "TIKTOK").
				Where("content_channels.auto_post_status = ?", enum.AutoPostStatusPending).
				Where("content_channels.external_post_id IS NOT NULL AND content_channels.external_post_id != ''")
		},
		[]string{"Channel", "Content"},
		0, 0) // No pagination - get all pending

	if err != nil {
		zap.L().Error("Failed to query pending TikTok posts", zap.Error(err))
		return
	}

	if len(contentChannels) == 0 {
		zap.L().Debug("No pending TikTok posts to poll")
		return
	}

	zap.L().Info("Polling TikTok post status",
		zap.Int("pending_count", len(contentChannels)))

	processedCount := 0
	completedCount := 0
	failedCount := 0

	for _, cc := range contentChannels {
		// Extract publish_id from ExternalPostID
		publishID := extractPublishID(cc.ExternalPostID)
		if publishID == "" {
			zap.L().Warn("ContentChannel has no publish_id in ExternalPostID",
				zap.String("content_channel_id", cc.ID.String()),
				zap.String("external_post_id", utils.DerefPtr(cc.ExternalPostID, "")))
			continue
		}

		// Get decrypted access token
		accessToken, err := j.channelService.GetDecryptedToken(ctx, cc.Channel.Name)
		if err != nil {
			zap.L().Error("Failed to decrypt access token for TikTok channel",
				zap.String("channel_name", cc.Channel.Name),
				zap.Error(err))
			continue
		}

		// Check post status via TikTok API
		statusResp, err := j.tiktokProxy.CheckPostStatus(ctx, publishID, accessToken)
		if err != nil {
			zap.L().Error("Failed to check TikTok post status",
				zap.String("content_channel_id", cc.ID.String()),
				zap.String("publish_id", publishID),
				zap.Error(err))
			continue
		}

		processedCount++

		// Update ContentChannel based on status
		if err := j.updateContentChannelStatus(ctx, &cc, statusResp); err != nil {
			zap.L().Error("Failed to update content channel status",
				zap.String("content_channel_id", cc.ID.String()),
				zap.Error(err))
			continue
		}

		// Track completion stats
		switch statusResp.Data.Status {
		case "PUBLISH_COMPLETE":
			completedCount++
		case "FAILED":
			failedCount++
		}
	}

	zap.L().Info("TikTok Status Poller Job completed",
		zap.Int("processed", processedCount),
		zap.Int("completed", completedCount),
		zap.Int("failed", failedCount))
}

// IsEnabled implements CronJob.
func (j *TikTokStatusPollerJob) IsEnabled() bool {
	return j.enabled
}

// SetEnabled implements CronJob.
func (j *TikTokStatusPollerJob) SetEnabled(enabled bool) {
	j.enabled = enabled
}

// GetLastRunTime implements CronJob.
func (j *TikTokStatusPollerJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// updateContentChannelStatus updates the ContentChannel based on TikTok status response
func (j *TikTokStatusPollerJob) updateContentChannelStatus(
	ctx context.Context,
	cc *model.ContentChannel,
	statusResp *dtos.TikTokPostStatusResponse,
) error {
	status := string(statusResp.Data.Status)
	postID := statusResp.Data.PubliclyAvailablePostID
	failReason := statusResp.Data.FailReason

	// Begin transaction for atomic update
	uow := j.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	ccRepo := uow.ContentChannels()

	switch status {
	case "PUBLISH_COMPLETE":
		// Update to POSTED with final post URL
		cc.AutoPostStatus = enum.AutoPostStatusPosted
		if len(postID) > 0 {
			// Update ExternalPostID with the final TikTok post ID
			// cc.ExternalPostID = postID[0]
			cc.ExternalPostID = utils.PtrOrNil(fmt.Sprintf("%d", postID[0]))
		}
		cc.LastError = nil
		cc.PublishedAt = utils.PtrOrNil(time.Now())

		zap.L().Info("TikTok post completed",
			zap.String("content_channel_id", cc.ID.String()),
			zap.Any("post_id", postID))

	case "FAILED":
		// Update to FAILED with error message
		cc.AutoPostStatus = enum.AutoPostStatusFailed
		if failReason != nil {
			cc.LastError = failReason
		} else {
			defaultError := "TikTok video upload failed"
			cc.LastError = &defaultError
		}

		zap.L().Warn("TikTok post failed",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("fail_reason", utils.DerefPtr(failReason, "unknown")))

	case "PROCESSING_UPLOAD", "PROCESSING_DOWNLOAD", "PROCESSING_PUBLISH":
		// Still processing - no update needed
		zap.L().Debug("TikTok post still processing",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("status", status))
		return nil // No database update needed

	default:
		zap.L().Warn("Unknown TikTok post status",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("status", status))
		return nil // No database update needed
	}

	// Update ContentChannel in database
	if err := ccRepo.Update(ctx, cc); err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to update content channel: %w", err)
	}

	// Check if all content channels are posted, then update Content status
	if err := j.updateContentStatusIfAllPosted(ctx, uow, cc.ContentID); err != nil {
		uow.Rollback()
		return fmt.Errorf("failed to update content status: %w", err)
	}

	if err := uow.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// updateContentStatusIfAllPosted checks if all channels are posted and updates Content status
func (j *TikTokStatusPollerJob) updateContentStatusIfAllPosted(
	ctx context.Context,
	uow irepository.UnitOfWork,
	contentID uuid.UUID,
) error {
	ccRepo := uow.ContentChannels()
	contentRepo := uow.Contents()

	// Get all content channels for this content
	contentChannels, _, err := ccRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ?", contentID)
		},
		nil,
		0, 0)

	if err != nil {
		return err
	}

	// Check if all are posted
	allPosted := true
	for _, cc := range contentChannels {
		if cc.AutoPostStatus != enum.AutoPostStatusPosted &&
			cc.AutoPostStatus != enum.AutoPostStatusSkipped {
			allPosted = false
			break
		}
	}

	if allPosted {
		// Update content status to POSTED
		content, err := contentRepo.GetByID(ctx, contentID, nil)
		if err != nil {
			return err
		}

		content.Status = enum.ContentStatusPosted
		if err := contentRepo.Update(ctx, content); err != nil {
			return err
		}

		zap.L().Info("Content status updated to POSTED",
			zap.String("content_id", contentID.String()))
	}

	return nil
}

// Helper functions

func extractPublishID(externalPostID *string) string {
	if externalPostID == nil || *externalPostID == "" {
		return ""
	}

	// ExternalPostID from InitVideoPost contains publish_id
	// Expected format: "publish_id: <id>" or just the ID
	id := *externalPostID
	if strings.Contains(id, "publish_id:") {
		parts := strings.Split(id, "publish_id:")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1])
		}
	}

	return id // Assume the entire value is the publish_id
}
