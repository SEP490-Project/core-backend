package milestonesm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type OngoingState struct{}

func (o OngoingState) Name() enum.MilestoneStatus {
	return enum.MilestoneStatusOnGoing
}

func (o OngoingState) Next(ctx *MilestoneContext, next MilestoneState) error {
	// Check tasks
	isTaskFinished := ctx.IsAllTasksFinished()

	if !isTaskFinished {
		return fmt.Errorf("invalid transition: cannot move to the next state when there are unfinished tasks")
	}

	if _, ok := o.AllowedTransitions()[next.Name()]; ok {
		if next.Name() == enum.MilestoneStatusCompleted && !isTaskFinished {
			return fmt.Errorf("invalid transition: cannot complete milestone when there are unfinished tasks")
		}
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", o.Name(), next.Name())
}

func (o OngoingState) AllowedTransitions() map[enum.MilestoneStatus]struct{} {
	return map[enum.MilestoneStatus]struct{}{
		enum.MilestoneStatusCompleted: {},
		enum.MilestoneStatusCancelled: {},
	}
}
