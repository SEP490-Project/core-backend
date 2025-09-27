// Package gormrepository provides GORM-based implementations of repositories.
package gormrepository

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	UserRepository          irepository.UserRepository
	LoggedSessionRepository irepository.LoggedSessionRepository
	ProductRepository       irepository.GenericRepository[model.Product]
	BrandRepository         irepository.GenericRepository[model.Brand]
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		UserRepository:          newUserRepository(db),
		LoggedSessionRepository: newLoggedSessionRepository(db),
		ProductRepository:       NewGenericRepository[model.Product](db),
		BrandRepository:         NewGenericRepository[model.Brand](db),
	}
}
