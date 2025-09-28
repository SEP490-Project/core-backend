package irepository

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
	Brands() GenericRepository[model.Brand]
	LoggedSessions() GenericRepository[model.LoggedSession]

	DB() *gorm.DB
}
