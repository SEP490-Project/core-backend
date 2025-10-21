package irepository

import (
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
)

type UnitOfWork interface {
	Begin() UnitOfWork
	Commit() error
	Rollback() error
	InTransaction() bool

	// Expose repos trong transaction
	Products() GenericRepository[model.Product]
	Users() GenericRepository[model.User]
	ShippingAddresses() GenericRepository[model.ShippingAddress]
	Brands() GenericRepository[model.Brand]
	LoggedSessions() GenericRepository[model.LoggedSession]
	Contracts() GenericRepository[model.Contract]
	ContractPayments() GenericRepository[model.ContractPayment]
	Campaigns() GenericRepository[model.Campaign]
	Milestones() GenericRepository[model.Milestone]
	Tasks() GenericRepository[model.Task]
	//Product flow
	ProductStory() GenericRepository[model.ProductStory]
	ProductVariant() GenericRepository[model.ProductVariant]
	VariantAttributes() GenericRepository[model.VariantAttribute]
	VariantImage() GenericRepository[model.VariantImage]
	VariantAttributeValue() GenericRepository[model.VariantAttributeValue]
	ModifiedHistories() GenericRepository[model.ModifiedHistory]
	AdminConfigs() GenericRepository[model.Config]

	// Product Category
	ProductCategory() GenericRepository[model.ProductCategory]

	//Concepts
	Concepts() GenericRepository[model.Concept]
	LimitedProducts() GenericRepository[model.LimitedProduct]

	// DB Get raw gorm instance
	DB() *gorm.DB
}
