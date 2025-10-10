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
	ShippingAddresses() GenericRepository[model.ShippingAddress]
	Brands() GenericRepository[model.Brand]
	LoggedSessions() GenericRepository[model.LoggedSession]
	Contracts() GenericRepository[model.Contract]
	ContractPayments() GenericRepository[model.ContractPayment]
	Campaigns() GenericRepository[model.Campaign]
	Milestones() GenericRepository[model.Milestone]
	Tasks() GenericRepository[model.Task]

	DB() *gorm.DB
}
