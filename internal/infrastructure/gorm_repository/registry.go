package gorm_repository

import (
	"core-backend/internal/application/repository"
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	UserRepository          repository.UserRepository
	LoggedSessionRepository repository.LoggedSessionRepository
	ProductRepository       repository.GenericRepository[model.Product]
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		UserRepository:          newUserRepository(db),
		LoggedSessionRepository: newLoggedSessionRepository(db),
		ProductRepository:       NewGenericRepository[model.Product](db),
	}
}
