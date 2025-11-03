package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContractPaymentMethod string

const (
	ContractPaymentMethodBankTransfer ContractPaymentMethod = "BANK_TRANSFER"
	ContractPaymentMethodCash         ContractPaymentMethod = "CASH"
	ContractPaymentMethodCheck        ContractPaymentMethod = "CHECK"
)

func (cpm ContractPaymentMethod) IsValid() bool {
	switch cpm {
	case ContractPaymentMethodBankTransfer, ContractPaymentMethodCash, ContractPaymentMethodCheck:
		return true
	}
	return false
}

func (cpm *ContractPaymentMethod) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractPaymentMethod: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cpm = ContractPaymentMethod(s)
	return nil
}

func (cpm ContractPaymentMethod) Value() (driver.Value, error) {
	return string(cpm), nil
}

func (cpm ContractPaymentMethod) String() string { return string(cpm) }
