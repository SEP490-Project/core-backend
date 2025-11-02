// Package scheduler contains the task schedulers using cron jobs used in the application.
package scheduler

import "context"

type TaskScheduler interface {
	Start(ctx context.Context)
	StartOnce(ctx context.Context) error
}
