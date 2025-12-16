package enum

import (
	"database/sql/driver"
	"fmt"
	"slices"
)

// ScheduleStatus represents the status of a content schedule
type ScheduleStatus string

const (
	ScheduleStatusPending    ScheduleStatus = "PENDING"
	ScheduleStatusProcessing ScheduleStatus = "PROCESSING"
	ScheduleStatusCompleted  ScheduleStatus = "COMPLETED"
	ScheduleStatusFailed     ScheduleStatus = "FAILED"
	ScheduleStatusCancelled  ScheduleStatus = "CANCELLED"
)

func (s ScheduleStatus) IsValid() bool {
	switch s {
	case ScheduleStatusPending, ScheduleStatusProcessing, ScheduleStatusCompleted, ScheduleStatusFailed, ScheduleStatusCancelled:
		return true
	}
	return false
}

func (s *ScheduleStatus) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*s = ScheduleStatus(v)
	case string:
		*s = ScheduleStatus(v)
	default:
		return fmt.Errorf("failed to scan ScheduleStatus: invalid type %T", value)
	}
	return nil
}

func (s ScheduleStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s ScheduleStatus) String() string { return string(s) }

// IsFinal returns true if the status is a terminal state
func (s ScheduleStatus) IsFinal() bool {
	return s == ScheduleStatusCompleted || s == ScheduleStatusFailed || s == ScheduleStatusCancelled
}

// CanTransitionTo checks if a transition to the target status is valid
func (s ScheduleStatus) CanTransitionTo(target ScheduleStatus) bool {
	transitions := map[ScheduleStatus][]ScheduleStatus{
		ScheduleStatusPending:    {ScheduleStatusProcessing, ScheduleStatusCancelled},
		ScheduleStatusProcessing: {ScheduleStatusCompleted, ScheduleStatusFailed, ScheduleStatusPending},
		ScheduleStatusFailed:     {}, // Terminal state
		ScheduleStatusCompleted:  {}, // Terminal state
		ScheduleStatusCancelled:  {}, // Terminal state
	}

	allowed, exists := transitions[s]
	if !exists {
		return false
	}

	return slices.Contains(allowed, target)
}
