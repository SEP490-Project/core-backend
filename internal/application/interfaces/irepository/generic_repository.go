// Package irepository defines the Repository interface for CRUD operations on entities.
package irepository

import (
	"context"
	"gorm.io/gorm"
)

type GenericRepository[T any] interface {
	GetAll(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, pageSize, pageNumber int) ([]T, int64, error)
	GetByID(ctx context.Context, id any, includes []string) (*T, error)
	GetByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string) (*T, error)
	Count(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (int64, error)
	Exists(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (bool, error)
	Add(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	UpdateByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, updates map[string]any) error
	Delete(ctx context.Context, entity *T) error
	DeleteByID(ctx context.Context, id any) error
}
