package enum

import (
	"database/sql/driver"
	"fmt"
)

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusApproved  RefundStatus = "APPROVED"
	RefundStatusRejected  RefundStatus = "REJECTED"
	RefundStatusCompleted RefundStatus = "COMPLETED"
)

func (rs RefundStatus) IsValid() bool {
	switch rs {
	case RefundStatusPending, RefundStatusApproved, RefundStatusRejected, RefundStatusCompleted:
		return true
	}
	return false
}

func (rs *RefundStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan RefundStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*rs = RefundStatus(s)
	return nil
}

func (rs RefundStatus) Value() (driver.Value, error) {
	return string(rs), nil
}
