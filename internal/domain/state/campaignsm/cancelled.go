package campaignsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CancelledState struct{}

func (c CancelledState) Name() enum.CampaignStatus {
	return enum.CampaignCanceled
}

func (c CancelledState) Next(ctx *CampaignContext, next CampaignState) error {
	if _, ok := c.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsCancelAndCascade(next)
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", c.Name(), next.Name())
}

func (c CancelledState) AllowedTransitions() map[enum.CampaignStatus]struct{} {
	return map[enum.CampaignStatus]struct{}{
		enum.CampaignCanceled: {},
	}
}
