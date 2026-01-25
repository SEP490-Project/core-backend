package asynqtask

import "github.com/google/uuid"

// AutoReceiveOrderTaskPayload is the payload for automatically marking an order as RECEIVED.
type AutoReceiveOrderTaskPayload struct {
	OrderID uuid.UUID `json:"order_id"`
}
