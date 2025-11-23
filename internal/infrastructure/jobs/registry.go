package jobs

import (
	"context"
	"core-backend/config"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CronJobRegistry struct {
	CTRAggregationJob       CronJob
	ExpiredLinkCleanupJob   CronJob
	PayOSExpiryCheckJob     CronJob
	PreOrderOpeningCheckJob CronJob
	TikTokStatusPollerJob   CronJob // Added for application layer
	CronScheduler           *cron.Cron
	jobs                    map[string]CronJob
}

func NewCronJobRegistry(dbReg *gormrepository.DatabaseRegistry, db *gorm.DB, adminConfig *config.AdminConfig) *CronJobRegistry {
	registry := &CronJobRegistry{
		CronScheduler: cron.New(),
		jobs:          make(map[string]CronJob),
	}

	registry.CTRAggregationJob = NewCTRAggregationJob(
		dbReg.ClickEventRepository,
		dbReg.KPIMetricsRepository,
		dbReg.AffiliateLinkRepository,
		registry.CronScheduler,
		adminConfig)

	registry.ExpiredLinkCleanupJob = NewExpiredLinkCleanupJob(
		dbReg.AffiliateLinkRepository,
		db,
		registry.CronScheduler,
		adminConfig)

	registry.jobs["ctr_aggregation_job"] = registry.CTRAggregationJob
	registry.jobs["expired_link_cleanup_job"] = registry.ExpiredLinkCleanupJob
	registry.jobs["pre_order_opening_check_job"] = registry.PreOrderOpeningCheckJob
	return registry
}

// RegisterApplicationLayerJob registers a job that depends on application services
func (r *CronJobRegistry) RegisterApplicationLayerJob(name string, job CronJob) {
	r.jobs[name] = job
}

func (r *CronJobRegistry) GetJobByName(name string) (CronJob, bool) {
	job, exists := r.jobs[name]
	return job, exists
}

func (r *CronJobRegistry) GetAllJobs() map[string]CronJob {
	return r.jobs
}

func (r *CronJobRegistry) InitializeAllJobs() error {
	for name, job := range r.jobs {
		if err := job.Initialize(); err != nil {
			return err
		}
		zap.L().Info("Initialized cron job", zap.String("job_name", name))
	}
	return nil
}

func (r *CronJobRegistry) StartCronScheduler() {
	zap.L().Info("Starting cron scheduler for registered jobs")
	r.CronScheduler.Start()
}

func (r *CronJobRegistry) StopCronScheduler() context.Context {
	zap.L().Info("Stopping cron scheduler for registered jobs")
	return r.CronScheduler.Stop()
}

// RegisterJob adds a new job to the registry
func (r *CronJobRegistry) RegisterJob(name string, job CronJob) {
	r.jobs[name] = job
	zap.L().Info("Registered new cron job", zap.String("job_name", name))
}
