package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// AwaitingPickupState represents AWAITING_PICKUP pre-order status
type AwaitingPickupState struct{}

func (s *AwaitingPickupState) Name() enum.PreOrderStatus { return enum.PreOrderStatusAwaitingPickup }
func (s *AwaitingPickupState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusReceived: true,
		//enum.PreOrderStatusCancelled: true,
	}
}
func (s *AwaitingPickupState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition from AWAITING_PICKUP to %s", next.Name())
}
