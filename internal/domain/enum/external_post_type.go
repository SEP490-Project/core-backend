package enum

import (
	"database/sql/driver"
	"fmt"
)

type ExternalPostType string

const (
	ExternalPostTypeText        ExternalPostType = "TEXT"
	ExternalPostTypeSingleImage ExternalPostType = "SINGLE_IMAGE"
	ExternalPostTypeMultiImage  ExternalPostType = "MULTI_IMAGE"
	ExternalPostTypeVideo       ExternalPostType = "VIDEO"
	ExternalPostTypeLongVideo   ExternalPostType = "LONG_VIDEO"
)

func (ept ExternalPostType) IsValid() bool {
	switch ept {
	case ExternalPostTypeText,
		ExternalPostTypeSingleImage,
		ExternalPostTypeMultiImage,
		ExternalPostTypeVideo,
		ExternalPostTypeLongVideo:
		return true
	}
	return false
}

func (ept *ExternalPostType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ExternalPostType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ept = ExternalPostType(s)
	return nil
}

func (ept ExternalPostType) Value() (driver.Value, error) {
	return string(ept), nil
}

func (ept ExternalPostType) String() string { return string(ept) }
