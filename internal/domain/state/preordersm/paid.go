package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// PaidState represents PAID pre-order status
type PaidState struct{}

func (s *PaidState) Name() enum.PreOrderStatus { return enum.PreOrderStatusPaid }
func (s *PaidState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusPreOrdered: true,
		//enum.PreOrderStatusCancelled:  true,
	}
}
func (s *PaidState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition from PAID to %s", next.Name())
}
