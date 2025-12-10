// Package gormrepository provides GORM-based implementations of repositories.
package gormrepository

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	GormDatabase              *gorm.DB
	UserRepository            irepository.GenericRepository[model.User]
	LoggedSessionRepository   irepository.GenericRepository[model.LoggedSession]
	ProductRepository         irepository.GenericRepository[model.Product]
	ProductVariantRepository  irepository.GenericRepository[model.ProductVariant]
	BrandRepository           irepository.GenericRepository[model.Brand]
	ProductCategoryRepository irepository.GenericRepository[model.ProductCategory]
	ContractRepository        irepository.GenericRepository[model.Contract]
	ContractPaymentRepository irepository.GenericRepository[model.ContractPayment]
	CampaignRepository        irepository.GenericRepository[model.Campaign]
	MilestoneRepository       irepository.GenericRepository[model.Milestone]
	TaskRepository            irepository.TaskRepository
	ChannelRepository         irepository.GenericRepository[model.Channel]
	ContentRepository         irepository.GenericRepository[model.Content]
	ContentChannelRepository  irepository.ContentChannelsRepository
	BlogRepository            irepository.GenericRepository[model.Blog]
	ModifiedHistoryRepository irepository.GenericRepository[model.ModifiedHistory]
	AdminConfigRepository     irepository.GenericRepository[model.Config]
	TagRepository             irepository.TagRepository

	//Limited Product and Concept
	LimitedProductRepository   irepository.GenericRepository[model.LimitedProduct]
	ConceptRepository          irepository.GenericRepository[model.Concept]
	VariantAttributeRepository irepository.GenericRepository[model.VariantAttribute]

	//Orders & Payment
	OrderRepository              irepository.OrderRepository
	OrderItemRepository          irepository.GenericRepository[model.OrderItem]
	PaymentTransactionRepository irepository.PaymentTransactionRepository

	//PreOrders
	PreOrderRepository irepository.PreOrderRepository

	//Notifications
	NotificationRepository irepository.NotificationRepository
	DeviceTokenRepository  irepository.DeviceTokenRepository

	//Location
	ShippingAddressRepository irepository.GenericRepository[model.ShippingAddress]
	ProvinceRepository        irepository.GenericRepository[model.Province]
	DistrictRepository        irepository.GenericRepository[model.District]
	WardRepository            irepository.GenericRepository[model.Ward]

	//Affiliate Link Tracking
	AffiliateLinkRepository irepository.AffiliateLinkRepository
	ClickEventRepository    irepository.ClickEventRepository
	KPIMetricsRepository    irepository.GenericRepository[model.KPIMetrics]

	//Marketing Analytics
	MarketingAnalyticsRepository irepository.MarketingAnalyticsRepository

	//Contract Payment Calculation
	ContractPaymentCalculationRepository irepository.ContractPaymentCalculationRepository

	//Sales Staff Analytics
	SalesStaffAnalyticsRepository irepository.SalesStaffAnalyticsRepository

	//Content Staff Analytics
	ContentStaffAnalyticsRepository irepository.ContentStaffAnalyticsRepository

	//Brand Partner Analytics
	BrandPartnerAnalyticsRepository irepository.BrandPartnerAnalyticsRepository

	//Admin Analytics
	AdminAnalyticsRepository irepository.AdminAnalyticsRepository

	FileRepository irepository.GenericRepository[model.File]
	// Reviews
	ReviewRepository irepository.GenericRepository[model.ProductReview]
	// Webhooks
	WebhookDataRepository irepository.GenericRepository[model.WebhookData]
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		GormDatabase:                         db,
		UserRepository:                       NewGenericRepository[model.User](db),
		LoggedSessionRepository:              NewGenericRepository[model.LoggedSession](db),
		ProductRepository:                    NewGenericRepository[model.Product](db),
		ProductVariantRepository:             NewGenericRepository[model.ProductVariant](db),
		BrandRepository:                      NewGenericRepository[model.Brand](db),
		ProductCategoryRepository:            NewGenericRepository[model.ProductCategory](db),
		ContractRepository:                   NewGenericRepository[model.Contract](db),
		ContractPaymentRepository:            NewGenericRepository[model.ContractPayment](db),
		CampaignRepository:                   NewGenericRepository[model.Campaign](db),
		MilestoneRepository:                  NewGenericRepository[model.Milestone](db),
		TaskRepository:                       NewTaskRepository(db),
		ModifiedHistoryRepository:            NewGenericRepository[model.ModifiedHistory](db),
		LimitedProductRepository:             NewGenericRepository[model.LimitedProduct](db),
		ConceptRepository:                    NewGenericRepository[model.Concept](db),
		AdminConfigRepository:                NewGenericRepository[model.Config](db),
		VariantAttributeRepository:           NewGenericRepository[model.VariantAttribute](db),
		ChannelRepository:                    NewGenericRepository[model.Channel](db),
		ContentRepository:                    NewGenericRepository[model.Content](db),
		ContentChannelRepository:             NewContentChannelsRepository(db),
		BlogRepository:                       NewGenericRepository[model.Blog](db),
		TagRepository:                        NewTagRepository(db),
		OrderRepository:                      NewOrderRepository(db),
		OrderItemRepository:                  NewGenericRepository[model.OrderItem](db),
		PaymentTransactionRepository:         NewPaymentTransactionRepository(db),
		NotificationRepository:               NewNotificationRepository(db),
		DeviceTokenRepository:                NewDeviceTokenRepository(db),
		ShippingAddressRepository:            NewGenericRepository[model.ShippingAddress](db),
		ProvinceRepository:                   NewGenericRepository[model.Province](db),
		DistrictRepository:                   NewGenericRepository[model.District](db),
		WardRepository:                       NewGenericRepository[model.Ward](db),
		AffiliateLinkRepository:              NewAffiliateLinkRepository(db),
		ClickEventRepository:                 NewClickEventRepository(db),
		KPIMetricsRepository:                 NewGenericRepository[model.KPIMetrics](db),
		PreOrderRepository:                   NewPreOrderRepository(db),
		MarketingAnalyticsRepository:         NewMarketingAnalyticsRepository(db),
		ContractPaymentCalculationRepository: NewContractPaymentCalculationRepository(db),
		SalesStaffAnalyticsRepository:        NewSalesStaffAnalyticsRepository(db),
		ContentStaffAnalyticsRepository:      NewContentStaffAnalyticsRepository(db),
		BrandPartnerAnalyticsRepository:      NewBrandPartnerAnalyticsRepository(db),
		AdminAnalyticsRepository:             NewAdminAnalyticsRepository(db),
		FileRepository:                       NewGenericRepository[model.File](db),
		ReviewRepository:                     NewGenericRepository[model.ProductReview](db),
		WebhookDataRepository:                NewGenericRepository[model.WebhookData](db),
	}
}
