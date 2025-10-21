package persistence

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type unitOfWork struct {
	db *gorm.DB
	tx *gorm.DB

	userRepo                        irepository.GenericRepository[model.User]
	shippingAddressRepo             irepository.GenericRepository[model.ShippingAddress]
	brandRepo                       irepository.GenericRepository[model.Brand]
	loggedSessionRepo               irepository.GenericRepository[model.LoggedSession]
	TaskRepository                  irepository.GenericRepository[model.Task]
	productRepo                     irepository.GenericRepository[model.Product]
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
	modifiedHistoryRepository       irepository.GenericRepository[model.ModifiedHistory]

	//ProductCategory
	productCategoryRepository irepository.GenericRepository[model.ProductCategory]

	//Concept & LimitedProduct
	limitProductRepository irepository.GenericRepository[model.LimitedProduct]
	conceptRepository      irepository.GenericRepository[model.Concept]
	configRepository                irepository.GenericRepository[model.Config]
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
	u.shippingAddressRepo = gormrepository.NewGenericRepository[model.ShippingAddress](u.tx)
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
	u.modifiedHistoryRepository = gormrepository.NewGenericRepository[model.ModifiedHistory](u.tx)
	u.configRepository = gormrepository.NewGenericRepository[model.Config](u.tx)
	u.productCategoryRepository = gormrepository.NewGenericRepository[model.ProductCategory](u.tx)

	//Concept & LimitProduct
	u.limitProductRepository = gormrepository.NewGenericRepository[model.LimitedProduct](u.tx)
	u.conceptRepository = gormrepository.NewGenericRepository[model.Concept](u.tx)

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

	u.tx = nil
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

	u.tx = nil
	return err
}

func (u *unitOfWork) Products() irepository.GenericRepository[model.Product] {
	return u.productRepo
}

func (u *unitOfWork) Users() irepository.GenericRepository[model.User] {
	return u.userRepo
}

func (u *unitOfWork) ShippingAddresses() irepository.GenericRepository[model.ShippingAddress] {
	return u.shippingAddressRepo
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

func (u *unitOfWork) ModifiedHistories() irepository.GenericRepository[model.ModifiedHistory] {
	return u.modifiedHistoryRepository
}

func (u *unitOfWork) AdminConfigs() irepository.GenericRepository[model.Config] {
	return u.configRepository
}

func (u *unitOfWork) ProductCategory() irepository.GenericRepository[model.ProductCategory] {
	return u.productCategoryRepository
}

func (u *unitOfWork) Concepts() irepository.GenericRepository[model.Concept] {
	return u.conceptRepository
}

func (u *unitOfWork) LimitedProducts() irepository.GenericRepository[model.LimitedProduct] {
	return u.limitProductRepository
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
