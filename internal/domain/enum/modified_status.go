package enum

import (
	"database/sql/driver"
	"fmt"
)

type ModifiedStatus string

const (
	ModifiedStatusInProgress ModifiedStatus = "IN_PROGRESS"
	ModifiedStatusCompleted  ModifiedStatus = "COMPLETED"
	ModifiedStatusFailed     ModifiedStatus = "FAILED"
)

func (ms ModifiedStatus) IsValid() bool {
	switch ms {
	case ModifiedStatusInProgress, ModifiedStatusCompleted, ModifiedStatusFailed:
		return true
	}
	return false
}

func (ms *ModifiedStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ModifiedStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ms = ModifiedStatus(s)
	return nil
}

func (ms ModifiedStatus) Value() (driver.Value, error) {
	return string(ms), nil
}

func (ms ModifiedStatus) String() string { return string(ms) }
