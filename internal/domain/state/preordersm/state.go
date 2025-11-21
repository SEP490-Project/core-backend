package preordersm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// PreOrderState interface defines behavior for pre-order states
type PreOrderState interface {
	Name() enum.PreOrderStatus
	Next(ctx *PreOrderContext, next PreOrderState) error
	AllowedTransitions() map[enum.PreOrderStatus]bool
}

// NewPreOrderState factory method to create state instances
func NewPreOrderState(status enum.PreOrderStatus) (PreOrderState, error) {
	switch status {
	case enum.PreOrderStatusPending:
		return &PendingState{}, nil
	case enum.PreOrderStatusPaid:
		return &PaidState{}, nil
	case enum.PreOrderStatusPreOrdered:
		return &PreOrderedState{}, nil
	case enum.PreOrderStatusCancelled:
		return &CancelledState{}, nil
	case enum.PreOrderStatusAwaitingPickup:
		return &AwaitingPickupState{}, nil
	case enum.PreOrderStatusInTransit:
		return &InTransitState{}, nil
	case enum.PreOrderStatusDelivered:
		return &DeliveredState{}, nil
	case enum.PreOrderStatusReceived:
		return &ReceivedState{}, nil
	default:
		return nil, errors.New("invalid pre-order status: " + string(status))
	}
}
