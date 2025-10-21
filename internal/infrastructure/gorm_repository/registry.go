// Package gormrepository provides GORM-based implementations of repositories.
package gormrepository

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
)

type DatabaseRegistry struct {
	UserRepository             irepository.GenericRepository[model.User]
	LoggedSessionRepository    irepository.GenericRepository[model.LoggedSession]
	ProductRepository          irepository.GenericRepository[model.Product]
	ProductVariantRepository   irepository.GenericRepository[model.ProductVariant]
	BrandRepository            irepository.GenericRepository[model.Brand]
	ProductCategoryRepository  irepository.GenericRepository[model.ProductCategory]
	ContractRepository         irepository.GenericRepository[model.Contract]
	ContractPaymentRepository  irepository.GenericRepository[model.ContractPayment]
	CampaignRepository         irepository.GenericRepository[model.Campaign]
	MilestoneRepository        irepository.GenericRepository[model.Milestone]
	TaskRepository             irepository.GenericRepository[model.Task]
	ModifiedHistoryRepository  irepository.GenericRepository[model.ModifiedHistory]
	VariantAttributeRepository irepository.GenericRepository[model.VariantAttribute]

	//Limited Product and Concept
	LimitedProductRepository irepository.GenericRepository[model.LimitedProduct]
	ConceptRepository        irepository.GenericRepository[model.Concept]
	AdminConfigRepository    irepository.GenericRepository[model.Config]
}

func NewDatabaseRegistry(db *gorm.DB) *DatabaseRegistry {
	return &DatabaseRegistry{
		UserRepository:             NewGenericRepository[model.User](db),
		LoggedSessionRepository:    NewGenericRepository[model.LoggedSession](db),
		ProductRepository:          NewGenericRepository[model.Product](db),
		ProductVariantRepository:   NewGenericRepository[model.ProductVariant](db),
		BrandRepository:            NewGenericRepository[model.Brand](db),
		ProductCategoryRepository:  NewGenericRepository[model.ProductCategory](db),
		ContractRepository:         NewGenericRepository[model.Contract](db),
		ContractPaymentRepository:  NewGenericRepository[model.ContractPayment](db),
		CampaignRepository:         NewGenericRepository[model.Campaign](db),
		MilestoneRepository:        NewGenericRepository[model.Milestone](db),
		TaskRepository:             NewGenericRepository[model.Task](db),
		ModifiedHistoryRepository:  NewGenericRepository[model.ModifiedHistory](db),
		LimitedProductRepository:   NewGenericRepository[model.LimitedProduct](db),
		ConceptRepository:          NewGenericRepository[model.Concept](db),
		AdminConfigRepository:      NewGenericRepository[model.Config](db),
		VariantAttributeRepository: NewGenericRepository[model.VariantAttribute](db),
	}
}
