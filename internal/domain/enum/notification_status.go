package enum

import (
	"database/sql/driver"
	"errors"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending  NotificationStatus = "PENDING"
	NotificationStatusSent     NotificationStatus = "SENT"
	NotificationStatusFailed   NotificationStatus = "FAILED"
	NotificationStatusRetrying NotificationStatus = "RETRYING"
)

// IsValid checks if the notification status is valid
func (ns NotificationStatus) IsValid() bool {
	switch ns {
	case NotificationStatusPending, NotificationStatusSent, NotificationStatusFailed, NotificationStatusRetrying:
		return true
	}
	return false
}

// Value implements the driver.Valuer interface for database compatibility
func (ns NotificationStatus) Value() (driver.Value, error) {
	if !ns.IsValid() {
		return nil, errors.New("invalid notification status")
	}
	return string(ns), nil
}

// Scan implements the sql.Scanner interface for database compatibility
func (ns *NotificationStatus) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		byteSlice, ok := value.([]byte)
		if !ok {
			return errors.New("failed to scan NotificationStatus: value is not a string or []byte")
		}
		str = string(byteSlice)
	}

	*ns = NotificationStatus(str)
	if !ns.IsValid() {
		return errors.New("invalid notification status value")
	}
	return nil
}

// String returns the string representation
func (ns NotificationStatus) String() string {
	return string(ns)
}
