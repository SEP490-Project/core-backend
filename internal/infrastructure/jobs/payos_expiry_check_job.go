package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type PayOSExpiryCheckJob struct {
	paymentTransactionService iservice.PaymentTransactionService
	cronScheduler             *cron.Cron
	intervalMinutes           int
	enabled                   bool
	lastRunTime               time.Time
	entryID                   cron.EntryID
}

func NewPayOSExpiryCheckJob(
	paymentTransactionService iservice.PaymentTransactionService,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	intervalMinutes := adminConfig.PayOSExpiryCheckIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 30 // Default to 30 minutes if not set
	}

	return &PayOSExpiryCheckJob{
		paymentTransactionService: paymentTransactionService,
		cronScheduler:             cronScheduler,
		intervalMinutes:           intervalMinutes,
		enabled:                   adminConfig.PayOSExpiryCheckEnabled,
		lastRunTime:               time.Now().Add(-time.Duration(intervalMinutes) * time.Minute),
	}
}

// Initialize implements CronJob.
func (j *PayOSExpiryCheckJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("PayOS Expiry Check Job is disabled via admin config")
		return nil
	}
	zap.L().Debug("Initializing PayOS Expiry Check Job...")

	// Generate cron expression (e.g., "*/30 * * * *" for every 30 minutes)
	cronExpr := fmt.Sprintf("*/%d * * * *", j.intervalMinutes)
	zap.L().Info("Scheduling PayOS Expiry Check Job",
		zap.String("cron_expression", cronExpr),
		zap.Int("interval_minutes", j.intervalMinutes))

	// Schedule the job
	entryID, err := j.cronScheduler.AddFunc(cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})
	j.entryID = entryID

	if err != nil {
		zap.L().Error("Failed to schedule PayOS Expiry Check Job", zap.Error(err))
		return fmt.Errorf("failed to schedule PayOS expiry check job: %w", err)
	}

	return nil
}

// Run implements CronJob.
func (j *PayOSExpiryCheckJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	j.lastRunTime = startTime
	zap.L().Info("Starting PayOS expiry check job execution")

	// Call service to cancel expired links
	cancelledCount, err := j.paymentTransactionService.CancelExpiredLinks(ctx)
	if err != nil {
		zap.L().Error("PayOS expiry check job failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))
		return
	}

	zap.L().Info("PayOS expiry check job completed successfully",
		zap.Int("cancelled_count", cancelledCount),
		zap.Duration("duration", time.Since(startTime)))
}

// Name implements CronJob.
func (j *PayOSExpiryCheckJob) Name() string {
	return "PayOSExpiryCheckJob"
}

// IsEnabled implements CronJob.
func (j *PayOSExpiryCheckJob) IsEnabled() bool {
	return j.enabled
}

// SetEnabled implements CronJob.
func (j *PayOSExpiryCheckJob) SetEnabled(enabled bool) {
	j.enabled = enabled
}

// GetLastRunTime implements CronJob.
func (j *PayOSExpiryCheckJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// Restart implements CronJob.
func (j *PayOSExpiryCheckJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting PayOS Expiry Check Job due to config change")

	// Update config
	j.enabled = adminConfig.PayOSExpiryCheckEnabled
	intervalMinutes := adminConfig.PayOSExpiryCheckIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 30
	}
	j.intervalMinutes = intervalMinutes

	// Remove existing job if it exists
	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		j.entryID = 0
	}

	// Re-initialize
	return j.Initialize()
}
