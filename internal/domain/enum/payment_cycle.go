package enum

import (
	"database/sql/driver"
	"fmt"
)

type PaymentCycle string

const (
	PaymentCycleMonthly   PaymentCycle = "MONTHLY"
	PaymentCycleQuarterly PaymentCycle = "QUARTERLY"
	PaymentCycleAnnually  PaymentCycle = "ANNUALLY"
)

func (pc PaymentCycle) IsValid() bool {
	switch pc {
	case PaymentCycleMonthly, PaymentCycleQuarterly, PaymentCycleAnnually:
		return true
	}
	return false
}

func (pc *PaymentCycle) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan AddressType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*pc = PaymentCycle(s)
	return nil
}

func (pc PaymentCycle) Value() (driver.Value, error) {
	return string(pc), nil
}
