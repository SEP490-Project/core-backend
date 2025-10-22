package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ConfirmedState struct{}

func (c ConfirmedState) Name() enum.OrderStatus {
	return enum.OrderStatusConfirmed
}

func (c ConfirmedState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := c.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", c.Name(), next.Name())
}

func (c ConfirmedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusShipped:   {},
		enum.OrderStatusCancelled: {},
	}
}
