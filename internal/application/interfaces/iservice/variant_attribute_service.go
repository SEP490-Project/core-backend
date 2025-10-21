package iservice

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/model"
)

type VariantAttributeService interface {
	GetConceptPagination(limit, offset int, search string) ([]model.Concept, int, error)
	CreateConcept(dto requests.ConceptRequest) (*model.Concept, error)
	UpdateConcept(dto requests.ConceptRequest) (*model.Concept, error)
	DeleteConcept(conceptID string) error
}
