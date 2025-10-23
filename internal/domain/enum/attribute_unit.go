package enum

import (
	"database/sql/driver"
	"fmt"
)

type AttributeUnit string

const (
	AttributeUnitPercent AttributeUnit = "%"
	AttributeUnitMG      AttributeUnit = "MG"
	AttributeUnitG       AttributeUnit = "G"
	AttributeUnitML      AttributeUnit = "ML"
	AttributeUnitL       AttributeUnit = "L"
	AttributeUnitIU      AttributeUnit = "IU"
	AttributeUnitPPM     AttributeUnit = "PPM"
	AttributeUnitNone    AttributeUnit = "NONE"
)

func (au AttributeUnit) IsValid() bool {
	switch au {
	case AttributeUnitPercent, AttributeUnitMG, AttributeUnitG, AttributeUnitML, AttributeUnitL, AttributeUnitIU, AttributeUnitPPM, AttributeUnitNone:
		return true
	}
	return false
}

func (au *AttributeUnit) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan AttributeUnit: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*au = AttributeUnit(s)
	return nil
}

func (au AttributeUnit) Value() (driver.Value, error) {
	return string(au), nil
}

func (au AttributeUnit) String() string {
	return string(au)
}
