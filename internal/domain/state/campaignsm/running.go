package campaignsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type RunningState struct{}

func (s RunningState) Name() enum.CampaignStatus { return enum.CampaignRunning }

func (s RunningState) Next(ctx *CampaignContext, next CampaignState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsCancelAndCascade(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s RunningState) AllowedTransitions() map[enum.CampaignStatus]struct{} {
	return map[enum.CampaignStatus]struct{}{
		enum.CampaignCompleted: {},
		enum.CampaignCancelled: {},
	}
}
