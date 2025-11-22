package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type InTransitState struct{}

func (i InTransitState) Name() enum.OrderStatus {
	return enum.OrderStatusInTransit
}

func (i InTransitState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i InTransitState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusDelivered: {},
	}
}
