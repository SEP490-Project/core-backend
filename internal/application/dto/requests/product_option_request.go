package requests

import (
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ProductOptionFilterRequest represents filtering options for product options
type ProductOptionFilterRequest struct {
	Type       *string `form:"type" validate:"omitempty,oneof=CAPACITY_UNIT CONTAINER_TYPE DISPENSER_TYPE ATTRIBUTE_UNIT"`
	ActiveOnly *bool   `form:"active_only"`
	Page       int     `form:"page" validate:"omitempty,min=1"`
	Limit      int     `form:"limit" validate:"omitempty,min=1,max=100"`
}

// CreateProductOptionRequest represents the request body for creating a new product option
type CreateProductOptionRequest struct {
	Type        string  `json:"type" validate:"required,oneof=CAPACITY_UNIT CONTAINER_TYPE DISPENSER_TYPE ATTRIBUTE_UNIT" example:"CAPACITY_UNIT"`
	Code        string  `json:"code" validate:"required,min=1,max=50" example:"FL_OZ"`
	Name        string  `json:"name" validate:"required,min=1,max=100" example:"Fluid Ounce"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500" example:"US fluid ounce measurement"`
	SortOrder   *int    `json:"sort_order,omitempty" validate:"omitempty,min=0" example:"6"`
}

// ToModel converts CreateProductOptionRequest to ProductOption model
func (r *CreateProductOptionRequest) ToModel() *model.ProductOption {
	option := &model.ProductOption{
		ID:       uuid.New(),
		Type:     model.ProductOptionType(r.Type),
		Code:     r.Code,
		Name:     r.Name,
		IsActive: true,
	}

	if r.Description != nil {
		option.Description = r.Description
	}

	if r.SortOrder != nil {
		option.SortOrder = *r.SortOrder
	}

	return option
}

// UpdateProductOptionRequest represents the request body for updating a product option
type UpdateProductOptionRequest struct {
	Code        *string `json:"code,omitempty" validate:"omitempty,min=1,max=50" example:"FL_OZ"`
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100" example:"Fluid Ounce"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500" example:"US fluid ounce measurement"`
	SortOrder   *int    `json:"sort_order,omitempty" validate:"omitempty,min=0" example:"6"`
	IsActive    *bool   `json:"is_active,omitempty" example:"true"`
}

// ApplyToModel applies the update request to an existing ProductOption model
func (r *UpdateProductOptionRequest) ApplyToModel(option *model.ProductOption) {
	if r.Code != nil {
		option.Code = *r.Code
	}
	if r.Name != nil {
		option.Name = *r.Name
	}
	if r.Description != nil {
		option.Description = r.Description
	}
	if r.SortOrder != nil {
		option.SortOrder = *r.SortOrder
	}
	if r.IsActive != nil {
		option.IsActive = *r.IsActive
	}
}
