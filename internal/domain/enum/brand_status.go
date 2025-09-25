package enum

import (
	"database/sql/driver"
	"fmt"
)

type BrandStatus string

const (
	BrandStatusActive   BrandStatus = "ACTIVE"
	BrandStatusInactive BrandStatus = "INACTIVE"
)

func (bs BrandStatus) IsValid() bool {
	switch bs {
	case BrandStatusActive, BrandStatusInactive:
		return true
	}
	return false
}

func (bs *BrandStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan BrandStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*bs = BrandStatus(s)
	return nil
}

func (bs BrandStatus) Value() (driver.Value, error) {
	return string(bs), nil
}
