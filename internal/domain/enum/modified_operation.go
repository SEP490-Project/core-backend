package enum

import (
	"database/sql/driver"
	"fmt"
)

type ModifiedOperation string

const (
	ModifiedOperationCreate ModifiedOperation = "CREATE"
	ModifiedOperationUpdate ModifiedOperation = "UPDATE"
	ModifiedOperationDelete ModifiedOperation = "DELETE"
)

func (mo ModifiedOperation) IsValid() bool {
	switch mo {
	case ModifiedOperationCreate, ModifiedOperationUpdate, ModifiedOperationDelete:
		return true
	}
	return false
}

func (mo *ModifiedOperation) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ModifiedOperation: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*mo = ModifiedOperation(s)
	return nil
}

func (mo ModifiedOperation) Value() (driver.Value, error) {
	return string(mo), nil
}

func (mo ModifiedOperation) String() string { return string(mo) }
