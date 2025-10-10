package enum

import (
	"database/sql/driver"
	"fmt"
)

type AddressType string

const (
	AddressTypeBilling  AddressType = "BILLING"
	AddressTypeShipping AddressType = "SHIPPING"
)

func (at AddressType) IsValid() bool {
	switch at {
	case AddressTypeBilling, AddressTypeShipping:
		return true
	}
	return false
}

func (at *AddressType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan AddressType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*at = AddressType(s)
	return nil
}

func (at AddressType) Value() (driver.Value, error) {
	return string(at), nil
}

func (at AddressType) String() string { return string(at) }
