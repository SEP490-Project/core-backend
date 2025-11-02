// Package jobs contains the Cron jobs interfaces and the specific implementations.
package jobs

import "time"

type CronJob interface {
	Initialize() error
	Run()
	IsEnabled() bool
	SetEnabled(enabled bool)
	GetLastRunTime() time.Time
}
