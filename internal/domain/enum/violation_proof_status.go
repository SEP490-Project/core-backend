package enum

import (
	"database/sql/driver"
	"fmt"
)

// ViolationProofStatus represents the status of a KOL's refund proof
type ViolationProofStatus string

const (
	ViolationProofStatusPending  ViolationProofStatus = "PENDING"
	ViolationProofStatusApproved ViolationProofStatus = "APPROVED"
	ViolationProofStatusRejected ViolationProofStatus = "REJECTED"
)

func (vps ViolationProofStatus) IsValid() bool {
	switch vps {
	case ViolationProofStatusPending, ViolationProofStatusApproved, ViolationProofStatusRejected:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (vps *ViolationProofStatus) Scan(value any) error {
	if value == nil {
		*vps = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*vps = ViolationProofStatus(v)
	case []byte:
		*vps = ViolationProofStatus(v)
	default:
		return fmt.Errorf("failed to scan ViolationProofStatus: unexpected type %T", value)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (vps ViolationProofStatus) Value() (driver.Value, error) {
	return string(vps), nil
}

// String returns the string representation of ViolationProofStatus
func (vps ViolationProofStatus) String() string {
	return string(vps)
}
