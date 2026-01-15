package asynqtask

import "github.com/google/uuid"

// PreOrderAutoReceiveTaskPayload is the payload for automatically marking a pre-order as RECEIVED
// after a configured number of days (default 30 days) from when it was delivered.
type PreOrderAutoReceiveTaskPayload struct {
	PreOrderID uuid.UUID `json:"preorder_id"`
}
