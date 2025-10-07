package milestonesm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type NotStartedState struct{}

func (n NotStartedState) Name() enum.MilestoneStatus {
	return enum.MilestoneStatusNotStarted
}

func (n NotStartedState) Next(ctx *MilestoneContext, next MilestoneState) error {
	if _, ok := n.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", n.Name(), next.Name())
}

func (n NotStartedState) AllowedTransitions() map[enum.MilestoneStatus]struct{} {
	return map[enum.MilestoneStatus]struct{}{
		enum.MilestoneStatusOnGoing:   {},
		enum.MilestoneStatusCancelled: {},
	}
}
