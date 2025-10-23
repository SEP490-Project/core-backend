package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// ChannelResponse represents channel information in responses
type ChannelResponse struct {
	ID          string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" example:"Facebook"`
	Description *string `json:"description,omitempty" example:"This is a social media channel."`
	HomePageURL *string `json:"home_page_url,omitempty" example:"https://www.facebook.com"`
	IsActive    bool    `json:"is_active" example:"true"`
	CreatedAt   string  `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// ToResponse converts a model.Channel to a ChannelResponse
func (ChannelResponse) ToResponse(model *model.Channel) *ChannelResponse {
	return &ChannelResponse{
		ID:          model.ID.String(),
		Name:        model.Name,
		Description: model.Description,
		HomePageURL: model.HomePageURL,
		IsActive:    model.IsActive,
		CreatedAt:   utils.FormatLocalTime(&model.CreatedAt, ""),
		UpdatedAt:   utils.FormatLocalTime(&model.UpdatedAt, ""),
	}
}

// ToListResponse converts a list of model.Channel to a list of ChannelResponse
func (ChannelResponse) ToListResponse(models []model.Channel) (responses []ChannelResponse) {
	if len(models) == 0 {
		return []ChannelResponse{}
	}

	for _, model := range models {
		responses = append(responses, *(ChannelResponse{}.ToResponse(&model)))
	}
	return
}

// ChannelListResponse represents a paginated response for channels.
// Only used for Swaggo swagger docs generation
type ChannelListResponse PaginationResponse[ChannelResponse]
