package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// DeliveredState represents DELIVERED pre-order status
type DeliveredState struct{}

func (s *DeliveredState) Name() enum.PreOrderStatus { return enum.PreOrderStatusDelivered }
func (s *DeliveredState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusReceived:          true,
		enum.PreOrderStatusCompensateRequest: true,
	}
}
func (s *DeliveredState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition from DELIVERED to %s", next.Name())
}
