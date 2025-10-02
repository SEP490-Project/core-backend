package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type DoneState struct{}

func (d *DoneState) Name() enum.TaskStatus { return enum.TaskStatusDone }

func (d *DoneState) Next(ctx *TaskContext, next TaskState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (d *DoneState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{}
}
