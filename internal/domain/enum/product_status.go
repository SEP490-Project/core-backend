package enum

import (
	"database/sql/driver"
	"fmt"
)

type ProductStatus string

const (
	ProductStatusDraft     ProductStatus = "DRAFT"
	ProductStatusSubmitted ProductStatus = "SUBMITTED"
	ProductStatusRevision  ProductStatus = "REVISION"
	ProductStatusApproved  ProductStatus = "APPROVED"
	ProductStatusActived   ProductStatus = "ACTIVED"
	ProductStatusInactived ProductStatus = "INACTIVED"
)

func (pt ProductStatus) IsValid() bool {
	switch pt {
	case ProductStatusDraft,
		ProductStatusSubmitted,
		ProductStatusRevision,
		ProductStatusApproved,
		ProductStatusActived,
		ProductStatusInactived:
		return true
	}
	return false
}

func (pt *ProductStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ProductStatus: invalid status %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*pt = ProductStatus(s)
	return nil
}

func (pt ProductStatus) Value() (driver.Value, error) {
	return string(pt), nil
}
