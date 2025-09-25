// Package irepository defines the Repository interface for CRUD operations on entities.
package irepository

import (
	"context"
	"gorm.io/gorm"
)

type GenericRepository[T any] interface {
	GetAll(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, pageSize, pageNumber int) ([]T, int64, error)
	GetByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, noTracking bool) (*T, error)
	Add(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
}
