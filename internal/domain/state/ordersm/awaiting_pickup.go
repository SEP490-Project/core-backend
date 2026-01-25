package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type AwaitingPickUpState struct{}

func (i AwaitingPickUpState) Name() enum.OrderStatus {
	return enum.OrderStatusAwaitingPickUp
}

func (i AwaitingPickUpState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i AwaitingPickUpState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusReceived: {},
	}
}
