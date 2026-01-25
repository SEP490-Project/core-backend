package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ShippedState struct{}

func (s ShippedState) Name() enum.OrderStatus {
	return enum.OrderStatusShipped
}

func (s ShippedState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s ShippedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusInTransit: {},
	}
}
