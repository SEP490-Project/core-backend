package gorm_repository

import (
	"core-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	UserRepository          repository.UserRepository
	LoggedSessionRepository repository.LoggedSessionRepository
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		UserRepository:          newUserRepository(db),
		LoggedSessionRepository: newLoggedSessionRepository(db),
	}
}
