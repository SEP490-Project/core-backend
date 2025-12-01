package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// RefundRequest represents PAID pre-order status
type RefundRequest struct{}

func (s *RefundRequest) Name() enum.PreOrderStatus { return enum.PreOrderStatusRefundRequest }
func (s *RefundRequest) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusRefunded: true,
	}
}
func (s *RefundRequest) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition from Refund Request to %s", next.Name())
}
