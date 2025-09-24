package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContractStatus string

const (
	ContractStatusActive   ContractStatus = "ACTIVE"
	ContractStatusPending  ContractStatus = "PENDING"
	ContractStatusExpired  ContractStatus = "EXPIRED"
	ContractStatusCanceled ContractStatus = "CANCELED"
)

func (cs ContractStatus) IsValid() bool {
	switch cs {
	case ContractStatusActive, ContractStatusPending, ContractStatusExpired, ContractStatusCanceled:
		return true
	}
	return false
}

func (cs *ContractStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cs = ContractStatus(s)
	return nil
}

func (cs ContractStatus) Value() (driver.Value, error) {
	return string(cs), nil
}
