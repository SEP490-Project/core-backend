package campaignsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type OnGoingState struct{}

func (o OnGoingState) Name() enum.CampaignStatus { return enum.CampaignRunning }

func (o OnGoingState) Next(ctx *CampaignContext, next CampaignState) error {
	if _, ok := o.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsCancelAndCascade(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", o.Name(), next.Name())
}

func (o OnGoingState) AllowedTransitions() map[enum.CampaignStatus]struct{} {
	return map[enum.CampaignStatus]struct{}{
		enum.CampaignCompleted: {},
		enum.CampaignCancelled: {},
	}
}
