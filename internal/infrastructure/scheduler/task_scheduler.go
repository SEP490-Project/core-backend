// Package scheduler provides interfaces and implementations for task scheduling.
package scheduler

import "context"

type TaskScheduler interface {
	Start(ctx context.Context)
	StartOnce(ctx context.Context) error
}
