// Package irepository defines the Repository interface for CRUD operations on entities.
package irepository

import (
	"context"

	"gorm.io/gorm"
)

type GenericRepository[T any] interface {
	// GetAll retrieves all entities from the database based on the given filter, includes, and pagination parameters.
	// If pageSize is less than or equal to 0, all records will be returned without pagination.
	// If pageSize is greater than 100, it will be capped at 100.
	GetAll(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, pageSize, pageNumber int) ([]T, int64, error)
	GetByID(ctx context.Context, id any, includes []string) (*T, error)
	GetByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string) (*T, error)
	Count(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (int64, error)
	Exists(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (bool, error)
	ExistsByID(ctx context.Context, id any) (bool, error)
	Add(ctx context.Context, entity *T) error

	// BulkAdd adds multiple entities to the database in batches of the specified size.
	// If batchSize is less than or equal to 100, the operations will be performed using the default batch size defined in config.
	BulkAdd(ctx context.Context, entities []*T, batchSize int) (rowsAffected int64, err error)
	Update(ctx context.Context, entity *T) error
	UpdateByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, updates map[string]any) error
	Delete(ctx context.Context, entity *T) error
	DeleteByID(ctx context.Context, id any) error
}
