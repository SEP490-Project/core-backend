package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateProductCategoryRequest struct {
	Name             string     `json:"name" validate:"required,min=2,max=100"`
	Description      *string    `json:"description,omitempty"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id,omitempty"`
}

type UpdateProductCategoryRequest struct {
	Name             *string    `json:"name"`
	Description      *string    `json:"description"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id"`
}

// ToModel map request to model
func (c *CreateProductCategoryRequest) ToModel() *model.ProductCategory {
	now := time.Now()
	return &model.ProductCategory{
		Name:             c.Name,
		Description:      c.Description,
		ParentCategoryID: c.ParentCategoryID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
