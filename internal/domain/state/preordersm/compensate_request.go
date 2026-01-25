package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompensateRequestStated struct{}

func (i CompensateRequestStated) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusCompensated: true,
		enum.PreOrderStatusDelivered:   true,
	}
}

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
