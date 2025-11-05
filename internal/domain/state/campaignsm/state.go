// Package campaignsm implements the state machine for Campaign entity
package campaignsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	milestonesm "core-backend/internal/domain/state/milestonesm"
)

type CampaignContext struct {
	State      CampaignState
	Campaign   *model.Campaign
	MileStones []*model.Milestone
}

type CampaignState interface {
	Name() enum.CampaignStatus
	Next(ctx *CampaignContext, next CampaignState) error
	AllowedTransitions() map[enum.CampaignStatus]struct{}
}

func NewCampaignState(status enum.CampaignStatus) CampaignState {
	switch status {
	case enum.CampaignRunning:
		return &OnGoingState{}
	case enum.CampaignCompleted:
		return &CompletedState{}
	case enum.CampaignCancelled:
		return &CancelledState{}
	default:
		return nil
	}
}

// region: ======= Helper Methods =======

func (c *CampaignContext) IsAllMilestonesFinished() bool {
	if c.MileStones == nil {
		return false
	}
	for _, m := range c.MileStones {
		if m.Status != enum.MilestoneStatusCompleted && m.Status != enum.MilestoneStatusCancelled {
			return false
		}
	}
	return true
}

// IsCancelAndCascade cascades cancellation down to milestones -> tasks -> products/contents (in-memory only)
func (c *CampaignContext) IsCancelAndCascade(state CampaignState) {
	if state.Name() != enum.CampaignCancelled {
		return
	}
	if len(c.MileStones) == 0 {
		return
	}
	for _, ms := range c.MileStones {
		if ms == nil || ms.Status == enum.MilestoneStatusCancelled {
			continue
		}
		mCtx := milestonesm.MilestoneContext{State: milestonesm.NewMilestoneState(ms.Status)}
		// Attach tasks (and their nested relations) if present so milestone state machine can cascade
		if ms.Tasks != nil {
			mCtx.Tasks = ms.Tasks
		}
		_ = mCtx.State.Next(&mCtx, &milestonesm.CancelledState{}) // ignore error; forced cancellation
		ms.Status = enum.MilestoneStatusCancelled

	}
}

// endregion
