package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	var rawMetrics []datatypes.JSON
	if err := r.db.WithContext(ctx).
		Model(&model.ContentChannel{}).
		Where("content_id IN ?", contentIDs).
		Find(&rawMetrics).Error; err != nil {
		return nil, err
	}

	return utils.MapSlice(rawMetrics, func(raw datatypes.JSON) model.ContentChannelMetrics {
		var metrics model.ContentChannelMetrics
		if err := json.Unmarshal(raw, &metrics); err != nil {
			return model.ContentChannelMetrics{}
		}
		return metrics
	}), nil
}
