package persistence

import (
	"core-backend/internal/application/repository"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/gorm_repository"
	"gorm.io/gorm"
)

type unitOfWork struct {
	db *gorm.DB
	tx *gorm.DB

	productRepo repository.GenericRepository[model.Product]
	userRepo    repository.GenericRepository[model.User]
}

func NewUnitOfWork(db *gorm.DB) repository.UnitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) Begin() repository.UnitOfWork {
	u.tx = u.db.Begin()

	u.productRepo = gorm_repository.NewGenericRepository[model.Product](u.tx)
	u.userRepo = gorm_repository.NewGenericRepository[model.User](u.tx)

	return u
}

func (u *unitOfWork) Commit() error {
	return u.tx.Commit().Error
}

func (u *unitOfWork) Rollback() error {
	return u.tx.Rollback().Error
}

func (u *unitOfWork) Products() repository.GenericRepository[model.Product] {
	return u.productRepo
}

func (u *unitOfWork) Users() repository.GenericRepository[model.User] {
	return u.userRepo
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
