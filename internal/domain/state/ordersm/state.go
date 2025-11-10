package ordersm

import (
	"core-backend/internal/domain/enum"
)

type OrderState interface {
	Name() enum.OrderStatus
	Next(ctx *OrderContext, next OrderState) error
	AllowedTransitions() map[enum.OrderStatus]struct{}
}

func NewOrderState(status enum.OrderStatus) OrderState {
	switch status {
	case enum.OrderStatusPending:
		return &PendingState{}
	case enum.OrderStatusPaid:
		return &PaidState{}
	case enum.OrderStatusConfirmed:
		return &ConfirmedState{}
	case enum.OrderStatusShipped:
		return &ShippedState{}
	case enum.OrderStatusInTransit:
		return &InTransitState{}
	case enum.OrderStatusDelivered:
		return &DeliveredState{}
	case enum.OrderStatusReceived:
		return nil
	case enum.OrderStatusCancelled:
		return &CancelledState{}
	case enum.OrderStatusRefunded:
		return &RefundedState{}
	default:
		return nil
	}
}
