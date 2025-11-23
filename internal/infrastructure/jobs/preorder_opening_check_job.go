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

type preOrderOpeningCheckJob struct {
	preOrderService iservice.PreOrderService
	cronScheduler   *cron.Cron
	intervalMinutes int
	enabled         bool
	adminConfig     *config.AdminConfig
	lastRunTime     time.Time
}

func NewPreOrderOpeningCheckJob(
	preOrderService iservice.PreOrderService,
	cronScheduler *cron.Cron,
	adminConfig *config.AdminConfig,
) CronJob {
	intervalMinutes := adminConfig.PreOrderOpeningCheckIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 30 // Default to 30 minutes if not set
	}

	return &preOrderOpeningCheckJob{
		preOrderService: preOrderService,
		cronScheduler:   cronScheduler,
		intervalMinutes: intervalMinutes,
		enabled:         adminConfig.PreOrderOpeningCheckEnable,
		adminConfig:     adminConfig,
	}
}

func (p preOrderOpeningCheckJob) Initialize() error {
	if !p.adminConfig.PreOrderOpeningCheckEnable {
		zap.L().Info("Pre-Order Opening Check Job is disabled via admin config")
		return nil
	}
	zap.L().Debug("Initializing Pre-Order Opening Check Job...")
	// Generate cron expression (e.g., "*/30 * * * *" for every 30 minutes)
	cronExpr := fmt.Sprintf("*/%d * * * *", p.intervalMinutes)
	zap.L().Info("Scheduling Pre-Order Opening Check Job",
		zap.String("cron_expression", cronExpr),
		zap.Int("interval_minutes", p.intervalMinutes))

	// Schedule the job
	_, err := p.cronScheduler.AddFunc(cronExpr, func() {
		if p.enabled {
			p.Run()
		}
	})

	if err != nil {
		zap.L().Error("Failed to schedule Pre-Order Opening Check Job", zap.Error(err))
		return fmt.Errorf("failed to schedule Pre-Order Opening Check Job: %w", err)
	}

	return err
}

func (p preOrderOpeningCheckJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.intervalMinutes)*time.Minute)
	defer cancel()

	startTime := time.Now()
	p.lastRunTime = startTime
	zap.L().Info("Starting Pre-Order Opening Check Job")
	_ = ctx
	found, failed, upcomming := p.preOrderService.PreOrderOpeningChecker(ctx)
	zap.L().Info("Pre-Order Opening Check Job completed",
		zap.Int("total_pre_orders_processed", found),
		zap.Int("total_pre_orders_failed", failed),
		zap.Int("total_pre_orders_upcoming", upcomming),
		zap.Duration("duration", time.Since(startTime)),
	)
}

func (p preOrderOpeningCheckJob) IsEnabled() bool {
	return p.enabled
}

func (p preOrderOpeningCheckJob) SetEnabled(enabled bool) {
	p.enabled = enabled
}

func (p preOrderOpeningCheckJob) GetLastRunTime() time.Time {
	return p.lastRunTime
}
