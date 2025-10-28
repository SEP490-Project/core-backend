package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// TagResponse represents the data transfer object for Tag model
type TagResponse struct {
	ID          string  `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	Name        string  `json:"name" example:"Technology"`
	Description *string `json:"description,omitempty" example:"Posts related to the latest technology trends."`
	UsageCount  int     `json:"usage_count" example:"42"`
	CreatedAt   string  `json:"created_at" example:"2024-01-01T12:00:00Z"`
	CreatedByID *string `json:"created_by,omitempty" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-02T12:00:00Z"`
	UpdatedByID *string `json:"updated_by,omitempty" example:"p5o4n3m2-l1k0-j9i8-h7g6-f5e4d3c2b1a0"`
}

// ToResponse converts a Tag model to a TagResponse DTO
func (TagResponse) ToResponse(model *model.Tag) *TagResponse {
	response := &TagResponse{
		ID:          model.ID.String(),
		Name:        model.Name,
		Description: model.Description,
		UsageCount:  model.UsageCount,
		CreatedAt:   utils.FormatLocalTime(&model.CreatedAt, utils.TimeFormat),
		UpdatedAt:   utils.FormatLocalTime(&model.UpdatedAt, utils.TimeFormat),
	}
	if model.CreatedByID != nil {
		response.CreatedByID = utils.PtrOrNil(model.CreatedByID.String())
	}
	if model.UpdatedByID != nil {
		response.UpdatedByID = utils.PtrOrNil(model.UpdatedByID.String())
	}
	return response
}

// ToListResponse converts a list of Tag models to a list of TagResponse DTOs
func (TagResponse) ToListResponse(model []model.Tag) []TagResponse {
	if len(model) == 0 {
		return []TagResponse{}
	}

	responses := make([]TagResponse, len(model))
	for i, v := range model {
		responses[i] = *TagResponse{}.ToResponse(&v)
	}
	return responses
}

// TagPaginationResponse represents a paginated response for TagResponse DTOs
// It is used only for Swagger documentation purposes
type TagPaginationResponse PaginationResponse[TagResponse]
