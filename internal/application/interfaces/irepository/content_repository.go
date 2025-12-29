package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContentRepository interface {
	GenericRepository[model.Content]

	GetContentByIDWIthDetails(ctx context.Context, id uuid.UUID) (*model.Content, error)
}
