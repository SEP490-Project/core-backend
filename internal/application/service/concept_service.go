package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type conceptService struct {
	conceptRepo irepository.GenericRepository[model.Concept]
}

// GetConceptPagination retrieves paginated concepts with optional search and status (ATTACHED | DANGLING | NIL) filtering.
func (c conceptService) GetConceptPagination(limit, page int, search string, status *string) ([]model.Concept, int, error) {
	ctx := context.Background()

	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where("name ILIKE ?", "%"+search+"%")
		}

		if status != nil {
			switch *status {
			case "ATTACHED":
				db = db.Where(`
				EXISTS (
					SELECT 1
					FROM limited_products lp
					WHERE lp.concept_id = concepts.id
				)
			`)
			case "DANGLING":
				db = db.Where(`
				NOT EXISTS (
					SELECT 1
					FROM limited_products lp
					WHERE lp.concept_id = concepts.id
				)
			`)
			case "ALL":
				// không filter
			}
		} else {
			//nil default to DANGLING
			db = db.Where(`
				NOT EXISTS (
					SELECT 1
					FROM limited_products lp
					WHERE lp.concept_id = concepts.id
				)
			`)
		}

		return db.Order("concepts.created_at DESC")
	}

	includes := []string{"LimitedProduct"}

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	concepts, total, err := c.conceptRepo.GetAll(ctx, filter, includes, limit, page)
	if err != nil {
		return nil, 0, err
	}
	return concepts, int(total), nil
}

func (c conceptService) CreateConcept(dto requests.ConceptRequest) (*model.Concept, error) {
	if dto.Name == "" {
		return nil, errors.New("name is required")
	}
	ctx := context.Background()
	entity := dto.ToModel()
	// persist
	if err := c.conceptRepo.Add(ctx, entity); err != nil {
		zap.L().Error("failed to create concept", zap.Error(err))
		return nil, err
	}
	return entity, nil
}

func (c conceptService) UpdateConcept(id uuid.UUID, dto requests.UpdateConceptRequest) (*model.Concept, error) {
	// Get concept by ID
	concept, err := c.conceptRepo.GetByID(context.Background(), id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get concept: %w", err)
	}
	if concept == nil {
		return nil, errors.New("concept not found")
	}

	// Update fields if provided
	if dto.Name != nil {
		concept.Name = *dto.Name
	}
	if dto.Description != nil {
		concept.Description = dto.Description
	}
	if dto.BannerURL != nil {
		concept.BannerURL = dto.BannerURL
	}
	if dto.VideoThumbnail != nil {
		concept.VideoThumbnail = dto.VideoThumbnail
	}

	// Save updates
	if err := c.conceptRepo.Update(context.Background(), concept); err != nil {
		return nil, fmt.Errorf("failed to update concept: %w", err)
	}

	return concept, nil
}

func (c conceptService) DeleteConcept(conceptID string) error {
	if conceptID == "" {
		return errors.New("concept id is required")
	}
	id, err := uuid.Parse(conceptID)
	if err != nil {
		return fmt.Errorf("invalid concept id: %w", err)
	}
	ctx := context.Background()
	// check existence
	exists, err := c.conceptRepo.ExistsByID(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("concept not found")
	}
	// delete (soft-delete if model supports)
	if err := c.conceptRepo.DeleteByID(ctx, id); err != nil {
		return err
	}
	return nil
}

func NewConceptService(repo irepository.GenericRepository[model.Concept]) iservice.ConceptService {
	return &conceptService{conceptRepo: repo}
}
