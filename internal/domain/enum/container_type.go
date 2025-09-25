package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContainerType string

const (
	ContainerTypeBottle       ContainerType = "BOTTLE"
	ContainerTypeTube         ContainerType = "TUBE"
	ContainerTypeJar          ContainerType = "JAR"
	ContainerTypeStick        ContainerType = "STICK"
	ContainerTypePencil       ContainerType = "PENCIL"
	ContainerTypeCompact      ContainerType = "COMPACT"
	ContainerTypePallete      ContainerType = "PALLETE"
	ContainerTypeSachet       ContainerType = "SACHET"
	ContainerTypeVial         ContainerType = "VIAL"
	ContainerTypeRollerBottle ContainerType = "ROLLER_BOTTLE"
)

func (ct ContainerType) IsValid() bool {
	switch ct {
	case ContainerTypeBottle, ContainerTypeTube, ContainerTypeJar, ContainerTypeStick, ContainerTypePencil, ContainerTypeCompact, ContainerTypePallete, ContainerTypeSachet, ContainerTypeVial, ContainerTypeRollerBottle:
		return true
	}
	return false
}

func (ct *ContainerType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContainerType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ct = ContainerType(s)
	return nil
}

func (ct ContainerType) Value() (driver.Value, error) {
	return string(ct), nil
}
