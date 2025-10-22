package ordersm

import "core-backend/internal/domain/enum"

type OrderContext struct {
	State OrderState
}

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
		return &Pai{}
	case enum.OrderStatusConfirmed:
		return &ShippedState{}
	case enum.OrderStatusShipped:
		return &DeliveredState{}
	case enum.OrderStatusInTransit:
		return &CancelledState{}
	case enum.OrderStatusDelivered:
		return &CompletedState{}
	case enum.OrderStatusReceived:
		return nil
	default:
		return nil
	}
}
