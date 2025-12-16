package constant

import "core-backend/internal/domain/enum"

// region: ======= Valid Order Status =========

type ValidOrderStatusGroup []enum.OrderStatus

var (
	ValidCompletedOrderStatus ValidOrderStatusGroup = []enum.OrderStatus{
		enum.OrderStatusDelivered,
		enum.OrderStatusReceived,
		enum.OrderStatusCompensateRequested,
	}
	ValidPendingOrderStatus ValidOrderStatusGroup = []enum.OrderStatus{
		enum.OrderStatusPending,
		enum.OrderStatusPaid,
		enum.OrderStatusConfirmed,
		enum.OrderStatusShipped,
		enum.OrderStatusInTransit,
		enum.OrderStatusAwaitingPickUp,
		enum.OrderStatusRefundRequested,
	}
	ValidCancelledOrderStatus ValidOrderStatusGroup = []enum.OrderStatus{
		enum.OrderStatusCancelled,
	}
	ValidRefundedOrderStatus ValidOrderStatusGroup = []enum.OrderStatus{
		enum.OrderStatusRefunded,
		enum.OrderStatusCompensated,
	}
)

func (os *ValidOrderStatusGroup) ToStringSlice() (result []string) {
	for _, s := range *os {
		result = append(result, s.String())
	}
	return result
}

// endregion

// region: ======= Valid Pre-Order Status =========

type ValidPreOrderStatusGroup []enum.PreOrderStatus

var (
	ValidCompletedPreOrderStatus ValidPreOrderStatusGroup = []enum.PreOrderStatus{
		enum.PreOrderStatusDelivered,
		enum.PreOrderStatusReceived,
		enum.PreOrderStatusCompensateRequest,
	}
	ValidPendingPreOrderStatus ValidPreOrderStatusGroup = []enum.PreOrderStatus{
		enum.PreOrderStatusPending,
		enum.PreOrderStatusPaid,
		enum.PreOrderStatusPreOrdered,
		enum.PreOrderStatusRefundRequest,
		enum.PreOrderStatusAwaitingPickup,
		enum.PreOrderStatusInTransit,
	}
	ValidCancelledPreOrderStatus ValidPreOrderStatusGroup = []enum.PreOrderStatus{
		enum.PreOrderStatusCancelled,
	}
	ValidRefundedPreOrderStatus ValidPreOrderStatusGroup = []enum.PreOrderStatus{
		enum.PreOrderStatusRefunded,
		enum.PreOrderStatusCompensated,
	}
)

func (pos *ValidPreOrderStatusGroup) ToStringSlice() (result []string) {
	for _, s := range *pos {
		result = append(result, s.String())
	}
	return result
}

// endregion
