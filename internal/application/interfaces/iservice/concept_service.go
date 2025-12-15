package iservice

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
)

type ConceptService interface {
	GetConceptPagination(limit, page int, search string, status *string) ([]model.Concept, int, error)
	CreateConcept(dto requests.ConceptRequest) (*model.Concept, error)
	UpdateConcept(id uuid.UUID, dto requests.UpdateConceptRequest) (*model.Concept, error)
	DeleteConcept(conceptID string) error
}
