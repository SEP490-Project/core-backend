package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContentType string

const (
	ContentTypePost  ContentType = "POST"
	ContentTypeVideo ContentType = "VIDEO"
)

func (ct ContentType) IsValid() bool {
	switch ct {
	case ContentTypePost, ContentTypeVideo:
		return true
	}
	return false
}

func (ct *ContentType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContentType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ct = ContentType(s)
	return nil
}

func (ct ContentType) Value() (driver.Value, error) {
	return string(ct), nil
}
