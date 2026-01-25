package responses

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type ProductCategoryResponse struct {
	ID              uuid.UUID                  `json:"id"`
	Name            string                     `json:"name"`
	Description     *string                    `json:"description,omitempty"`
	ParentCategory  *ProductCategoryResponse   `json:"parent_category,omitempty"`
	ChildCategories []*ProductCategoryResponse `json:"child_categories,omitempty"`
	CreatedAt       *time.Time                 `json:"create_at,omitempty"`
	UpdatedAt       *time.Time                 `json:"update_at,omitempty"`
}

func (p *ProductCategoryResponse) ToModelResponse(c *model.ProductCategory) *ProductCategoryResponse {
	if c == nil {
		return nil
	}

	return &ProductCategoryResponse{
		ID:             c.ID,
		Name:           c.Name,
		Description:    c.Description,
		CreatedAt:      &c.CreatedAt,
		UpdatedAt:      &c.UpdatedAt,
		ParentCategory: p.ToModelResponse(c.ParentCategory),
		ChildCategories: func() []*ProductCategoryResponse {
			if len(c.ChildCategories) == 0 {
				return nil
			}
			childResponses := make([]*ProductCategoryResponse, 0, len(c.ChildCategories))
			for _, child := range c.ChildCategories {
				childResponse := p.ToModelResponse(&child)
				childResponses = append(childResponses, childResponse)
			}
			return childResponses
		}(),
	}
}
