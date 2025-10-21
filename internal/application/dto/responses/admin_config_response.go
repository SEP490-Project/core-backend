package responses

import "core-backend/internal/domain/model"

// AdminConfigResponse represents the response structure for an admin configuration
type AdminConfigResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Key         string  `json:"key" example:"site_name"`
	ValueType   string  `json:"value_type" example:"STRING"`
	Value       string  `json:"value" example:"My Awesome Site"`
	Description *string `json:"description" example:"The name of the site"`
	CreatedAt   int64   `json:"created_at" example:"2006-01-02 15:04:05"`
	UpdatedAt   int64   `json:"updated_at" example:"2006-01-02 15:04:05"`
	UpdatedByID string  `json:"updated_by" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ToResponse converts a model.Config to an AdminConfigResponse
func (AdminConfigResponse) ToResponse(model *model.Config) *AdminConfigResponse {
	return &AdminConfigResponse{
		ID:          model.ID.String(),
		Key:         model.Key,
		ValueType:   string(model.ValueType),
		Value:       model.Value,
		Description: model.Description,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		UpdatedByID: model.UpdatedByID.String(),
	}
}

// ToResponseList converts a list of model.Config to a list of AdminConfigResponse
func (AdminConfigResponse) ToResponseList(models []model.Config) []AdminConfigResponse {
	if models == nil {
		return []AdminConfigResponse{}
	}

	responses := make([]AdminConfigResponse, len(models))
	for i, model := range models {
		responses[i] = *AdminConfigResponse{}.ToResponse(&model)
	}
	return responses
}

// AdminConfigListResponse represents a paginated response for admin configurations.
// Only used for Swaggo swagger docs generation
type AdminConfigListResponse PaginationResponse[AdminConfigResponse]
