package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CancelledState struct{}

func (c CancelledState) Name() enum.OrderStatus {
	return enum.OrderStatusCancelled
}

func (c CancelledState) Next(ctx *OrderContext, next OrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c CancelledState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{}
}
