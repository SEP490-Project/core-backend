package milestonesm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/tasksm"
)

type MilestoneContext struct {
	State MilestoneState
	Tasks []*model.Task
}

type MilestoneState interface {
	Name() enum.MilestoneStatus
	Next(ctx *MilestoneContext, next MilestoneState) error
	AllowedTransitions() map[enum.MilestoneStatus]struct{}
}

func NewMilestoneState(status enum.MilestoneStatus) MilestoneState {
	switch status {
	case enum.MilestoneStatusNotStarted:
		return &NotStartedState{}
	case enum.MilestoneStatusOnGoing:
		return &OngoingState{}
	case enum.MilestoneStatusCompleted:
		return &CompletedState{}
	case enum.MilestoneStatusCancelled:
		return &CancelledState{}
	default:
		return nil
	}
}

// helper
func (c *MilestoneContext) IsAllTasksFinished() bool {
	if c.Tasks == nil {
		return false
	}

	for _, t := range c.Tasks {
		if t.Status != enum.TaskStatusDone && t.Status != enum.TaskStatusCancelled {
			return false
		}
	}
	return true
}

func (c *MilestoneContext) IsCancelAndCascade(state MilestoneState) {
	if state.Name() != enum.MilestoneStatusCancelled {
		return
	}

	//Cascade all tasks
	if c.Tasks != nil && len(c.Tasks) > 0 {
		for _, t := range c.Tasks {
			if t.Status == enum.TaskStatusCancelled {
				continue
			}

			taskCtx := tasksm.TaskContext{
				State:    tasksm.NewTaskState(t.Status),
				Products: t.Products,
				Contents: t.Contents,
			}
			// Cascade all product & content to cancelled
			taskCtx.State.Next(&taskCtx, &tasksm.CancelledState{})
			t.Status = taskCtx.State.Name()
		}
	}
}
