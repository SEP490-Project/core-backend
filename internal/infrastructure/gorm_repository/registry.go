// Package gormrepository provides GORM-based implementations of repositories.
package gormrepository

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	UserRepository          irepository.GenericRepository[model.User]
	LoggedSessionRepository irepository.GenericRepository[model.LoggedSession]
	ProductRepository       irepository.GenericRepository[model.Product]
	BrandRepository         irepository.GenericRepository[model.Brand]
	TaskRepository          irepository.GenericRepository[model.Task]
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		UserRepository:          NewGenericRepository[model.User](db),
		LoggedSessionRepository: NewGenericRepository[model.LoggedSession](db),
		ProductRepository:       NewGenericRepository[model.Product](db),
		BrandRepository:         NewGenericRepository[model.Brand](db),
		TaskRepository:          NewGenericRepository[model.Task](db),
	}
}
