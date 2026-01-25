package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type PendingState struct{}

func (p PendingState) Name() enum.OrderStatus {
	return enum.OrderStatusPending
}

func (p PendingState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", p.Name(), next.Name())
}

func (p PendingState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusPaid:      {},
		enum.OrderStatusCancelled: {},
	}
}
