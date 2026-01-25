// Package contractsm implement the finite state machine pattern specifically for managing contract states and their transitions.
package contractsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/campaignsm"
	"fmt"
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
	case enum.ContractStatusApproved:
		return &ApprovedState{}
	case enum.ContractStatusActive:
		return &ActiveState{}
	case enum.ContractStatusCompleted:
		return &CompletedState{}
	case enum.ContractStatusTerminated:
		return &TerminatedState{}
	// Brand violation states
	case enum.ContractStatusBrandViolated:
		return &BrandViolatedState{}
	case enum.ContractStatusBrandPenaltyPending:
		return &BrandPenaltyPendingState{}
	case enum.ContractStatusBrandPenaltyPaid:
		return &BrandPenaltyPaidState{}
	// KOL violation states
	case enum.ContractStatusKOLViolated:
		return &KOLViolatedState{}
	case enum.ContractStatusKOLRefundPending:
		return &KOLRefundPendingState{}
	case enum.ContractStatusKOLProofSubmitted:
		return &KOLProofSubmittedState{}
	case enum.ContractStatusKOLProofRejected:
		return &KOLProofRejectedState{}
	case enum.ContractStatusKOLRefundApproved:
		return &KOLRefundApprovedState{}
	default:
		return nil
	}
}

// region: ================ Helper Methods ================

// transition is a helper function to manage state transitions and apply side effects if needed
func transition(ctx *ContractContext, current, next ContractState, effects func(next ContractState) bool) error {
	if _, ok := current.AllowedTransitions()[next.Name()]; ok {
		if effects == nil || effects(next) {
			ctx.State = next
			return nil
		} else { // effects returned false, meaning transition should not occur
			return fmt.Errorf("transition effects prevented state change: %s -> %s", current.Name(), next.Name())
		}
	}

	return fmt.Errorf("invalid transition: %s -> %s", current.Name(), next.Name())
}

// IsCampaignCompleted checks if the associated campaign is completed
func (c *ContractContext) IsCampaignCompleted() bool {
	if c.Campaign == nil {
		return false
	}

	return c.Campaign.Status == enum.CampaignCompleted
}

// IsTerminatedAndCascade checks if the contract is terminated and cascades the termination to the associated campaign
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

// endregion
