package gorm_repository

import (
	"context"
	"core-backend/internal/application/repository"
	"gorm.io/gorm"
)

// genericRepository implementation
type genericRepository[T any] struct {
	db *gorm.DB
}

func NewGenericRepository[T any](db *gorm.DB) repository.GenericRepository[T] {
	return &genericRepository[T]{db: db}
}

// GetAll với filter + include + paging
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

// GetByCondition
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

// Add
func (r *genericRepository[T]) Add(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update
func (r *genericRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete
func (r *genericRepository[T]) Delete(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Delete(entity).Error
}
