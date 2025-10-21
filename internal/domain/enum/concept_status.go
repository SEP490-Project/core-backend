package enum

import "database/sql/driver"

// ConceptStatus represents lifecycle status for concept entity defined in DB type concept_status
type ConceptStatus string

const (
	ConceptStatusUnpublished ConceptStatus = "UNPUBLISHED"
	ConceptStatusDraft       ConceptStatus = "DRAFT"
	ConceptStatusPublished   ConceptStatus = "PUBLISHED"
)

// IsValid returns true when the status is one of allowed values
func (s ConceptStatus) IsValid() bool {
	switch s {
	case ConceptStatusUnpublished, ConceptStatusDraft, ConceptStatusPublished:
		return true
	default:
		return false
	}
}

func (s ConceptStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s ConceptStatus) String() string { return string(s) }
