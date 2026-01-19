package asynqtask

import (
	"time"

	"github.com/google/uuid"
)

type AutoCloseMilestoneTaskPayload struct {
	RequestID           string                  `json:"request_id"`
	ContractID          uuid.UUID               `json:"contract_id"`
	MilestoneDueDateMap map[uuid.UUID]time.Time `json:"milestone_due_date_map"`
}
