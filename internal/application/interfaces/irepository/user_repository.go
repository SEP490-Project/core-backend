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
	GetContractIDsByUserBrandID(ctx context.Context, userbrandID uuid.UUID) ([]uuid.UUID, error)
	GetUserFullnameByID(ctx context.Context, userID uuid.UUID) (string, error)
}
