package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type RefundRequestedState struct{}

func (r RefundRequestedState) Name() enum.OrderStatus {
	return enum.OrderStatusRefundRequested
}

func (r RefundRequestedState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := r.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", r.Name(), next.Name())
}

func (r RefundRequestedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusRefunded: {},
	}
}
