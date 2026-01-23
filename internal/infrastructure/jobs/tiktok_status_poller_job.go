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
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	DefaultPublishError string = "TikTok video publish failed"
	// Upload status values for ContentChannel.Metadata
	UploadStatusPending    string = "pending"
	UploadStatusProcessing string = "processing"
	UploadStatusCompleted  string = "completed"
	UploadStatusFailed     string = "failed"
)

type TikTokStatusPollerJob struct {
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	contentRepo        irepository.GenericRepository[model.Content]
	channelRepo        irepository.GenericRepository[model.Channel]
	tiktokProxy        iproxies.TikTokProxy
	channelService     iservice.ChannelService
	uow                irepository.UnitOfWork
	cronScheduler      *cron.Cron
	cronExpr           string
	enabled            bool
	lastRunTime        time.Time
	entryID            cron.EntryID
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
	intervalSeconds := adminConfig.TikTokStatusPollerCronExpr
	if intervalSeconds == "" {
		intervalSeconds = "*/30 * * * * *" // Default to 30 seconds if not set
	}

	return &TikTokStatusPollerJob{
		contentChannelRepo: contentChannelRepo,
		contentRepo:        contentRepo,
		channelRepo:        channelRepo,
		tiktokProxy:        tiktokProxy,
		channelService:     channelService,
		uow:                uow,
		cronScheduler:      cronScheduler,
		cronExpr:           intervalSeconds,
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

	zap.L().Info("Scheduling TikTok Status Poller Job",
		zap.String("cron_expression", j.cronExpr))

	// Schedule the job
	entryID, err := j.cronScheduler.AddFunc(j.cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})
	j.entryID = entryID

	if err != nil {
		zap.L().Error("Failed to schedule TikTok Status Poller Job", zap.Error(err))
		return fmt.Errorf("failed to schedule tiktok status poller job: %w", err)
	}

	return nil
}

// Run implements CronJob.
func (j *TikTokStatusPollerJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Panic recovered in TikTok Status Poller Job, disabling job to negate further panics", zap.Any("recover", r))
			j.SetEnabled(false)
		}
	}()

	ctx := context.Background()

	j.lastRunTime = time.Now()

	zap.L().Debug("TikTok Status Poller Job starting...")

	// Query ContentChannels where:
	// - Channel code is TIKTOK
	// - AutoPostStatus is IN_PROGRESS
	// - Metadata->>'upload_id' IS NOT NULL (indicates async upload initiated)
	// - Metadata->>'upload_status' is 'processing' (not yet completed/failed)
	contentChannels, _, err := j.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.
				Joins("JOIN channels ON channels.id = content_channels.channel_id").
				Where("channels.code = ?", "TIKTOK").
				Where("content_channels.auto_post_status = ? OR (content_channels.metadata->>'upload_id' IS NOT NULL AND content_channels.metadata->>'upload_status' = ?)",
					enum.AutoPostStatusInProgress, UploadStatusProcessing)
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

	// Extract publish_id from ExternalPostID
	// Get decrypted access token
	accessToken, err := j.channelService.GetDecryptedToken(ctx, "TIKTOK")
	if err != nil {
		zap.L().Error("Failed to decrypt access token for TikTok channel",
			zap.String("channel_name", "TIKTOK"),
			zap.Error(err))
		return
	}
	for _, cc := range contentChannels {
		// Check post status via TikTok API
		// Extract upload_id (publish_id) from metadata instead of ExternalPostID
		metadata, err := cc.GetMetadata()
		if err != nil {
			zap.L().Error("Failed to parse metadata for TikTok content channel",
				zap.String("content_channel_id", cc.ID.String()),
				zap.Error(err))
			continue
		}
		if metadata.UploadID == nil || *metadata.UploadID == "" {
			zap.L().Warn("Content channel has nil or empty upload_id in metadata, skipping",
				zap.String("content_channel_id", cc.ID.String()),
				zap.String("channel_name", cc.Channel.Name),
				zap.String("content_id", cc.ContentID.String()))
			continue
		}

		statusResp, err := j.tiktokProxy.CheckPostStatus(ctx, *metadata.UploadID, accessToken)
		if err != nil {
			zap.L().Error("Failed to check TikTok post status",
				zap.String("content_channel_id", cc.ID.String()),
				zap.String("upload_id", utils.DerefPtr(metadata.UploadID, "UNKNOWN")),
				zap.Error(err))
			continue
		}
		if statusResp == nil || len(statusResp.Data.PubliclyAvailablePostID) == 0 || statusResp.Data.FailReason != nil {
			zap.L().Warn("Received invalid TikTok post status response",
				zap.String("content_channel_id", cc.ID.String()),
				zap.String("upload_id", utils.DerefPtr(metadata.UploadID, "UNKNOWN")),
				zap.Any("status_resp", statusResp))
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
		case dtos.TikTokVideoPostStatusPublishComplete:
			completedCount++
		case dtos.TikTokVideoPostStatusFailed:
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

// Restart implements CronJob.
func (j *TikTokStatusPollerJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting TikTok Status Poller Job due to config change")

	// Update config
	j.enabled = adminConfig.TikTokStatusPollerEnabled
	j.cronExpr = adminConfig.TikTokStatusPollerCronExpr

	// Remove existing job if it exists
	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		j.entryID = 0
	}

	// Re-initialize
	return j.Initialize()
}

// updateContentChannelStatus updates the ContentChannel based on TikTok status response
func (j *TikTokStatusPollerJob) updateContentChannelStatus(
	ctx context.Context,
	cc *model.ContentChannel,
	statusResp *dtos.TikTokPostStatusResponse,
) error {
	status := statusResp.Data.Status
	postID := statusResp.Data.PubliclyAvailablePostID
	failReason := statusResp.Data.FailReason

	zap.L().Info("Updating TikTok content channel status",
		zap.String("content_channel_id", cc.ID.String()),
		zap.String("status", string(status)),
		zap.Any("post_id", postID),
		zap.String("fail_reason", utils.DerefPtr(failReason, "N/A")))

	// Begin transaction for atomic update
	uow := j.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			zap.L().Error("Panic recovered in updateContentChannelStatus", zap.Any("recover", r))
		}
	}()

	ccRepo := uow.ContentChannels()

	// Get current metadata to preserve upload_id
	metadata, err := cc.GetMetadata()
	if err != nil {
		metadata = &model.ContentChannelMetadata{}
	}

	switch status {
	case dtos.TikTokVideoPostStatusPublishComplete:
		// Update to POSTED with final post URL
		cc.AutoPostStatus = enum.AutoPostStatusPosted
		if len(postID) > 0 {
			// Update ExternalPostID with the final TikTok post ID
			cc.ExternalPostID = utils.PtrOrNil(fmt.Sprintf("%d", postID[len(postID)-1]))
			cc.ExternalPostURL = utils.PtrOrNil(fmt.Sprintf("https://www.tiktok.com/@%s/video/%d", cc.Channel.Name, postID[len(postID)-1]))
		}
		cc.LastError = nil
		cc.PublishedAt = utils.PtrOrNil(time.Now())

		// Update metadata status to "completed" (keep upload_id for historical reference)
		metadata.UploadStatus = utils.PtrOrNil(UploadStatusCompleted)
		if err := cc.SetMetadata(metadata); err != nil {
			zap.L().Warn("Failed to update metadata on completion", zap.Error(err))
		}

		zap.L().Info("TikTok post completed",
			zap.String("content_channel_id", cc.ID.String()),
			zap.Any("post_id", postID))

	case dtos.TikTokVideoPostStatusFailed:
		// Update to FAILED with error message
		cc.AutoPostStatus = enum.AutoPostStatusFailed
		if failReason != nil {
			cc.LastError = failReason
		} else {
			cc.LastError = utils.PtrOrNil(DefaultPublishError)
		}

		// Update metadata status to "failed" (keep upload_id for debugging)
		metadata.UploadStatus = utils.PtrOrNil(UploadStatusFailed)
		if err := cc.SetMetadata(metadata); err != nil {
			zap.L().Warn("Failed to update metadata on failure", zap.Error(err))
		}

		zap.L().Warn("TikTok post failed",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("fail_reason", utils.DerefPtr(failReason, "unknown")))

	// case "PROCESSING_UPLOAD", "PROCESSING_DOWNLOAD", "PROCESSING_PUBLISH":
	case dtos.TikTokVideoPostStatusProcessingUpload,
		dtos.TikTokVideoPostStatusProcessingDownload,
		dtos.TikTokVideoPostStatusSendToUserInbox:
		// Still processing - no update needed
		zap.L().Debug("TikTok post still processing",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("status", string(status)))
		return nil // No database update needed

	default:
		zap.L().Warn("Unknown TikTok post status",
			zap.String("content_channel_id", cc.ID.String()),
			zap.String("status", string(status)))
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
