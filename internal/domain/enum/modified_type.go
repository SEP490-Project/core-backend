package enum

import (
	"database/sql/driver"
	"fmt"
)

type ModifiedType string

const (
	ModifiedTypeCampaign  ModifiedType = "CAMPAIGN"
	ModifiedTypeMilestone ModifiedType = "MILESTONE"
	ModifiedTypeTask      ModifiedType = "TASK"
	ModifiedTypeContent   ModifiedType = "CONTENT"
	ModifiedTypeProduct   ModifiedType = "PRODUCT"
	ModifiedTypeBlog      ModifiedType = "BLOG"
)

func (mt ModifiedType) IsValid() bool {
	switch mt {
	case ModifiedTypeCampaign, ModifiedTypeMilestone, ModifiedTypeTask, ModifiedTypeContent, ModifiedTypeProduct, ModifiedTypeBlog:
		return true
	}
	return false
}

func (mt *ModifiedType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ModifiedType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*mt = ModifiedType(s)
	return nil
}

func (mt ModifiedType) Value() (driver.Value, error) {
	return string(mt), nil
}

func (mt ModifiedType) String() string { return string(mt) }
