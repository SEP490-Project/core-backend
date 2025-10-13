package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
)

// CreateProductDTO internal service DTO. TaskID MUST be provided (business rule: product depends on a task).
type CreateProductDTO struct {
	BrandID     uuid.UUID
	CategoryID  uuid.UUID
	TaskID      *uuid.UUID // required (non-nil)
	Name        string
	Description *string
	Price       float64
	Type        string // STANDARD | LIMITED
}

// CreateProductVariantDTO carries variant data into the service layer.
type CreateProductVariantDTO struct {
	Price           float64
	CurrentStock    int
	Capacity        float64
	CapacityUnit    string // will be validated and cast to enum in service
	ContainerType   string // will be validated and cast to enum in service
	DispenserType   string // will be validated and cast to enum in service
	Uses            string
	ManufactureDate *string // RFC3339, parsed in service
	ExpiryDate      *string // RFC3339, parsed in service
	Instructions    string
	IsDefault       bool
}

// ToModel maps the DTO to a Product domain model.
func (d *CreateProductDTO) ToModel(createdBy uuid.UUID) *model.Product {
	if d == nil {
		return nil
	}
	return &model.Product{
		BrandID:     d.BrandID,
		CategoryID:  d.CategoryID,
		TaskID:      d.TaskID,
		Name:        d.Name,
		Description: d.Description,
		Price:       d.Price,
		Type:        enum.ProductType(d.Type),
		CreatedByID: createdBy,
	}
}
