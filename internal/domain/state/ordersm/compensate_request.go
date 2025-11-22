package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompensateRequestStated struct{}

func (i CompensateRequestStated) Name() enum.OrderStatus {
	return enum.OrderStatusCompensateRequested
}

func (i CompensateRequestStated) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i CompensateRequestStated) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusCompensated: {},
		enum.OrderStatusDelivered:   {},
	}
}
