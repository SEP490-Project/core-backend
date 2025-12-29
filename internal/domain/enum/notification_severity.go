package enum

import (
	"database/sql/driver"
	"errors"
)

type NotificationSeverity string

const (
	NotificationSeverityInfo    NotificationSeverity = "INFO"
	NotificationSeverityWarn    NotificationSeverity = "WARN"
	NotificationSeverityError   NotificationSeverity = "ERROR"
	NotificationSeveritySuccess NotificationSeverity = "SUCCESS"
)

func (ns NotificationSeverity) IsValid() bool {
	switch ns {
	case NotificationSeverityInfo, NotificationSeverityWarn, NotificationSeverityError, NotificationSeveritySuccess:
		return true
	}
	return false
}

func (ns NotificationSeverity) String() string { return string(ns) }

func (ns NotificationSeverity) Value() (driver.Value, error) {
	if !ns.IsValid() {
		return nil, errors.New("invalid notification severity")
	}
	return string(ns), nil
}

func (ns *NotificationSeverity) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		byteSlice, ok := value.([]byte)
		if !ok {
			return errors.New("failed to scan NotificationSeverity: value is not a string or []byte")
		}
		str = string(byteSlice)
	}

	*ns = NotificationSeverity(str)
	if !ns.IsValid() {
		return errors.New("invalid notification severity value")
	}
	return nil
}
