package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// ShippedState represents SHIPPED pre-order status
type ShippedState struct{}

func (s *ShippedState) Name() enum.PreOrderStatus { return enum.PreOrderStatusShipped }
func (s *ShippedState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusInTransit: true,
	}
}
func (s *ShippedState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition from SHIPPED to %s", next.Name())
}
