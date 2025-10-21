package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
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

func (c conceptService) GetConceptPagination(limit, page int, search string) ([]model.Concept, int, error) {
	ctx := context.Background()
	offset := (page - 1) * limit

	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where("name ILIKE ?", "%"+search+"%")
		}

		return db.Order("concepts.created_at DESC")
	}

	// Step 1: get IDs page
	var ids []uuid.UUID
	err := c.conceptRepo.DB().
		WithContext(ctx).
		Model(&model.Concept{}).
		Scopes(filter).
		Select("concepts.id").
		Limit(limit).
		Offset(offset).
		Pluck("concepts.id", &ids).Error
	if err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []model.Concept{}, 0, nil
	}

	// Step 2: total
	_, total, err := c.conceptRepo.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to count total products", zap.Error(err))
		return nil, 0, err
	}

	// Step 3: load concepts with limited product and nested product relations
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("concepts.id IN ?", ids).Order("concepts.created_at DESC")
	}

	concepts, _, err := c.conceptRepo.GetAll(ctx, finalFilter, nil, 0, 0)
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

func (c conceptService) UpdateConcept(dto requests.ConceptRequest) (*model.Concept, error) {
	// The request struct currently doesn't carry ID; require caller to provide ID via Name match is not safe.
	// Expectation: caller will load concept and pass fields to update. Here we'll return not implemented error to avoid accidental misuse.
	return nil, errors.New("UpdateConcept not implemented: request must include concept ID to update")
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

func NewConceptService(repo irepository.GenericRepository[model.Concept]) *conceptService {
	return &conceptService{conceptRepo: repo}
}
