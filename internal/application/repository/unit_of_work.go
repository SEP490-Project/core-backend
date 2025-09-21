package repository

import (
	"core-backend/internal/domain/model"
	"gorm.io/gorm"
)

type UnitOfWork interface {
	Begin() UnitOfWork
	Commit() error
	Rollback() error

	// Expose repos trong transaction
	Products() GenericRepository[model.Product]
	Users() GenericRepository[model.User]

	DB() *gorm.DB
}
