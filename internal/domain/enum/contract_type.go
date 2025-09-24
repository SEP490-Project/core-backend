package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContractType string

const (
	ContractTypeAdvertising ContractType = "ADVERTISING"
	ContractTypeAffiliate   ContractType = "AFFILIATE"
	ContractTypeAmbassador  ContractType = "BRAND_AMBASSADOR"
	ContractTypeCoProduce   ContractType = "CO_PRODUCING"
)

func (ct ContractType) IsValid() bool {
	switch ct {
	case ContractTypeAdvertising, ContractTypeAffiliate, ContractTypeAmbassador, ContractTypeCoProduce:
		return true
	}
	return false
}

func (ct *ContractType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ct = ContractType(s)
	return nil
}

func (ct ContractType) Value() (driver.Value, error) {
	return string(ct), nil
}
