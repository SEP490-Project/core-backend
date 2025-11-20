package jobs

import (
	"core-backend/internal/application/interfaces/iservice"
	"github.com/robfig/cron/v3"
	"time"
)

type preOrderOpeningCheckJob struct {
	stateTransferService iservice.StateTransferService
	cronScheduler        *cron.Cron
	intervalMinutes      int
	enabled              bool
	lastRunTime          time.Time
}

func NewPreOrderOpeningCheckJob(
	stateTransferService iservice.StateTransferService,
	cronScheduler *cron.Cron,
	intervalMinutes int,
	enabled bool,
) CronJob {
	return &preOrderOpeningCheckJob{
		stateTransferService: stateTransferService,
		cronScheduler:        cronScheduler,
		intervalMinutes:      intervalMinutes,
		enabled:              enabled,
		lastRunTime:          time.Now().Add(-time.Duration(intervalMinutes) * time.Minute),
	}
}

func (p preOrderOpeningCheckJob) Initialize() error {
	//TODO implement me
	panic("implement me")
}

func (p preOrderOpeningCheckJob) Run() {
	//TODO implement me
	panic("implement me")
}

func (p preOrderOpeningCheckJob) IsEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (p preOrderOpeningCheckJob) SetEnabled(enabled bool) {
	//TODO implement me
	panic("implement me")
}

func (p preOrderOpeningCheckJob) GetLastRunTime() time.Time {
	//TODO implement me
	panic("implement me")
}
