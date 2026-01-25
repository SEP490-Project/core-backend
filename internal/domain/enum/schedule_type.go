package enum

import (
	"database/sql/driver"
	"fmt"
	"slices"
)

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleTypeContentPublish       ScheduleType = "CONTENT_PUBLISH"
	ScheduleTypeContractNotification ScheduleType = "CONTRACT_NOTIFICATION"
	ScheduleTypeOther                ScheduleType = "OTHER"
)

func (s ScheduleType) IsValid() bool {
	return slices.Contains([]ScheduleType{
		ScheduleTypeContentPublish,
		ScheduleTypeContractNotification,
		ScheduleTypeOther,
	}, s)
}

func (s ScheduleType) String() string {
	return string(s)
}

func (s *ScheduleType) Scan(value any) error {
	if value == nil {
		return nil
	}
	str, ok := value.(string)
	if !ok {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("failed to scan ScheduleType: %v", value)
		}
		str = string(bytes)
	}
	*s = ScheduleType(str)
	return nil
}

func (s ScheduleType) Value() (driver.Value, error) {
	return string(s), nil
}
