package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// PendingState represents PENDING pre-order status
type PendingState struct{}

func (p *PendingState) Name() enum.PreOrderStatus { return enum.PreOrderStatusPending }
func (p *PendingState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusPaid:      true,
		enum.PreOrderStatusCancelled: true,
	}
}
func (p *PendingState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from PENDING to %s", next.Name())
}
