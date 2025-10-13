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

	productRepo                irepository.GenericRepository[model.Product]
	userRepo                   irepository.GenericRepository[model.User]
	brandRepo                  irepository.GenericRepository[model.Brand]
	loggedSessionRepo          irepository.GenericRepository[model.LoggedSession]
	ContractRepository         irepository.GenericRepository[model.Contract]
	ContractPaymentRepository  irepository.GenericRepository[model.ContractPayment]
	CampaignRepository         irepository.GenericRepository[model.Campaign]
	MilestoneRepository        irepository.GenericRepository[model.Milestone]
	TaskRepository             irepository.GenericRepository[model.Task]
	ProductStoryRepository     irepository.GenericRepository[model.ProductStory]
	ProductVariantRepository   irepository.GenericRepository[model.ProductVariant]
	VariantAttributeRepository irepository.GenericRepository[model.VariantAttribute]
	variantImageRepository     irepository.GenericRepository[model.VariantImage]
	variantAttributeRepository irepository.GenericRepository[model.VariantAttribute]
}

func (u *unitOfWork) VariantImage() irepository.GenericRepository[model.VariantImage] {
	return u.variantImageRepository
}

func (u *unitOfWork) ProductStory() irepository.GenericRepository[model.ProductStory] {
	return u.ProductStoryRepository
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
	u.ContractRepository = gormrepository.NewGenericRepository[model.Contract](u.tx)
	u.ContractPaymentRepository = gormrepository.NewGenericRepository[model.ContractPayment](u.tx)
	u.CampaignRepository = gormrepository.NewGenericRepository[model.Campaign](u.tx)
	u.MilestoneRepository = gormrepository.NewGenericRepository[model.Milestone](u.tx)
	u.TaskRepository = gormrepository.NewGenericRepository[model.Task](u.tx)
	u.ProductStoryRepository = gormrepository.NewGenericRepository[model.ProductStory](u.tx)
	u.ProductVariantRepository = gormrepository.NewGenericRepository[model.ProductVariant](u.tx)
	u.VariantAttributeRepository = gormrepository.NewGenericRepository[model.VariantAttribute](u.tx)
	u.variantImageRepository = gormrepository.NewGenericRepository[model.VariantImage](u.tx)
	u.variantAttributeRepository = gormrepository.NewGenericRepository[model.VariantAttribute](u.tx)

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
	return u.ContractRepository
}

func (u *unitOfWork) ContractPayments() irepository.GenericRepository[model.ContractPayment] {
	return u.ContractPaymentRepository
}

func (u *unitOfWork) Campaigns() irepository.GenericRepository[model.Campaign] {
	return u.CampaignRepository
}

func (u *unitOfWork) Milestones() irepository.GenericRepository[model.Milestone] {
	return u.MilestoneRepository
}

func (u *unitOfWork) Tasks() irepository.GenericRepository[model.Task] {
	return u.TaskRepository
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
