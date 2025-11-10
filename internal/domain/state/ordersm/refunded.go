package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type RefundedState struct{}

func (r RefundedState) Name() enum.OrderStatus {
	return enum.OrderStatusCancelled
}

func (r RefundedState) Next(ctx *OrderContext, next OrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (r RefundedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{}
}
