package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type Refunded struct{}

func (c Refunded) Name() enum.PreOrderStatus {
	return enum.PreOrderStatusRefunded
}

func (c Refunded) Next(ctx *PreOrderContext, next PreOrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c Refunded) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{}
}
