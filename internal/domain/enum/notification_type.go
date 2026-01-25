package enum

import (
	"database/sql/driver"
	"errors"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "EMAIL"
	NotificationTypePush  NotificationType = "PUSH"
	NotificationTypeInApp NotificationType = "IN_APP"
	NotificationTypeAll   NotificationType = "ALL"
)

// IsValid checks if the notification type is valid
func (nt NotificationType) IsValid() bool {
	switch nt {
	case NotificationTypeEmail, NotificationTypePush, NotificationTypeInApp, NotificationTypeAll:
		return true
	}
	return false
}

// Value implements the driver.Valuer interface for database compatibility
func (nt NotificationType) Value() (driver.Value, error) {
	if !nt.IsValid() {
		return nil, errors.New("invalid notification type")
	}
	return string(nt), nil
}

// Scan implements the sql.Scanner interface for database compatibility
func (nt *NotificationType) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		byteSlice, ok := value.([]byte)
		if !ok {
			return errors.New("failed to scan NotificationType: value is not a string or []byte")
		}
		str = string(byteSlice)
	}

	*nt = NotificationType(str)
	if !nt.IsValid() {
		return errors.New("invalid notification type value")
	}
	return nil
}

// String returns the string representation
func (nt NotificationType) String() string {
	return string(nt)
}
