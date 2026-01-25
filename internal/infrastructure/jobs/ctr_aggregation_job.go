package jobs

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type CTRAggregationJob struct {
	clickEventRepo    irepository.ClickEventRepository
	kpiMetricsRepo    irepository.KPIMetricsRepository
	affiliateLinkRepo irepository.AffiliateLinkRepository
	cronScheduler     *cron.Cron
	lastRunTime       time.Time
	intervalMinutes   int
	enabled           bool
	entryID           cron.EntryID
}

func NewCTRAggregationJob(
	clickEventRepo irepository.ClickEventRepository,
	kpiMetricsRepo irepository.KPIMetricsRepository,
	affiliateLinkRepo irepository.AffiliateLinkRepository,
	crontScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	intervalMinutes := adminConfig.CTRAggregationIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 5 // Default to 5 minutes if not set
	}

	return &CTRAggregationJob{
		clickEventRepo:    clickEventRepo,
		kpiMetricsRepo:    kpiMetricsRepo,
		affiliateLinkRepo: affiliateLinkRepo,
		cronScheduler:     crontScheduler,
		lastRunTime:       time.Now().Add(-time.Duration(intervalMinutes) * time.Minute), // Initialize to interval minutes ago
		intervalMinutes:   intervalMinutes,
		enabled:           adminConfig.CTRAggregationEnabled,
	}
}

// Initialize implements CronJob.
func (j *CTRAggregationJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("CTR Aggregation Job is disabled via admin config")
		return nil
	}
	zap.L().Debug("Initializing CTR Aggregation Job...")

	// Generate cron expression (e.g., "*/5 * * * *" for every 5 minutes)
	cronExpr := fmt.Sprintf("0 */%d * * * *", j.intervalMinutes)
	zap.L().Info("Scheduling CTR Aggregation Job",
		zap.String("cron_expression", cronExpr),
		zap.Int("interval_minutes", j.intervalMinutes))

	// Schedule the job
	entryID, err := j.cronScheduler.AddFunc(cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})

	if err != nil {
		zap.L().Error("Failed to schedule CTR Aggregation Job", zap.Error(err))
		return fmt.Errorf("failed to schedule CTR aggregation job: %w", err)
	}
	j.entryID = entryID

	return nil
}

// Restart implements CronJob.
func (j *CTRAggregationJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting CTR Aggregation Job...")

	// Remove existing job if it exists
	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		j.entryID = 0
	}

	// Update configuration
	j.enabled = adminConfig.CTRAggregationEnabled
	j.intervalMinutes = adminConfig.CTRAggregationIntervalMinutes
	if j.intervalMinutes <= 0 {
		j.intervalMinutes = 5
	}

	// Re-initialize
	return j.Initialize()
}

// Run implements CronJob.
func (j *CTRAggregationJob) Run() {
	ctx := context.Background()
	startTime := time.Now()

	zap.L().Info("Starting CTR aggregation job",
		zap.Time("last_run", j.lastRunTime),
		zap.Duration("interval", time.Duration(j.intervalMinutes)*time.Minute))

	// Aggregate clicks by affiliate link directly in DB
	aggregates, err := j.clickEventRepo.GetAggregatedClicks(ctx, j.lastRunTime)
	if err != nil {
		zap.L().Error("Failed to retrieve aggregated clicks", zap.Error(err))
		return
	}

	if len(aggregates) == 0 {
		zap.L().Info("No new click events to aggregate")
		j.lastRunTime = time.Now()
		return
	}

	zap.L().Info("Processing aggregated clicks", zap.Int("unique_links", len(aggregates)))

	// Store aggregated metrics in kpi_metrics table
	successCount := 0
	errorCount := 0
	totalClicks := 0

	for linkID, clickCount := range aggregates {
		totalClicks += clickCount
		metric := &model.KPIMetrics{
			ReferenceID:   linkID,
			ReferenceType: enum.KPIReferenceTypeAffiliateLink,
			Type:          enum.KPIValueTypeClickThrough,
			Value:         float64(clickCount),
			RecordedDate:  time.Now(),
		}

		if err := j.kpiMetricsRepo.Add(ctx, metric); err != nil {
			zap.L().Error("Failed to store KPI metric",
				zap.String("affiliate_link_id", linkID.String()),
				zap.Int("click_count", clickCount),
				zap.Error(err))
			errorCount++
		} else {
			successCount++
		}
	}

	// Update last run time
	j.lastRunTime = time.Now()

	duration := time.Since(startTime)
	zap.L().Info("CTR aggregation job completed",
		zap.Int("total_clicks", totalClicks),
		zap.Int("unique_links", len(aggregates)),
		zap.Int("success_count", successCount),
		zap.Int("error_count", errorCount),
		zap.Duration("duration", duration))

}

// GetLastRunTime returns the timestamp of the last successful run
func (j *CTRAggregationJob) GetLastRunTime() time.Time {
	return j.lastRunTime
}

// GetIntervalMinutes returns the configured interval in minutes
func (j *CTRAggregationJob) GetIntervalMinutes() int {
	return j.intervalMinutes
}

// SetEnabled enables or disables the job
func (j *CTRAggregationJob) SetEnabled(enabled bool) {
	j.enabled = enabled
	if enabled {
		zap.L().Info("CTR aggregation job enabled")
	} else {
		zap.L().Info("CTR aggregation job disabled")
	}
}

// IsEnabled returns whether the job is enabled
func (j *CTRAggregationJob) IsEnabled() bool {
	return j.enabled
}
