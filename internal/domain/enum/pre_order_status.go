package enum

import (
	"database/sql/driver"
	"fmt"
)

type PreOrderStatus string

const (
	PreOrderStatusPending       PreOrderStatus = "PENDING"
	PreOrderStatusPaid          PreOrderStatus = "PAID"
	PreOrderStatusPreOrdered    PreOrderStatus = "PRE_ORDERED"
	PreOrderStatusCancelled     PreOrderStatus = "CANCELLED"
	PreOrderStatusRefundRequest PreOrderStatus = "REFUND_REQUEST"
	PreOrderStatusRefunded      PreOrderStatus = "REFUNDED"

	PreOrderStatusAwaitingPickup    PreOrderStatus = "AWAITING_PICKUP"
	PreOrderStatusInTransit         PreOrderStatus = "IN_TRANSIT"
	PreOrderStatusDelivered         PreOrderStatus = "DELIVERED"
	PreOrderStatusCompensateRequest PreOrderStatus = "COMPENSATE_REQUEST"
	PreOrderStatusCompensated       PreOrderStatus = "COMPENSATED"
	PreOrderStatusReceived          PreOrderStatus = "RECEIVED"
)

func (pos PreOrderStatus) IsValid() bool {
	switch pos {
	case PreOrderStatusPending, PreOrderStatusPaid, PreOrderStatusPreOrdered, PreOrderStatusCancelled, PreOrderStatusAwaitingPickup, PreOrderStatusInTransit, PreOrderStatusDelivered, PreOrderStatusReceived, PreOrderStatusCompensateRequest, PreOrderStatusCompensated, PreOrderStatusRefundRequest, PreOrderStatusRefunded:
		return true
	}
	return false
}

func (pos *PreOrderStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan PreOrderStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*pos = PreOrderStatus(s)
	return nil
}

func (pos PreOrderStatus) Value() (driver.Value, error) {
	return string(pos), nil
}

func (pos PreOrderStatus) String() string {
	return string(pos)
}
