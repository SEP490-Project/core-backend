package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContentStatus string

const (
	ContentStatusDraft      ContentStatus = "DRAFT"
	ContentStatusAwaitStaff ContentStatus = "AWAIT_STAFF"
	ContentStatusAwaitBrand ContentStatus = "AWAIT_BRAND"
	ContentStatusRejected   ContentStatus = "REJECTED"
	ContentStatusApproved   ContentStatus = "APPROVED"
	ContentStatusPosted     ContentStatus = "POSTED"
)

func (cs ContentStatus) IsValid() bool {
	switch cs {
	case ContentStatusDraft, ContentStatusAwaitStaff, ContentStatusAwaitBrand, ContentStatusRejected, ContentStatusApproved, ContentStatusPosted:
		return true
	}
	return false
}

func (cs *ContentStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContentStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cs = ContentStatus(s)
	return nil
}

func (cs ContentStatus) Value() (driver.Value, error) {
	return string(cs), nil
}

func (cs ContentStatus) String() string { return string(cs) }
