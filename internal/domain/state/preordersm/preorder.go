package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// PreOrderedState represents PRE_ORDERED pre-order status
type PreOrderedState struct{}

func (s *PreOrderedState) Name() enum.PreOrderStatus { return enum.PreOrderStatusPreOrdered }
func (s *PreOrderedState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusAwaitingPickup: true,
		enum.PreOrderStatusInTransit:      true,
		enum.PreOrderStatusCancelled:      true,
	}
}
func (s *PreOrderedState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from PRE_ORDERED to %s", next.Name())
}
