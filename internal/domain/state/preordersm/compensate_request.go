package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompensateRequestStated struct{}

func (i CompensateRequestStated) Name() enum.PreOrderStatus {
	return enum.PreOrderStatusCompensateRequest
}

func (i CompensateRequestStated) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i CompensateRequestStated) AllowedTransitions() map[enum.PreOrderStatus]struct{} {
	return map[enum.PreOrderStatus]struct{}{
		enum.PreOrderStatusCompensated: {},
		enum.PreOrderStatusDelivered:   {},
	}
}
