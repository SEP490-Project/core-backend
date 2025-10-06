package milestonesm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CancelledState struct{}

func (n CancelledState) Name() enum.MilestoneStatus {
	return enum.MilestoneStatusCancelled
}

func (n CancelledState) Next(ctx *MilestoneContext, next MilestoneState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (n CancelledState) AllowedTransitions() map[enum.MilestoneStatus]struct{} {
	return map[enum.MilestoneStatus]struct{}{}
}
