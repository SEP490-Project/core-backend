package enum

import (
	"database/sql/driver"
	"fmt"
)

type KPIReferenceType string

const (
	KPIReferenceTypeContent  KPIReferenceType = "CONTENT"
	KPIReferenceTypeCampaign KPIReferenceType = "CAMPAIGN"
)

func (rt KPIReferenceType) IsValid() bool {
	switch rt {
	case KPIReferenceTypeContent, KPIReferenceTypeCampaign:
		return true
	}
	return false
}

func (rt *KPIReferenceType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan KPIReferenceType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*rt = KPIReferenceType(s)
	return nil
}

func (rt KPIReferenceType) Value() (driver.Value, error) {
	return string(rt), nil
}

func (rt KPIReferenceType) String() string {
	return string(rt)
}
