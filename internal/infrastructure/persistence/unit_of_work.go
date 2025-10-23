package persistence

import (
	"context"
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
	channelRepository               irepository.GenericRepository[model.Channel]
	contentRepository               irepository.GenericRepository[model.Content]
	contentChannelRepository        irepository.GenericRepository[model.ContentChannel]
	blogRepository                  irepository.GenericRepository[model.Blog]

	//ProductCategory
	productCategoryRepository irepository.GenericRepository[model.ProductCategory]

	//Concept & LimitedProduct
	limitProductRepository irepository.GenericRepository[model.LimitedProduct]
	conceptRepository      irepository.GenericRepository[model.Concept]
	configRepository       irepository.GenericRepository[model.Config]

	//Orders & Payment
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
}

func NewUnitOfWork(db *gorm.DB) irepository.UnitOfWork {
	return &unitOfWork{db: db}
}

// Begin create a new instance of UnitOfWork with a new transaction
// and retain the reference to the original DB connection
func (u *unitOfWork) Begin(ctx context.Context) irepository.UnitOfWork {
	// if u.tx != nil {
	// 	zap.L().Warn("Transaction already started")
	// 	return u
	// }
	zap.L().Debug("Beginning database transaction")

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		zap.L().Error("Failed to begin database transaction", zap.Error(u.tx.Error))
		return u
	}

	txUow := &unitOfWork{
		db: u.db,
		tx: tx,
	}

	txUow.productRepo = gormrepository.NewGenericRepository[model.Product](txUow.tx)
	txUow.userRepo = gormrepository.NewGenericRepository[model.User](txUow.tx)
	txUow.shippingAddressRepo = gormrepository.NewGenericRepository[model.ShippingAddress](txUow.tx)
	txUow.brandRepo = gormrepository.NewGenericRepository[model.Brand](txUow.tx)
	txUow.loggedSessionRepo = gormrepository.NewGenericRepository[model.LoggedSession](txUow.tx)
	txUow.contractRepository = gormrepository.NewGenericRepository[model.Contract](txUow.tx)
	txUow.contractPaymentRepository = gormrepository.NewGenericRepository[model.ContractPayment](txUow.tx)
	txUow.campaignRepository = gormrepository.NewGenericRepository[model.Campaign](txUow.tx)
	txUow.milestoneRepository = gormrepository.NewGenericRepository[model.Milestone](txUow.tx)
	txUow.taskRepository = gormrepository.NewGenericRepository[model.Task](txUow.tx)
	txUow.channelRepository = gormrepository.NewGenericRepository[model.Channel](txUow.tx)
	txUow.contentRepository = gormrepository.NewGenericRepository[model.Content](txUow.tx)
	txUow.contentChannelRepository = gormrepository.NewGenericRepository[model.ContentChannel](txUow.tx)
	txUow.blogRepository = gormrepository.NewGenericRepository[model.Blog](txUow.tx)

	//Product flow
	txUow.productStoryRepository = gormrepository.NewGenericRepository[model.ProductStory](txUow.tx)
	txUow.productVariantRepository = gormrepository.NewGenericRepository[model.ProductVariant](txUow.tx)
	txUow.variantAttributeRepository = gormrepository.NewGenericRepository[model.VariantAttribute](txUow.tx)
	txUow.variantImageRepository = gormrepository.NewGenericRepository[model.VariantImage](txUow.tx)
	txUow.variantAttributeValueRepository = gormrepository.NewGenericRepository[model.VariantAttributeValue](txUow.tx)
	txUow.modifiedHistoryRepository = gormrepository.NewGenericRepository[model.ModifiedHistory](txUow.tx)
	txUow.configRepository = gormrepository.NewGenericRepository[model.Config](txUow.tx)
	txUow.productCategoryRepository = gormrepository.NewGenericRepository[model.ProductCategory](txUow.tx)

	//Concept & LimitProduct
	txUow.limitProductRepository = gormrepository.NewGenericRepository[model.LimitedProduct](txUow.tx)
	txUow.conceptRepository = gormrepository.NewGenericRepository[model.Concept](txUow.tx)

	//Orders & Payment
	u.orderRepository = gormrepository.NewGenericRepository[model.Order](u.tx)
	u.orderItemRepository = gormrepository.NewGenericRepository[model.OrderItem](u.tx)
	u.paymentTransactionRepository = gormrepository.NewGenericRepository[model.PaymentTransaction](u.tx)

	zap.L().Debug("Database transaction started successfully")
	return txUow
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

func (u *unitOfWork) Order() irepository.GenericRepository[model.Order] {
	return u.orderRepository
}

func (u *unitOfWork) OrderItem() irepository.GenericRepository[model.OrderItem] {
	return u.orderItemRepository
}

func (u *unitOfWork) PaymentTransaction() irepository.GenericRepository[model.PaymentTransaction] {
	return u.paymentTransactionRepository
}

func (u *unitOfWork) Channels() irepository.GenericRepository[model.Channel] {
	return u.channelRepository
}

func (u *unitOfWork) Contents() irepository.GenericRepository[model.Content] {
	return u.contentRepository
}

func (u *unitOfWork) ContentChannels() irepository.GenericRepository[model.ContentChannel] {
	return u.contentChannelRepository
}

func (u *unitOfWork) Blogs() irepository.GenericRepository[model.Blog] {
	return u.blogRepository
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
