package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ExpiredLinkCleanupJob struct {
	affiliateLinkRepo irepository.AffiliateLinkRepository
	db                *gorm.DB
	CronScheduler     *cron.Cron
	lastRunTime       time.Time
	adminConfig       *config.AdminConfig
	entryID           cron.EntryID
}

func NewExpiredLinkCleanupJob(
	affiliateLinkRepo irepository.AffiliateLinkRepository,
	db *gorm.DB,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	return &ExpiredLinkCleanupJob{
		affiliateLinkRepo: affiliateLinkRepo,
		db:                db,
		CronScheduler:     cronScheduler,
		adminConfig:       adminConfig,
	}
}

// Initialize implements CronJob.
func (j *ExpiredLinkCleanupJob) Initialize() error {
	if !j.adminConfig.ExpiredContractCleanupEnabled {
		zap.L().Info("Expired Link Cleanup Job is disabled via admin config")
		return nil
	}

	zap.L().Debug("Initializing Expired Link Cleanup Job...")

	cronExpr := j.adminConfig.ExpiredContractCleanupCronExpr
	if cronExpr == "" {
		cronExpr = "0 0 0 * * *" // Default to daily at midnight if not set
	}
	zap.L().Info("Scheduling Expired Link Cleanup Job",
		zap.String("cron_expression", cronExpr),
		zap.String("schedule", "Daily at midnight"))

	// Schedule the job
	entryID, err := j.CronScheduler.AddFunc(cronExpr, func() {
		if j.adminConfig.ExpiredContractCleanupEnabled {
			j.Run()
		}
	})
	j.entryID = entryID

	if err != nil {
		zap.L().Error("Failed to schedule Expired Link Cleanup Job", zap.Error(err))
		return fmt.Errorf("failed to schedule expired link cleanup job: %w", err)
	}

	zap.L().Info("Expired Link Cleanup Job scheduled successfully")

	return nil

}

// Run implements CronJob.
func (j *ExpiredLinkCleanupJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	zap.L().Info("Starting expired link cleanup job")
	startTime := time.Now()

	// Mark links as EXPIRED where:
	// 1. Link status is ACTIVE but contract is not ACTIVE
	// 2. Link status is ACTIVE but content is not POSTED

	// Query 1: Links with inactive contracts
	query1 := `
		UPDATE affiliate_links al
		SET status = ?, updated_at = NOW()
		FROM contracts c
		WHERE al.contract_id = c.id
		  AND al.status = ?
		  AND c.status != ?
		  AND al.deleted_at IS NULL
		  AND c.deleted_at IS NULL
	`

	result1 := j.db.WithContext(ctx).Exec(query1,
		enum.AffiliateLinkStatusExpired,
		enum.AffiliateLinkStatusActive,
		enum.ContractStatusActive,
	)

	if result1.Error != nil {
		zap.L().Error("Failed to mark links with inactive contracts as expired",
			zap.Error(result1.Error))
	} else {
		zap.L().Info("Marked links with inactive contracts as expired",
			zap.Int64("count", result1.RowsAffected))
	}

	// Query 2: Links with unpublished content
	query2 := `
		UPDATE affiliate_links al
		SET status = ?, updated_at = NOW()
		FROM contents c
		WHERE al.content_id = c.id
		  AND al.status = ?
		  AND c.status != ?
		  AND al.deleted_at IS NULL
		  AND c.deleted_at IS NULL
	`

	result2 := j.db.WithContext(ctx).Exec(query2,
		enum.AffiliateLinkStatusExpired,
		enum.AffiliateLinkStatusActive,
		enum.ContentStatusPosted,
	)

	if result2.Error != nil {
		zap.L().Error("Failed to mark links with unpublished content as expired",
			zap.Error(result2.Error))
	} else {
		zap.L().Info("Marked links with unpublished content as expired",
			zap.Int64("count", result2.RowsAffected))
	}

	// Invalidate cache for expired links
	// TODO: Implement cache invalidation for expired links
	// This could be done by iterating through expired links and calling cache.Delete(hash)

	duration := time.Since(startTime)
	totalExpired := result1.RowsAffected + result2.RowsAffected

	zap.L().Info("Expired link cleanup job completed",
		zap.Int64("total_expired", totalExpired),
		zap.Duration("duration", duration))

}

// GetLastRunTime implements CronJob.
func (j *ExpiredLinkCleanupJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// IsEnabled implements CronJob.
func (j *ExpiredLinkCleanupJob) IsEnabled() bool {
	return j.adminConfig.ExpiredContractCleanupEnabled
}

// SetEnabled implements CronJob.
func (j *ExpiredLinkCleanupJob) SetEnabled(enabled bool) {
	j.adminConfig.ExpiredContractCleanupEnabled = enabled
}

// Restart implements CronJob.
func (j *ExpiredLinkCleanupJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting Expired Link Cleanup Job due to config change")

	// Update config
	j.adminConfig = adminConfig

	// Remove existing job if it exists
	if j.entryID != 0 {
		j.CronScheduler.Remove(j.entryID)
		j.entryID = 0
	}

	// Re-initialize
	return j.Initialize()
}
