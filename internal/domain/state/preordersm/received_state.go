package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// ReceivedState represents RECEIVED pre-order status
type ReceivedState struct{}

func (s *ReceivedState) Name() enum.PreOrderStatus { return enum.PreOrderStatusReceived }
func (s *ReceivedState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{}
}
func (s *ReceivedState) Next(ctx *PreOrderContext, next PreOrderState) error {
	return fmt.Errorf("no transitions allowed from RECEIVED")
}
