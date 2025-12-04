package constant

import "core-backend/internal/domain/enum"

// region: ======= Valid Order Status =========

var (
	ValidCompletedOrderStatus = []enum.OrderStatus{
		enum.OrderStatusDelivered,
		enum.OrderStatusReceived,
		enum.OrderStatusCompensateRequested,
	}
	ValidPendingOrderStatus = []enum.OrderStatus{
		enum.OrderStatusPending,
		enum.OrderStatusPaid,
		enum.OrderStatusConfirmed,
		enum.OrderStatusShipped,
		enum.OrderStatusInTransit,
		enum.OrderStatusAwaitingPickUp,
		enum.OrderStatusRefundRequested,
	}
	ValidCancelledOrderStatus = []enum.OrderStatus{
		enum.OrderStatusCancelled,
	}
	ValidRefundedOrderStatus = []enum.OrderStatus{
		enum.OrderStatusRefunded,
		enum.OrderStatusCompensated,
	}
)

// endregion

// region: ======= Valid Pre-Order Status =========

var (
	ValidCompletedPreOrderStatus = []enum.PreOrderStatus{
		enum.PreOrderStatusDelivered,
		enum.PreOrderStatusReceived,
		enum.PreOrderStatusCompensateRequest,
	}
	ValidPendingPreOrderStatus = []enum.PreOrderStatus{
		enum.PreOrderStatusPending,
		enum.PreOrderStatusPaid,
		enum.PreOrderStatusPreOrdered,
		enum.PreOrderStatusRefundRequest,
		enum.PreOrderStatusAwaitingPickup,
		enum.PreOrderStatusInTransit,
	}
	ValidCancelledPreOrderStatus = []enum.PreOrderStatus{
		enum.PreOrderStatusCancelled,
	}
	ValidRefundedPreOrderStatus = []enum.PreOrderStatus{
		enum.PreOrderStatusRefunded,
		enum.PreOrderStatusCompensated,
	}
)

// endregion
