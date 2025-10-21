package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// AdminConfigResponse represents the response structure for an admin configuration
type AdminConfigResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Key         string  `json:"key" example:"site_name"`
	ValueType   string  `json:"value_type" example:"STRING"`
	Value       string  `json:"value" example:"My Awesome Site"`
	Description *string `json:"description" example:"The name of the site"`
	CreatedAt   string  `json:"created_at" example:"2006-01-02 15:04:05"`
	UpdatedAt   string  `json:"updated_at" example:"2006-01-02 15:04:05"`
	UpdatedByID string  `json:"updated_by,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ToResponse converts a model.Config to an AdminConfigResponse
func (r AdminConfigResponse) ToResponse(model model.Config) *AdminConfigResponse {
	response := &AdminConfigResponse{
		ID:          model.ID.String(),
		Key:         model.Key,
		ValueType:   model.ValueType.String(),
		Value:       model.Value,
		Description: model.Description,
		CreatedAt:   utils.FormatLocalTime(&model.CreatedAt, ""),
		UpdatedAt:   utils.FormatLocalTime(&model.UpdatedAt, ""),
	}

	if model.UpdatedByID != uuid.Nil {
		response.UpdatedByID = model.UpdatedByID.String()
	}

	return response
}

// ToResponseList converts a list of model.Config to a list of AdminConfigResponse
func (r AdminConfigResponse) ToResponseList(models []model.Config) []AdminConfigResponse {
	if models == nil {
		return []AdminConfigResponse{}
	}

	responses := make([]AdminConfigResponse, 0, len(models))
	for _, model := range models {
		responses = append(responses, *(AdminConfigResponse{}.ToResponse(model)))
	}
	return responses
}

// AdminConfigListResponse represents a paginated response for admin configurations.
// Only used for Swaggo swagger docs generation
type AdminConfigListResponse PaginationResponse[AdminConfigResponse]
