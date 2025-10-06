package contractsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/campaignsm"
)

type ContractContext struct {
	State    ContractState
	Campaign *model.Campaign
}

type ContractState interface {
	Name() enum.ContractStatus
	Next(ctx *ContractContext, next ContractState) error
	AllowedTransitions() map[enum.ContractStatus]struct{}
}

func NewContractState(state enum.ContractStatus) ContractState {
	switch state {
	case enum.ContractStatusDraft:
		return &DraftState{}
	case enum.ContractStatusActive:
		return &ActiveState{}
	case enum.ContractStatusCompleted:
		return &CompletedState{}
	case enum.ContractStatusTerminated:
		return &TerminatedState{}
	default:
		return nil
	}
}

// helper
func (c *ContractContext) IsCampaignCompleted() bool {
	if c.Campaign == nil {
		return false
	}

	return c.Campaign.Status == enum.CampaignCompleted
}

func (c *ContractContext) IsTerminatedAndCascade(state ContractState) {
	if state.Name() != enum.ContractStatusTerminated {
		return
	}

	// Cancel campaign -> cascade all
	if c.Campaign != nil {
		campaignCtx := campaignsm.CampaignContext{
			State:      campaignsm.NewCampaignState(c.Campaign.Status),
			MileStones: c.Campaign.Milestones,
		}
		campaignCtx.State.Next(&campaignCtx, &campaignsm.CancelledState{})
		c.Campaign.Status = campaignCtx.State.Name()
	}
}
