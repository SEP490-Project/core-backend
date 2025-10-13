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

	productRepo                     irepository.GenericRepository[model.Product]
	userRepo                        irepository.GenericRepository[model.User]
	brandRepo                       irepository.GenericRepository[model.Brand]
	loggedSessionRepo               irepository.GenericRepository[model.LoggedSession]
	contractRepository              irepository.GenericRepository[model.Contract]
	contractPaymentRepository       irepository.GenericRepository[model.ContractPayment]
	campaignRepository              irepository.GenericRepository[model.Campaign]
	milestoneRepository             irepository.GenericRepository[model.Milestone]
	taskRepository                  irepository.GenericRepository[model.Task]
	productStoryRepository          irepository.GenericRepository[model.ProductStory]
	productVariantRepository        irepository.GenericRepository[model.ProductVariant]
	variantAttributeRepository      irepository.GenericRepository[model.VariantAttribute]
	variantImageRepository          irepository.GenericRepository[model.VariantImage]
	variantAttributeValueRepository irepository.GenericRepository[model.VariantAttributeValue]
}

func (u *unitOfWork) ProductVariant() irepository.GenericRepository[model.ProductVariant] {
	return u.productVariantRepository
}

func (u *unitOfWork) VariantAttributes() irepository.GenericRepository[model.VariantAttribute] {
	return u.variantAttributeRepository
}

func (u *unitOfWork) VariantAttributeValue() irepository.GenericRepository[model.VariantAttributeValue] {
	return u.variantAttributeValueRepository
}

func (u *unitOfWork) VariantImage() irepository.GenericRepository[model.VariantImage] {
	return u.variantImageRepository
}

func (u *unitOfWork) ProductStory() irepository.GenericRepository[model.ProductStory] {
	return u.productStoryRepository
}

func (u *unitOfWork) InTransaction() bool {
	return u.tx != nil
}

func NewUnitOfWork(db *gorm.DB) irepository.UnitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) Begin() irepository.UnitOfWork {
	if u.tx != nil {
		zap.L().Warn("Transaction already started")
		return u
	}
	zap.L().Debug("Beginning database transaction")

	u.tx = u.db.Begin()
	if u.tx.Error != nil {
		zap.L().Error("Failed to begin database transaction", zap.Error(u.tx.Error))
		return u
	}

	u.productRepo = gormrepository.NewGenericRepository[model.Product](u.tx)
	u.userRepo = gormrepository.NewGenericRepository[model.User](u.tx)
	u.brandRepo = gormrepository.NewGenericRepository[model.Brand](u.tx)
	u.loggedSessionRepo = gormrepository.NewGenericRepository[model.LoggedSession](u.tx)
	u.contractRepository = gormrepository.NewGenericRepository[model.Contract](u.tx)
	u.contractPaymentRepository = gormrepository.NewGenericRepository[model.ContractPayment](u.tx)
	u.campaignRepository = gormrepository.NewGenericRepository[model.Campaign](u.tx)
	u.milestoneRepository = gormrepository.NewGenericRepository[model.Milestone](u.tx)
	u.taskRepository = gormrepository.NewGenericRepository[model.Task](u.tx)
	//Product flow
	u.productStoryRepository = gormrepository.NewGenericRepository[model.ProductStory](u.tx)
	u.productVariantRepository = gormrepository.NewGenericRepository[model.ProductVariant](u.tx)
	u.variantAttributeRepository = gormrepository.NewGenericRepository[model.VariantAttribute](u.tx)
	u.variantImageRepository = gormrepository.NewGenericRepository[model.VariantImage](u.tx)
	u.variantAttributeValueRepository = gormrepository.NewGenericRepository[model.VariantAttributeValue](u.tx)

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

func (u *unitOfWork) Brands() irepository.GenericRepository[model.Brand] {
	return u.brandRepo
}

func (u *unitOfWork) LoggedSessions() irepository.GenericRepository[model.LoggedSession] {
	return u.loggedSessionRepo
}

func (u *unitOfWork) Contracts() irepository.GenericRepository[model.Contract] {
	return u.contractRepository
}

func (u *unitOfWork) ContractPayments() irepository.GenericRepository[model.ContractPayment] {
	return u.contractPaymentRepository
}

func (u *unitOfWork) Campaigns() irepository.GenericRepository[model.Campaign] {
	return u.campaignRepository
}

func (u *unitOfWork) Milestones() irepository.GenericRepository[model.Milestone] {
	return u.milestoneRepository
}

func (u *unitOfWork) Tasks() irepository.GenericRepository[model.Task] {
	return u.taskRepository
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
