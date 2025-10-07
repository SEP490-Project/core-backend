package enum

import (
	"database/sql/driver"
	"fmt"
)

type MilestoneStatus string

const (
	MilestoneStatusNotStarted MilestoneStatus = "NOT_STARTED"
	MilestoneStatusOnGoing    MilestoneStatus = "ON_GOING"
	MilestoneStatusCancelled  MilestoneStatus = "CANCELLED"
	MilestoneStatusCompleted  MilestoneStatus = "COMPLETED"
)

func (ms MilestoneStatus) IsValid() bool {
	switch ms {
	case MilestoneStatusNotStarted, MilestoneStatusOnGoing, MilestoneStatusCancelled, MilestoneStatusCompleted:
		return true
	}
	return false
}

func (ms *MilestoneStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan MilestoneStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ms = MilestoneStatus(s)
	return nil
}

func (ms MilestoneStatus) Value() (driver.Value, error) {
	return string(ms), nil
}

func (ms MilestoneStatus) String() string {
	return string(ms)
}
