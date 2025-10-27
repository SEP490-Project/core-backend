package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ProductListRequest represents product list query parameters
type ProductListRequest struct {
	PaginationRequest
	Search *string `form:"search" json:"search" validate:"omitempty,max=255" example:"laptop"`
	Type   *string `form:"type" json:"type" validate:"omitempty,oneof=STANDARD LIMITED" example:"STANDARD"`
}

/*===========================STANDARD PRODUCTS=====================================*/
// CreateStandardProductRequest represents create product request
type CreateStandardProductRequest struct {
	BrandID     uuid.UUID `json:"brand_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name" validate:"required,min=1,max=255" example:"Product Name"`
	Description *string   `json:"description" validate:"omitempty,max=1000" example:"Product description"`
}

// UpdateProductRequest represents update product request
type UpdateProductRequest struct {
	Name        string  `json:"name" validate:"omitempty,min=1,max=255" example:"Updated Product Name"`
	Description *string `json:"description" validate:"omitempty,max=1000" example:"Updated product description"`
}

func (d *CreateStandardProductRequest) ToStandardModel(createdBy uuid.UUID) *model.Product {
	if d == nil {
		return nil
	}
	return &model.Product{
		BrandID:     d.BrandID,
		CategoryID:  d.CategoryID,
		TaskID:      nil,
		Name:        d.Name,
		Description: d.Description,
		Type:        enum.ProductTypeStandard,
		CreatedByID: createdBy,
	}
}

/*===========================LIMITED PRODUCTS=====================================*/
type CreateLimitedProductRequest struct {
	BrandID          uuid.UUID                `json:"brand_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID       uuid.UUID                `json:"category_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskID           uuid.UUID                `json:"task_id" validate:"required,uuid" example:"660e8400-e29b-41d4-a716-446655440000"`
	Name             string                   `json:"name" validate:"required,min=1,max=255" example:"Product Name"`
	Description      *string                  `json:"description" validate:"omitempty,max=1000" example:"Product description"`
	LimitedAttribute LimitedProductAttributes `json:"limited_attribute" validate:"required"`
}

type LimitedProductAttributes struct {
	MaxStock              int        `json:"max_stock" validate:"required,min=0" example:"100"`
	IsFreeShipping        bool       `json:"is_free_shipping" example:"false"`
	BoughtLimit           int        `json:"bought_limit" validate:"required,min=1" example:"1"`
	PremiereDate          *string    `json:"premiere_date" validate:"required,datetime=2006-01-02T15:04:05Z07:00" example:"2023-10-01T10:00:00Z"`
	AvailabilityStartDate *string    `json:"availability_start_date" validate:"required,datetime=2006-01-02T15:04Z07:00" example:"2023-10-01T10:00+07:00"`
	AvailabilityEndDate   *string    `json:"availability_end_date" validate:"required,datetime=2006-01-02T15:04Z07:00" example:"2023-10-31T10:00+07:00"`
	ConceptID             *uuid.UUID `json:"concept_id" validate:"omitempty,uuid" example:"770e8400-e29b-41d4-a716-446655440000"`
}

func (l *LimitedProductAttributes) ToLimitedProductModel() *model.LimitedProduct {
	if l == nil {
		return nil
	}
	return &model.LimitedProduct{
		MaxStock:              l.MaxStock,
		IsFreeShipping:        l.IsFreeShipping,
		BoughtLimit:           l.BoughtLimit,
		PremiereDate:          parseTime(l.PremiereDate),
		AvailabilityStartDate: parseTime(l.AvailabilityStartDate),
		AvailabilityEndDate:   parseTime(l.AvailabilityEndDate),
		ConceptID:             l.ConceptID,
	}
}

func (d *CreateLimitedProductRequest) ToProductWithLimitedModel(createdBy uuid.UUID) *model.Product {
	if d == nil {
		return nil
	}
	return &model.Product{
		BrandID:     d.BrandID,
		CategoryID:  d.CategoryID,
		TaskID:      nil,
		Name:        d.Name,
		Description: d.Description,
		Type:        enum.ProductTypeLimited,
		CreatedByID: createdBy,
		Limited:     d.LimitedAttribute.ToLimitedProductModel(),
	}
}

func parseTime(date *string) time.Time {
	if date == nil {
		return time.Time{}
	}
	parsedTime, err := time.Parse(time.RFC3339, *date)
	if err != nil {
		return time.Time{}
	}
	return parsedTime
}
