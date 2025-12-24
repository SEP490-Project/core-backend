package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contentChannelsRepository struct {
	*genericRepository[model.ContentChannel]
}

func NewContentChannelsRepository(db *gorm.DB) irepository.ContentChannelsRepository {
	return &contentChannelsRepository{
		genericRepository: &genericRepository[model.ContentChannel]{db: db},
	}
}

func (r *contentChannelsRepository) GetMetricsByContentIDs(ctx context.Context, contentIDs []uuid.UUID) ([]model.ContentChannelMetrics, error) {
	type result struct {
		Metrics model.ContentChannelMetrics `gorm:"type:jsonb;column:metrics"`
	}
	var results []result
	if err := r.db.WithContext(ctx).
		Model(&model.ContentChannel{}).
		Where("content_id IN ?", contentIDs).
		Where("metrics IS NOT NULL").
		Select("metrics").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	return utils.MapSlice(results, func(r result) model.ContentChannelMetrics { return r.Metrics }), nil
}
