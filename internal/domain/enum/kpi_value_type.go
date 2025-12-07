package enum

import (
	"database/sql/driver"
	"fmt"
)

type KPIValueType string

const (
	KPIValueTypeReach        KPIValueType = "REACH"
	KPIValueTypeLikes        KPIValueType = "LIKES"
	KPIValueTypeImpressions  KPIValueType = "IMPRESSIONS"
	KPIValueTypeComments     KPIValueType = "COMMENTS"
	KPIValueTypeShares       KPIValueType = "SHARES"
	KPIValueTypeEngagement   KPIValueType = "ENGAGEMENT"
	KPIValueTypeClickThrough KPIValueType = "CLICK_THROUGH"
	KPIValueTypeUnitsSold    KPIValueType = "UNITS_SOLD"
	KPIValueTypeRevenue      KPIValueType = "REVENUE"
)

func (vt KPIValueType) IsValid() bool {
	switch vt {
	case KPIValueTypeReach, KPIValueTypeLikes, KPIValueTypeImpressions, KPIValueTypeComments, KPIValueTypeShares, KPIValueTypeEngagement, KPIValueTypeClickThrough:
		return true
	}
	return false
}

func (vt *KPIValueType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan KPIValueType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*vt = KPIValueType(s)
	return nil
}

func (vt KPIValueType) Value() (driver.Value, error) {
	return string(vt), nil
}

func (vt KPIValueType) String() string {
	return string(vt)
}
