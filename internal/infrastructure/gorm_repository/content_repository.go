package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contentRepository struct {
	*genericRepository[model.Content]
}

// GetContentByIDWIthDetails implements [irepository.ContentRepository].
func (c *contentRepository) GetContentByIDWIthDetails(ctx context.Context, id uuid.UUID) (*model.Content, error) {
	var content model.Content
	if err := c.db.WithContext(ctx).Model(&model.Content{}).
		Preload("ContentChannels.AffiliateLink").
		Preload("Task").
		Where("id = ?", id).
		First(&content).Error; err != nil {
		return nil, err
	}
	return &content, nil
}

func NewContentRepository(db *gorm.DB) irepository.ContentRepository {
	return &contentRepository{genericRepository: &genericRepository[model.Content]{db: db}}
}
