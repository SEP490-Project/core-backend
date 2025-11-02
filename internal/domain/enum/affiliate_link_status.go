package enum

import (
	"database/sql/driver"
	"fmt"
)

// AffiliateLinkStatus represents the status of an affiliate link
type AffiliateLinkStatus string

const (
	AffiliateLinkStatusActive   AffiliateLinkStatus = "active"
	AffiliateLinkStatusInactive AffiliateLinkStatus = "inactive"
	AffiliateLinkStatusExpired  AffiliateLinkStatus = "expired"
)

func (s AffiliateLinkStatus) IsValid() bool {
	switch s {
	case AffiliateLinkStatusActive, AffiliateLinkStatusInactive, AffiliateLinkStatusExpired:
		return true
	}
	return false
}

func (s *AffiliateLinkStatus) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*s = AffiliateLinkStatus(v)
	case string:
		*s = AffiliateLinkStatus(v)
	default:
		return fmt.Errorf("failed to scan AffiliateLinkStatus: invalid type %T", value)
	}

	if !s.IsValid() {
		return fmt.Errorf("invalid AffiliateLinkStatus value: %s", string(*s))
	}

	return nil
}

func (s AffiliateLinkStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s AffiliateLinkStatus) String() string {
	return string(s)
}
