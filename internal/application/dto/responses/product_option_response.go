package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// ProductOptionResponse represents the response DTO for a product option
type ProductOptionResponse struct {
	ID          string  `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	Type        string  `json:"type" example:"CAPACITY_UNIT"`
	Code        string  `json:"code" example:"ML"`
	Name        string  `json:"name" example:"Milliliter"`
	Description *string `json:"description,omitempty" example:"Volume measurement in milliliters"`
	SortOrder   int     `json:"sort_order" example:"1"`
	IsActive    bool    `json:"is_active" example:"true"`
	CreatedAt   string  `json:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-02T12:00:00Z"`
}

// ToResponse converts a ProductOption model to ProductOptionResponse DTO
func (ProductOptionResponse) ToResponse(m *model.ProductOption) *ProductOptionResponse {
	if m == nil {
		return nil
	}
	return &ProductOptionResponse{
		ID:          m.ID.String(),
		Type:        string(m.Type),
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		SortOrder:   m.SortOrder,
		IsActive:    m.IsActive,
		CreatedAt:   utils.FormatLocalTime(&m.CreatedAt, utils.TimeFormat),
		UpdatedAt:   utils.FormatLocalTime(&m.UpdatedAt, utils.TimeFormat),
	}
}

// ToListResponse converts a list of ProductOption models to a list of ProductOptionResponse DTOs
func (ProductOptionResponse) ToListResponse(models []model.ProductOption) []ProductOptionResponse {
	if len(models) == 0 {
		return []ProductOptionResponse{}
	}

	responses := make([]ProductOptionResponse, len(models))
	for i, m := range models {
		responses[i] = *ProductOptionResponse{}.ToResponse(&m)
	}
	return responses
}

// ProductOptionPaginationResponse represents a paginated response for ProductOptionResponse DTOs
// It is used only for Swagger documentation purposes
type ProductOptionPaginationResponse PaginationResponse[ProductOptionResponse]
