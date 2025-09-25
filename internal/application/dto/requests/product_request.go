package requests

import "github.com/google/uuid"

// ProductListRequest represents product list query parameters
type ProductListRequest struct {
	PaginationRequest
	Search *string `form:"search" json:"search" validate:"omitempty,max=255" example:"laptop"`
	Type   *string `form:"type" json:"type" validate:"omitempty,oneof=STANDARD LIMITED" example:"STANDARD"`
}

// CreateProductRequest represents create product request
type CreateProductRequest struct {
	BrandID     uuid.UUID `json:"brand_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name" validate:"required,min=1,max=255" example:"Product Name"`
	Description *string   `json:"description" validate:"omitempty,max=1000" example:"Product description"`
	Price       float64   `json:"price" validate:"required,min=0" example:"99.99"`
	Type        string    `json:"type" validate:"required,oneof=STANDARD LIMITED" example:"STANDARD"`
}

// UpdateProductRequest represents update product request
type UpdateProductRequest struct {
	Name        string  `json:"name" validate:"omitempty,min=1,max=255" example:"Updated Product Name"`
	Description *string `json:"description" validate:"omitempty,max=1000" example:"Updated product description"`
	Price       float64 `json:"price" validate:"omitempty,min=0" example:"149.99"`
}

