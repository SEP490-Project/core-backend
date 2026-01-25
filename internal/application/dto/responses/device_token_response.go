package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// DeviceTokenResponse represents a single device token response
type DeviceTokenResponse struct {
	ID         uuid.UUID         `json:"id"`
	UserID     uuid.UUID         `json:"user_id"`
	Token      string            `json:"token"`
	Platform   enum.PlatformType `json:"platform"`
	IsValid    bool              `json:"is_valid"`
	LastUsedAt *time.Time        `json:"last_used_at,omitempty"`
	CreatedAt  *time.Time        `json:"created_at,omitempty"`
}

// DeviceTokenListResponse represents a list of device tokens
type DeviceTokenListResponse struct {
	Tokens []DeviceTokenResponse `json:"tokens"`
	Total  int                   `json:"total"`
}

// ToDeviceTokenResponse converts a DeviceToken model to a response DTO
func ToDeviceTokenResponse(token *model.DeviceToken) *DeviceTokenResponse {
	if token == nil {
		return nil
	}

	return &DeviceTokenResponse{
		ID:         token.ID,
		UserID:     token.UserID,
		Token:      token.Token,
		Platform:   token.Platform,
		IsValid:    token.IsValid,
		LastUsedAt: token.LastUsedAt,
		CreatedAt:  token.CreatedAt,
	}
}

// ToDeviceTokenListResponse converts a slice of DeviceToken models to a list response DTO
func ToDeviceTokenListResponse(tokens []model.DeviceToken) *DeviceTokenListResponse {
	responses := make([]DeviceTokenResponse, 0, len(tokens))
	for _, token := range tokens {
		responses = append(responses, *ToDeviceTokenResponse(&token))
	}

	return &DeviceTokenListResponse{
		Tokens: responses,
		Total:  len(responses),
	}
}
