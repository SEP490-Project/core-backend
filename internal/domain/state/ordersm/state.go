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
	case enum.OrderStatusRefundRequested:
		return &RefundRequestedState{}
	case enum.OrderStatusRefunded:
		return &RefundedState{}
	case enum.OrderStatusConfirmed:
		return &ConfirmedState{}
	case enum.OrderStatusShipped:
		return &ShippedState{}
	case enum.OrderStatusInTransit:
		return &InTransitState{}
	case enum.OrderStatusDelivered:
		return &DeliveredState{}
	case enum.OrderStatusCompensateRequested:
		return &CompensateRequestStated{}
	case enum.OrderStatusCompensated:
		return &Compensated{}
	case enum.OrderStatusReceived:
		return &ReceivedState{}
	case enum.OrderStatusCancelled:
		return &CancelledState{}
	case enum.OrderStatusAwaitingPickUp:
		return &AwaitingPickUpState{}
	default:
		return nil
	}
}
