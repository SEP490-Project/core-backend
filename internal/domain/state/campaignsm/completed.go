package campaignsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompletedState struct{}

func (c CompletedState) Name() enum.CampaignStatus {
	return enum.CampaignCompleted
}

func (c CompletedState) Next(ctx *CampaignContext, next CampaignState) error {
	if _, ok := c.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsCancelAndCascade(next)
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", c.Name(), next.Name())
}

func (c CompletedState) AllowedTransitions() map[enum.CampaignStatus]struct{} {
	return map[enum.CampaignStatus]struct{}{
		enum.CampaignCanceled: {},
	}
}
