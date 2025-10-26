package requests

import "core-backend/internal/domain/enum"

// RegisterDeviceTokenRequest represents a request to register a device token
type RegisterDeviceTokenRequest struct {
	Token    string            `json:"token" validate:"required,min=10"`
	Platform enum.PlatformType `json:"platform" validate:"required,oneof=IOS ANDROID"`
}

// UpdateDeviceTokenRequest represents a request to update a device token
type UpdateDeviceTokenRequest struct {
	NewToken string            `json:"new_token" validate:"required,min=10"`
	Platform enum.PlatformType `json:"platform" validate:"required,oneof=IOS ANDROID"`
}
