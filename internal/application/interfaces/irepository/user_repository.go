package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	GenericRepository[model.User]
	GetUserIDsByFilter(ctx context.Context, filter func(*gorm.DB) *gorm.DB) ([]uuid.UUID, error)
}
