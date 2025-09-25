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

// GetByCondition gets an entity from the database based on the given filter and includes.
func (r *genericRepository[T]) GetByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB, includes []string, noTracking bool) (*T, error) {
	var item T

	query := r.db.WithContext(ctx).Model(new(T))

	if noTracking {
		query = query.Session(&gorm.Session{NewDB: true}) // giống AsNoTracking
	}

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

// Delete deletes an entity from the database.
func (r *genericRepository[T]) Delete(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Delete(entity).Error
}
