package enum

import (
	"database/sql/driver"
	"fmt"
)

type TaskStatus string

const (
	TaskStatusToDo              TaskStatus = "TODO"
	TaskStatusInProgress        TaskStatus = "IN_PROGRESS"
	TaskStatusCancelled         TaskStatus = "CANCELLED"
	TaskStatusSubmitted         TaskStatus = "SUBMITTED"
	TaskStatusRevisionRequested TaskStatus = "REVISION_REQUESTED"
	TaskStatusApproved          TaskStatus = "APPROVED"
	TaskStatusOnRelease         TaskStatus = "ON_RELEASE"
	TaskStatusRecap             TaskStatus = "RECAP"
	TaskStatusDone              TaskStatus = "DONE"
)

func (ts TaskStatus) IsValid() bool {
	switch ts {
	case TaskStatusToDo, TaskStatusInProgress, TaskStatusCancelled, TaskStatusSubmitted, TaskStatusRevisionRequested, TaskStatusApproved, TaskStatusOnRelease, TaskStatusRecap, TaskStatusDone:
		return true
	}
	return false
}

func (ts *TaskStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan TaskStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ts = TaskStatus(s)
	return nil
}

func (ts TaskStatus) Value() (driver.Value, error) {
	return string(ts), nil
}
