package preordersm

import (
	"core-backend/internal/domain/enum"
)

// PreOrderState interface defines behavior for pre-order states
type PreOrderState interface {
	Name() enum.PreOrderStatus
	Next(ctx *PreOrderContext, next PreOrderState) error
	AllowedTransitions() map[enum.PreOrderStatus]bool
}

// NewPreOrderState factory method to create state instances
func NewPreOrderState(status enum.PreOrderStatus) PreOrderState {
	switch status {
	case enum.PreOrderStatusPending:
		return &PendingState{}
	case enum.PreOrderStatusPaid:
		return &PaidState{}
	case enum.PreOrderStatusPreOrdered:
		return &PreOrderedState{}
	case enum.PreOrderStatusCancelled:
		return &CancelledState{}
	case enum.PreOrderStatusAwaitingPickup:
		return &AwaitingPickupState{}
	case enum.PreOrderStatusInTransit:
		return &InTransitState{}
	case enum.PreOrderStatusDelivered:
		return &DeliveredState{}
	case enum.PreOrderStatusReceived:
		return &ReceivedState{}
	default:
		return nil
	}
}
