package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
)

// CreateProductDTO is the refined DTO for creating a Product entity.
// Use this when you want a strongly typed enum based request instead of raw strings.
// It intentionally omits server managed fields (ID, Status, CreatedAt, UpdatedAt, CreatedByID, UpdatedByID).
// Status will default in service / repository layer (e.g. DRAFT) and auditing fields are injected there.
// Variants are not included here; they should be created via their own endpoint or extended DTO if needed.
type CreateProductDTO struct {
	BrandID      uuid.UUID        `json:"brand_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID   uuid.UUID        `json:"category_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string           `json:"name" validate:"required,min=1,max=255" example:"Product Name"`
	Description  *string          `json:"description" validate:"omitempty,max=1000" example:"Product description"`
	Price        float64          `json:"price" validate:"required,gte=0" example:"99.99"`
	Type         enum.ProductType `json:"type" validate:"required,oneof=STANDARD LIMITED" example:"STANDARD"`
	CurrentStock *int             `json:"current_stock" validate:"omitempty,gte=0" example:"100"`
}

// ToModel maps the CreateProductDTO to a domain model Product.
// createdBy is injected from the authenticated context (current user id).
func (dto *CreateProductDTO) ToModel(createdBy uuid.UUID) *model.Product {
	return &model.Product{
		BrandID:      dto.BrandID,
		CategoryID:   dto.CategoryID,
		Name:         dto.Name,
		Description:  dto.Description,
		Price:        dto.Price,
		Type:         dto.Type,
		CurrentStock: dto.CurrentStock,
		CreatedByID:  createdBy,
	}
}
