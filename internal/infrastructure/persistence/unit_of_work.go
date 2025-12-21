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

	userRepo                        irepository.UserRepository
	shippingAddressRepo             irepository.GenericRepository[model.ShippingAddress]
	brandRepo                       irepository.GenericRepository[model.Brand]
	loggedSessionRepo               irepository.GenericRepository[model.LoggedSession]
	productRepo                     irepository.GenericRepository[model.Product]
	contractRepository              irepository.GenericRepository[model.Contract]
	contractPaymentRepository       irepository.GenericRepository[model.ContractPayment]
	campaignRepository              irepository.GenericRepository[model.Campaign]
	milestoneRepository             irepository.GenericRepository[model.Milestone]
	taskRepository                  irepository.TaskRepository
	productStoryRepository          irepository.GenericRepository[model.ProductStory]
	productVariantRepository        irepository.GenericRepository[model.ProductVariant]
	variantAttributeRepository      irepository.GenericRepository[model.VariantAttribute]
	variantImageRepository          irepository.GenericRepository[model.VariantImage]
	variantAttributeValueRepository irepository.GenericRepository[model.VariantAttributeValue]
	modifiedHistoryRepository       irepository.GenericRepository[model.ModifiedHistory]
	channelRepository               irepository.GenericRepository[model.Channel]
	contentRepository               irepository.GenericRepository[model.Content]
	contentChannelRepository        irepository.ContentChannelsRepository
	scheduleRepository              irepository.ScheduleRepository
	blogRepository                  irepository.GenericRepository[model.Blog]
	tagRepository                   irepository.TagRepository
	webhookRepository               irepository.GenericRepository[model.WebhookData]

	//ProductCategory
	productCategoryRepository irepository.GenericRepository[model.ProductCategory]

	//Concept & LimitedProduct
	limitProductRepository irepository.GenericRepository[model.LimitedProduct]
	conceptRepository      irepository.GenericRepository[model.Concept]
	configRepository       irepository.GenericRepository[model.Config]

	//Orders & Payment
	orderRepository              irepository.OrderRepository
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.PaymentTransactionRepository
	preOrderRepository           irepository.PreOrderRepository
	productReviewRepository      irepository.GenericRepository[model.ProductReview]

	//Notifications
	notificationRepository irepository.NotificationRepository
	deviceTokenRepository  irepository.DeviceTokenRepository

	//Affiliate Link Tracking
	affiliateLinkRepository irepository.AffiliateLinkRepository
	clickEventRepository    irepository.ClickEventRepository
	kpiMetricsRepository    irepository.GenericRepository[model.KPIMetrics]
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

	txUow.productRepo = gormrepository.NewGenericRepository[model.Product](tx)
	txUow.userRepo = gormrepository.NewUserRepository(tx)
	txUow.shippingAddressRepo = gormrepository.NewGenericRepository[model.ShippingAddress](tx)
	txUow.brandRepo = gormrepository.NewGenericRepository[model.Brand](tx)
	txUow.loggedSessionRepo = gormrepository.NewGenericRepository[model.LoggedSession](tx)
	txUow.contractRepository = gormrepository.NewGenericRepository[model.Contract](tx)
	txUow.contractPaymentRepository = gormrepository.NewGenericRepository[model.ContractPayment](tx)
	txUow.campaignRepository = gormrepository.NewGenericRepository[model.Campaign](tx)
	txUow.milestoneRepository = gormrepository.NewGenericRepository[model.Milestone](tx)
	txUow.taskRepository = gormrepository.NewTaskRepository(tx)
	txUow.channelRepository = gormrepository.NewGenericRepository[model.Channel](tx)
	txUow.contentRepository = gormrepository.NewGenericRepository[model.Content](tx)
	txUow.contentChannelRepository = gormrepository.NewContentChannelsRepository(tx)
	txUow.scheduleRepository = gormrepository.NewScheduleRepository(tx)
	txUow.blogRepository = gormrepository.NewGenericRepository[model.Blog](tx)
	txUow.tagRepository = gormrepository.NewTagRepository(tx)
	txUow.webhookRepository = gormrepository.NewGenericRepository[model.WebhookData](tx)

	//Product flow
	txUow.productStoryRepository = gormrepository.NewGenericRepository[model.ProductStory](tx)
	txUow.productVariantRepository = gormrepository.NewGenericRepository[model.ProductVariant](tx)
	txUow.variantAttributeRepository = gormrepository.NewGenericRepository[model.VariantAttribute](tx)
	txUow.variantImageRepository = gormrepository.NewGenericRepository[model.VariantImage](tx)
	txUow.variantAttributeValueRepository = gormrepository.NewGenericRepository[model.VariantAttributeValue](tx)
	txUow.modifiedHistoryRepository = gormrepository.NewGenericRepository[model.ModifiedHistory](tx)
	txUow.configRepository = gormrepository.NewGenericRepository[model.Config](tx)
	txUow.productCategoryRepository = gormrepository.NewGenericRepository[model.ProductCategory](tx)

	//Concept & LimitProduct
	txUow.limitProductRepository = gormrepository.NewGenericRepository[model.LimitedProduct](tx)
	txUow.conceptRepository = gormrepository.NewGenericRepository[model.Concept](tx)

	//Orders & Payment
	txUow.orderRepository = gormrepository.NewOrderRepository(tx)
	txUow.orderItemRepository = gormrepository.NewGenericRepository[model.OrderItem](tx)
	txUow.paymentTransactionRepository = gormrepository.NewPaymentTransactionRepository(tx)
	// Initialize PreOrder repository
	txUow.preOrderRepository = gormrepository.NewPreOrderRepository(tx)

	//Notifications
	txUow.notificationRepository = gormrepository.NewNotificationRepository(tx)
	txUow.deviceTokenRepository = gormrepository.NewDeviceTokenRepository(tx)

	//Affiliate Link Tracking
	txUow.affiliateLinkRepository = gormrepository.NewAffiliateLinkRepository(tx)
	txUow.clickEventRepository = gormrepository.NewClickEventRepository(tx)
	txUow.kpiMetricsRepository = gormrepository.NewGenericRepository[model.KPIMetrics](tx)
	txUow.productReviewRepository = gormrepository.NewGenericRepository[model.ProductReview](tx)

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

func (u *unitOfWork) Users() irepository.UserRepository {
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

func (u *unitOfWork) Tasks() irepository.TaskRepository {
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

func (u *unitOfWork) Order() irepository.OrderRepository {
	return u.orderRepository
}

func (u *unitOfWork) OrderItem() irepository.GenericRepository[model.OrderItem] {
	return u.orderItemRepository
}

func (u *unitOfWork) PaymentTransaction() irepository.PaymentTransactionRepository {
	return u.paymentTransactionRepository
}

func (u *unitOfWork) Channels() irepository.GenericRepository[model.Channel] {
	return u.channelRepository
}

func (u *unitOfWork) Contents() irepository.GenericRepository[model.Content] {
	return u.contentRepository
}

func (u *unitOfWork) ContentChannels() irepository.ContentChannelsRepository {
	return u.contentChannelRepository
}

func (u *unitOfWork) Blogs() irepository.GenericRepository[model.Blog] {
	return u.blogRepository
}

func (u *unitOfWork) Notifications() irepository.NotificationRepository {
	return u.notificationRepository
}

func (u *unitOfWork) DeviceTokens() irepository.DeviceTokenRepository {
	return u.deviceTokenRepository
}

func (u *unitOfWork) Tags() irepository.TagRepository {
	return u.tagRepository
}

func (u *unitOfWork) AffiliateLinks() irepository.AffiliateLinkRepository {
	return u.affiliateLinkRepository
}

func (u *unitOfWork) ClickEvents() irepository.ClickEventRepository {
	return u.clickEventRepository
}

func (u *unitOfWork) KPIMetrics() irepository.GenericRepository[model.KPIMetrics] {
	return u.kpiMetricsRepository
}

func (u *unitOfWork) PreOrder() irepository.PreOrderRepository {
	return u.preOrderRepository
}

func (u *unitOfWork) WebhookData() irepository.GenericRepository[model.WebhookData] {
	return u.webhookRepository
}

func (u *unitOfWork) ProductReview() irepository.GenericRepository[model.ProductReview] {
	return u.productReviewRepository
}

func (u *unitOfWork) Schedules() irepository.ScheduleRepository {
	return u.scheduleRepository
}

func (u *unitOfWork) DB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
