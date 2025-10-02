package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type RecapState struct{}

func (r *RecapState) Name() enum.TaskStatus { return enum.TaskStatusRecap }

func (r *RecapState) Next(ctx *TaskContext, next TaskState) error {
	if _, ok := r.AllowedTransitions()[next.Name()]; ok {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", r.Name(), next.Name())
}

func (r *RecapState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{
		enum.TaskStatusDone: {},
	}
}
