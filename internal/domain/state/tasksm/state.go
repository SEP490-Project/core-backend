package tasksm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
)

type TaskContext struct {
	State    TaskState
	Products []*model.Product
	Contents []*model.Content
}

type TaskState interface {
	Name() enum.TaskStatus
	Next(ctx *TaskContext, next TaskState) error
	AllowedTransitions() map[enum.TaskStatus]struct{}
}

func NewTaskState(status enum.TaskStatus) TaskState {
	switch status {
	case enum.TaskStatusToDo:
		return &ToDoState{}
	case enum.TaskStatusInProgress:
		return &InProgressState{}
	case enum.TaskStatusRecap:
		return &RecapState{}
	case enum.TaskStatusDone:
		return &DoneState{}
	case enum.TaskStatusCancelled:
		return &CancelledState{}
	default:
		return nil
	}
}

func isAllowed(current TaskState, targetState enum.TaskStatus) bool {
	_, ok := current.AllowedTransitions()[targetState]
	return ok
}

func PrintAllowedTransitions(state TaskState) []string {
	transitions := make([]string, 0, len(state.AllowedTransitions()))
	for k := range state.AllowedTransitions() {
		transitions = append(transitions, k.String())
	}
	return transitions
}

// helper
func (c *TaskContext) IsAllProductsActive() bool {
	if c.Products == nil || len(c.Products) == 0 {
		return false
	}

	for _, p := range c.Products {
		if p.Status != enum.ProductStatusActived {
			return false
		}
	}

	return true
}

func (c *TaskContext) IsAllContentsPosted() bool {
	if c.Contents == nil || len(c.Contents) == 0 {
		return false
	}

	for _, ct := range c.Contents {
		if ct.ContentStatus != enum.ContentStatusPosted {
			return false
		}
	}

	return true
}
