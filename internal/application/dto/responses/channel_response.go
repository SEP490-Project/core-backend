package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// ChannelResponse represents channel information in responses
type ChannelResponse struct {
	ID          string            `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Code        string            `json:"code" example:"FACEBOOK"`
	Name        string            `json:"name" example:"Facebook"`
	Description *string           `json:"description,omitempty" example:"This is a social media channel."`
	HomePageURL *string           `json:"home_page_url,omitempty" example:"https://www.facebook.com"`
	IsActive    bool              `json:"is_active" example:"true"`
	CreatedAt   string            `json:"created_at" example:"2023-01-01 00:00:00"`
	UpdatedAt   string            `json:"updated_at" example:"2023-01-01 00:00:00"`
	TokenInfo   *ChannelTokenInfo `json:"token_info"`
}

type ChannelTokenInfo struct {
	ExternalID            string  `json:"external_id" example:"9876543210"`
	AccountName           string  `json:"account_name" example:"my_facebook_account"`
	AccessTokenExpiresAt  string  `json:"access_token_expires_at" example:"2023-12-31 23:59:59"`
	RefreshTokenExpiresAt *string `json:"refresh_token_expires_at,omitempty" example:"2024-12-31 23:59:59"`
	LastSyncedAt          string  `json:"last_synced_at" example:"2023-06-01 12:00:00"`
}

// ToResponse converts a model.Channel to a ChannelResponse
func (ChannelResponse) ToResponse(model *model.Channel) *ChannelResponse {
	return &ChannelResponse{
		ID:          model.ID.String(),
		Name:        model.Name,
		Code:        model.Code,
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
