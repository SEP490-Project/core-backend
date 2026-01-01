package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
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

// GetContentIDsByCampaignID returns all content IDs associated with tasks in a campaign.
// This method bypasses the 100-record limit of GenericRepository.GetAll by using
// direct SQL with JOINs instead of fetching full entities.
func (c *contentRepository) GetContentIDsByCampaignID(ctx context.Context, campaignID uuid.UUID, excludeStatus ...enum.ContentStatus) ([]uuid.UUID, error) {
	var contentIDs []uuid.UUID
	query := c.db.WithContext(ctx).
		Model(&model.Content{}).
		Select("contents.id").
		Joins("JOIN tasks ON tasks.id = contents.task_id").
		Joins("JOIN milestones ON milestones.id = tasks.milestone_id").
		Where("milestones.campaign_id = ?", campaignID).
		Where("contents.deleted_at IS NULL")

	if len(excludeStatus) > 0 {
		query = query.Where("contents.status NOT IN (?)", excludeStatus)
	}

	if err := query.Pluck("contents.id", &contentIDs).Error; err != nil {
		return nil, err
	}
	return contentIDs, nil
}

func NewContentRepository(db *gorm.DB) irepository.ContentRepository {
	return &contentRepository{genericRepository: &genericRepository[model.Content]{db: db}}
}
