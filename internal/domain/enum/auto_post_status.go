package enum

import (
	"database/sql/driver"
	"fmt"
)

type AutoPostStatus string

const (
	AutoPostStatusPending AutoPostStatus = "PENDING"
	AutoPostStatusPosted  AutoPostStatus = "POSTED"
	AutoPostStatusFailed  AutoPostStatus = "FAILED"
	AutoPostStatusSkipped AutoPostStatus = "SKIPPED"
)

func (aps AutoPostStatus) IsValid() bool {
	switch aps {
	case AutoPostStatusPending, AutoPostStatusPosted, AutoPostStatusFailed, AutoPostStatusSkipped:
		return true
	}
	return false
}

func (aps *AutoPostStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan AutoPostStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*aps = AutoPostStatus(s)
	return nil
}

func (aps AutoPostStatus) Value() (driver.Value, error) {
	return string(aps), nil
}

func (aps AutoPostStatus) String() string {
	return string(aps)
}
