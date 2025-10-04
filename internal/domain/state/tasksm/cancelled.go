package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CancelledState struct{}

func (c *CancelledState) Name() enum.TaskStatus { return enum.TaskStatusCancelled }

func (c *CancelledState) Next(ctx *TaskContext, next TaskState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c *CancelledState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{}
}
