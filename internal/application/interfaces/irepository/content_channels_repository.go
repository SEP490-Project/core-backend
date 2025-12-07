package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContentChannelsRepository interface {
	GenericRepository[model.ContentChannel]
	GetMetricsByContentIDs(ctx context.Context, contentIDs []uuid.UUID) ([]model.ContentChannelMetrics, error)
}
