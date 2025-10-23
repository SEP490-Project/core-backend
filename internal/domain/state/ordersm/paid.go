package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type PaidState struct{}

func (p PaidState) Name() enum.OrderStatus {
	return enum.OrderStatusPaid
}

func (p PaidState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", p.Name(), next.Name())
}

func (p PaidState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusConfirmed: {},
	}
}
