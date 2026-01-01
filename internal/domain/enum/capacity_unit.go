package enum

// Deprecated: CapacityUnit is deprecated and will be removed in a future version.
// Use the product_options table with type='CAPACITY_UNIT' instead.
// Values are now stored as strings and managed via ProductOptionService.
// See: internal/application/service/product_option_service.go

import (
	"database/sql/driver"
	"fmt"
)

type CapacityUnit string

const (
	CapacityUnitML CapacityUnit = "ML"
	CapacityUnitL  CapacityUnit = "L"
	CapacityUnitG  CapacityUnit = "G"
	CapacityUnitKG CapacityUnit = "KG"
	CapacityUnitOZ CapacityUnit = "OZ"
)

func (cu CapacityUnit) IsValid() bool {
	switch cu {
	case CapacityUnitML, CapacityUnitL, CapacityUnitG, CapacityUnitKG, CapacityUnitOZ:
		return true
	}
	return false
}

func (cu *CapacityUnit) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan CapacityUnit: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cu = CapacityUnit(s)
	return nil
}

func (cu CapacityUnit) Value() (driver.Value, error) {
	return string(cu), nil
}

func (cu CapacityUnit) String() string {
	return string(cu)
}
