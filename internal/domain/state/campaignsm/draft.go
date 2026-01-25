package campaignsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type DraftState struct{}

func (s DraftState) Name() enum.CampaignStatus { return enum.CampaignDraft }

func (s DraftState) Next(ctx *CampaignContext, next CampaignState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsCancelAndCascade(next)
		if next.Name() == enum.CampaignRunning {
			ctx.Campaign.RejectReason = nil
		}
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s DraftState) AllowedTransitions() map[enum.CampaignStatus]struct{} {
	return map[enum.CampaignStatus]struct{}{
		enum.CampaignRunning:   {},
		enum.CampaignCancelled: {},
	}
}
