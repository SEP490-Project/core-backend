package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type Compensated struct{}

func (c Compensated) Name() enum.OrderStatus {
	return enum.OrderStatusCompensated
}

func (c Compensated) Next(ctx *OrderContext, next OrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c Compensated) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{}
}
