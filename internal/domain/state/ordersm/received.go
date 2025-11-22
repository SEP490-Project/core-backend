package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ReceivedState struct{}

func (c ReceivedState) Name() enum.OrderStatus {
	return enum.OrderStatusCancelled
}

func (c ReceivedState) Next(ctx *OrderContext, next OrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c ReceivedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{}
}
