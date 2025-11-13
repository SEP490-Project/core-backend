package enum

import (
	"database/sql/driver"
	"fmt"
)

type ProductType string

const (
	ProductTypeStandard ProductType = "STANDARD"
	ProductTypeLimited  ProductType = "LIMITED"
)

func (pt ProductType) IsValid() bool {
	switch pt {
	case ProductTypeStandard, ProductTypeLimited:
		return true
	}
	return false
}

func (pt *ProductType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ProductType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*pt = ProductType(s)
	return nil
}

func (pt ProductType) Value() (driver.Value, error) {
	return string(pt), nil
}

func (pt ProductType) String() string {
	return string(pt)
}
