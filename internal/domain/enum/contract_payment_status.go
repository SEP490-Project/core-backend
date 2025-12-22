package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContractPaymentStatus string

const (
	ContractPaymentStatusPending    ContractPaymentStatus = "PENDING"
	ContractPaymentStatusPaid       ContractPaymentStatus = "PAID"
	ContractPaymentStatusOverdue    ContractPaymentStatus = "OVERDUE"
	ContractPaymentStatusTerminated ContractPaymentStatus = "TERMINATED"
)

func (cps ContractPaymentStatus) IsValid() bool {
	switch cps {
	case ContractPaymentStatusPending, ContractPaymentStatusPaid, ContractPaymentStatusOverdue:
		return true
	}
	return false
}

func (cps *ContractPaymentStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractPaymentStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cps = ContractPaymentStatus(s)
	return nil
}

func (cps ContractPaymentStatus) Value() (driver.Value, error) {
	return string(cps), nil
}

func (cps ContractPaymentStatus) String() string { return string(cps) }
