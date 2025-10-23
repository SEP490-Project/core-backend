package enum

import (
	"database/sql/driver"
	"fmt"
)

type DispenserType string

const (
	DispenserTypePump    DispenserType = "PUMP"
	DispenserTypeSpray   DispenserType = "SPRAY"
	DispenserTypeDropper DispenserType = "DROPPER"
	DispenserTypeRollOn  DispenserType = "ROLL_ON"
	DispenserTypeTwistUp DispenserType = "TWIST_UP"
	DispenserTypeSqueeze DispenserType = "SQUEEZE"
	DispenserTypeNone    DispenserType = "NONE"
)

func (dt DispenserType) IsValid() bool {
	switch dt {
	case DispenserTypePump, DispenserTypeSpray, DispenserTypeDropper, DispenserTypeRollOn, DispenserTypeTwistUp, DispenserTypeSqueeze, DispenserTypeNone:
		return true
	}
	return false
}

func (dt *DispenserType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan DispenserType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*dt = DispenserType(s)
	return nil
}

func (dt DispenserType) Value() (driver.Value, error) {
	return string(dt), nil
}

func (dt DispenserType) String() string {
	return string(dt)
}
