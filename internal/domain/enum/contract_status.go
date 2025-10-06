package enum

import (
	"database/sql/driver"
	"fmt"
)

// ContractStatus represents the status of a contract.
// Possible values are: "DRAFT", "ACTIVE", "COMPLETED", "TERMINATED".
type ContractStatus string

const (
	ContractStatusDraft      ContractStatus = "DRAFT"
	ContractStatusActive     ContractStatus = "ACTIVE"
	ContractStatusCompleted  ContractStatus = "COMPLETED"
	ContractStatusTerminated ContractStatus = "TERMINATED"
)

func (cs ContractStatus) IsValid() bool {
	switch cs {
	case ContractStatusDraft, ContractStatusActive, ContractStatusCompleted, ContractStatusTerminated:
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

func (cs ContractStatus) String() string {
	return string(cs)
}
