package asynqtask

import "github.com/google/uuid"

type PreOrderOpeningTaskPayload struct {
	PreOrderID uuid.UUID `json:"preorder_id"`
}
