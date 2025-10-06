package milestonesm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompletedState struct{}

func (n CompletedState) Name() enum.MilestoneStatus {
	return enum.MilestoneStatusCompleted
}

func (n CompletedState) Next(ctx *MilestoneContext, next MilestoneState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (n CompletedState) AllowedTransitions() map[enum.MilestoneStatus]struct{} {
	return map[enum.MilestoneStatus]struct{}{
		enum.MilestoneStatusCancelled: {},
	}
}
