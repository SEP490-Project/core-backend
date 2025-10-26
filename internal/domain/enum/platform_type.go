package enum

import (
	"database/sql/driver"
	"errors"
)

// PlatformType represents the mobile platform type
type PlatformType string

const (
	PlatformTypeIOS     PlatformType = "IOS"
	PlatformTypeAndroid PlatformType = "ANDROID"
)

// IsValid checks if the platform type is valid
func (pt PlatformType) IsValid() bool {
	switch pt {
	case PlatformTypeIOS, PlatformTypeAndroid:
		return true
	}
	return false
}

// Value implements the driver.Valuer interface for database compatibility
func (pt PlatformType) Value() (driver.Value, error) {
	if !pt.IsValid() {
		return nil, errors.New("invalid platform type")
	}
	return string(pt), nil
}

// Scan implements the sql.Scanner interface for database compatibility
func (pt *PlatformType) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		byteSlice, ok := value.([]byte)
		if !ok {
			return errors.New("failed to scan PlatformType: value is not a string or []byte")
		}
		str = string(byteSlice)
	}

	*pt = PlatformType(str)
	if !pt.IsValid() {
		return errors.New("invalid platform type value")
	}
	return nil
}

// String returns the string representation
func (pt PlatformType) String() string {
	return string(pt)
}
