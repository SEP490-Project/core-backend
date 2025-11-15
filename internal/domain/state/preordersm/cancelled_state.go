package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// CancelledState represents CANCELLED pre-order status
type CancelledState struct{}

func (s *CancelledState) Name() enum.PreOrderStatus { return enum.PreOrderStatusCancelled }
func (s *CancelledState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{}
}
func (s *CancelledState) Next(ctx *PreOrderContext, next PreOrderState) error {
	return fmt.Errorf("no transitions allowed from CANCELLED")
}
