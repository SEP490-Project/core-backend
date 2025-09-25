package persistence

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/gorm_repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type unitOfWork struct {
	db *gorm.DB
	tx *gorm.DB

	productRepo irepository.GenericRepository[model.Product]
	userRepo    irepository.GenericRepository[model.User]
}

func NewUnitOfWork(db *gorm.DB) irepository.UnitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) Begin() irepository.UnitOfWork {
	zap.L().Debug("Beginning database transaction")
	
	u.tx = u.db.Begin()
	if u.tx.Error != nil {
		zap.L().Error("Failed to begin database transaction", zap.Error(u.tx.Error))
		return u
	}

	u.productRepo = gormrepository.NewGenericRepository[model.Product](u.tx)
	u.userRepo = gormrepository.NewGenericRepository[model.User](u.tx)

	zap.L().Debug("Database transaction started successfully")
	return u
}

func (u *unitOfWork) Commit() error {
	zap.L().Debug("Committing database transaction")
	
	if u.tx == nil {
		zap.L().Warn("Attempted to commit nil transaction")
		return nil
	}
	
	err := u.tx.Commit().Error
	if err != nil {
		zap.L().Error("Failed to commit database transaction", zap.Error(err))
	} else {
		zap.L().Debug("Database transaction committed successfully")
	}
	
	return err
}

func (u *unitOfWork) Rollback() error {
	zap.L().Debug("Rolling back database transaction")
	
	if u.tx == nil {
		zap.L().Warn("Attempted to rollback nil transaction")
		return nil
	}
	
	err := u.tx.Rollback().Error
	if err != nil {
		zap.L().Error("Failed to rollback database transaction", zap.Error(err))
	} else {
		zap.L().Debug("Database transaction rolled back successfully")
	}
	
	return err
}

func (u *unitOfWork) Products() irepository.GenericRepository[model.Product] {
	return u.productRepo
}

func (u *unitOfWork) Users() irepository.GenericRepository[model.User] {
	return u.userRepo
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
