package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type DeliveredState struct{}

func (d DeliveredState) Name() enum.OrderStatus {
	return enum.OrderStatusInTransit
}

func (d DeliveredState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := d.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", d.Name(), next.Name())
}

func (d DeliveredState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusReceived: {},
	}
}
