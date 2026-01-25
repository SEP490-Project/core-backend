package enum

import (
	"database/sql/driver"
	"fmt"
)

// ViolationType represents who violated the contract
type ViolationType string

const (
	ViolationTypeBrand ViolationType = "BRAND"
	ViolationTypeKOL   ViolationType = "KOL"
)

func (vt ViolationType) IsValid() bool {
	switch vt {
	case ViolationTypeBrand, ViolationTypeKOL:
		return true
	}
	return false
}

// Scan implements the sql.Scanner interface
func (vt *ViolationType) Scan(value any) error {
	if value == nil {
		*vt = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*vt = ViolationType(v)
	case []byte:
		*vt = ViolationType(v)
	default:
		return fmt.Errorf("failed to scan ViolationType: unexpected type %T", value)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (vt ViolationType) Value() (driver.Value, error) {
	return string(vt), nil
}

// String returns the string representation of ViolationType
func (vt ViolationType) String() string {
	return string(vt)
}
