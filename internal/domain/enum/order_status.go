package enum

import (
	"database/sql/driver"
	"fmt"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusRefunded  OrderStatus = "REFUNDED"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusCanceled  OrderStatus = "CANCELED"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusInTransit OrderStatus = "IN_TRANSIT"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusReceived  OrderStatus = "RECEIVED"
)

func (os OrderStatus) IsValid() bool {
	switch os {
	case OrderStatusPending, OrderStatusPaid, OrderStatusRefunded, OrderStatusConfirmed, OrderStatusCanceled, OrderStatusShipped, OrderStatusInTransit, OrderStatusDelivered, OrderStatusReceived:
		return true
	}
	return false
}

func (os *OrderStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan OrderStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*os = OrderStatus(s)
	return nil
}

func (os OrderStatus) Value() (driver.Value, error) {
	return string(os), nil
}
