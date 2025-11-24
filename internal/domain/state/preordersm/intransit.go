package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// InTransitState represents IN_TRANSIT pre-order status
type InTransitState struct{}

func (s *InTransitState) Name() enum.PreOrderStatus { return enum.PreOrderStatusInTransit }
func (s *InTransitState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusDelivered: true,
	}
}
func (s *InTransitState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from IN_TRANSIT to %s", next.Name())
}
