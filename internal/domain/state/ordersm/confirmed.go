package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ConfirmedState struct{}

func (c ConfirmedState) Name() enum.OrderStatus {
	return enum.OrderStatusConfirmed
}

func (c ConfirmedState) Next(ctx *OrderContext, next OrderState) error {
	if _, ok := c.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", c.Name(), next.Name())
}

func (c ConfirmedState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		//For Standard orders
		enum.OrderStatusShipped:   {},
		enum.OrderStatusCancelled: {},

		//For Limited Orders
		enum.OrderStatusInTransit:      {},
		enum.OrderStatusAwaitingPickUp: {},
	}
}

//func (c ConfirmedState) limitedStatesInterceptor(status enum.OrderStatus, ctx *OrderContext) error {
//	if status == enum.OrderStatusInTransit {
//		if ctx.Order == nil || ctx.Order.OrderItems == nil || len(ctx.Order.OrderItems) == 0 {
//			return fmt.Errorf("order not found in context")
//		}
//		//Make sure it's a limited order
//		for _, item := range ctx.Order.OrderItems {
//			if item.Variant.Product {
//				return nil
//			}
//		}
//		return fmt.Errorf("invalid transition: %s -> %s for standard order", c.Name(), status)
//	}
//}
