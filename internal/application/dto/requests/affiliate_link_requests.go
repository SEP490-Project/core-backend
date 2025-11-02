package requests

import "github.com/google/uuid"

// CreateAffiliateLinkRequest represents the request to create a new affiliate link
type CreateAffiliateLinkRequest struct {
	ContractID  uuid.UUID `json:"contract_id" validate:"required,uuid"`
	ContentID   uuid.UUID `json:"content_id" validate:"required,uuid"`
	ChannelID   uuid.UUID `json:"channel_id" validate:"required,uuid"`
	TrackingURL string    `json:"tracking_url" validate:"required,url,max=2048"`
}

// UpdateAffiliateLinkRequest represents the request to update an affiliate link
type UpdateAffiliateLinkRequest struct {
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive expired"`
	TrackingURL *string `json:"tracking_url,omitempty" validate:"omitempty,url,max=2048"`
}

// GetAffiliateLinkRequest represents query parameters for listing affiliate links
type GetAffiliateLinkRequest struct {
	ContractID *uuid.UUID `json:"contract_id,omitempty" form:"contract_id"`
	ContentID  *uuid.UUID `json:"content_id,omitempty" form:"content_id"`
	ChannelID  *uuid.UUID `json:"channel_id,omitempty" form:"channel_id"`
	Status     *string    `json:"status,omitempty" form:"status" validate:"omitempty,oneof=active inactive expired"`
	PageSize   int        `json:"page_size,omitempty" form:"page_size" validate:"omitempty,min=1,max=100"`
	PageNumber int        `json:"page_number,omitempty" form:"page_number" validate:"omitempty,min=1"`
}
