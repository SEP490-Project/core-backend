package irepository

import (
	"context"
	"core-backend/internal/domain/model"
)

type TagRepository interface {
	GenericRepository[model.Tag]
	CreateIfNotExists(ctx context.Context, tags []model.Tag) ([]model.Tag, error)
}
