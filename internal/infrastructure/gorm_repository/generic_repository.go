package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"

	"gorm.io/gorm"
)

// genericRepository implementation
type genericRepository[T any] struct {
	db *gorm.DB
}

// NewGenericRepository creates a new instance of genericRepository for the specified entity type T.
func NewGenericRepository[T any](db *gorm.DB) irepository.GenericRepository[T] {
	return &genericRepository[T]{db: db}
}

// GetAll retrieves all entities from the database based on the given filter, includes, and pagination parameters.
func (r *genericRepository[T]) GetAll(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, pageSize, pageNumber int) ([]T, int64, error) {
	var items []T
	var total int64

	query := r.db.WithContext(ctx).Model(new(T))

	// filter
	if filter != nil {
		query = filter(query)
	}

	// count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// include (Preload)
	for _, inc := range includes {
		query = query.Preload(inc)
	}

	// paging
	if pageSize > 0 {
		if pageSize > 100 {
			pageSize = 100
		}
		query = query.Offset((pageNumber - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID gets an entity from the database by its ID, with optional includes and no-tracking.
func (r *genericRepository[T]) GetByID(ctx context.Context, id any, includes []string) (*T, error) {
	var item T
	query := r.db.WithContext(ctx).Model(new(T))
	// include (Preload)
	for _, inc := range includes {
		query = query.Preload(inc)
	}
	if err := query.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByCondition gets an entity from the database based on the given filter and includes.
func (r *genericRepository[T]) GetByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string) (*T, error) {
	var item T

	query := r.db.WithContext(ctx).Model(new(T))

	if filter != nil {
		query = filter(query)
	}

	for _, inc := range includes {
		query = query.Preload(inc)
	}

	if err := query.First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

// Add adds a new entity to the database.
func (r *genericRepository[T]) Add(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update updates an existing entity in the database.
func (r *genericRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Updates(entity).Error
}

// UpdateByCondition updates entities in the database that match the given filter with the provided updates.
// The updates parameter is a map where keys are column names and values are the new values to set.
func (r *genericRepository[T]) UpdateByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, updates map[string]any) error {
	query := r.db.WithContext(ctx).Model(new(T))
	if filter != nil {
		query = filter(query)
	}
	return query.Updates(updates).Error
}

// Delete deletes an entity from the database.
func (r *genericRepository[T]) Delete(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Delete(entity).Error
}

func (r *genericRepository[T]) DeleteByID(ctx context.Context, id any) error {
	return r.db.WithContext(ctx).Delete(new(T), id).Error
}

// Exists checks if any entity exists in the database that matches the given filter.
func (r *genericRepository[T]) Exists(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(new(T))
	if filter != nil {
		query = filter(query)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count counts the number of entities in the database that match the given filter.
func (r *genericRepository[T]) Count(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(new(T))
	if filter != nil {
		query = filter(query)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// func (r *genericRepository[T]) Count *gorm.DB {
