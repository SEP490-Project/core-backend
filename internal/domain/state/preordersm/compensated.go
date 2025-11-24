package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type Compensated struct{}

func (c Compensated) Name() enum.PreOrderStatus {
	return enum.PreOrderStatusCompensated
}

func (c Compensated) Next(ctx *PreOrderContext, next PreOrderState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c Compensated) AllowedTransitions() map[enum.PreOrderStatus]struct{} {
	return map[enum.PreOrderStatus]struct{}{}
}
