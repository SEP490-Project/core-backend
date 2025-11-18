package responses

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// PublishContentResponse represents the response after publishing content to a single channel
type PublishContentResponse struct {
	ContentChannelID uuid.UUID `json:"content_channel_id"`
	ExternalPostID   string    `json:"external_post_id"`
	PostURL          string    `json:"post_url"`
	PublishedAt      time.Time `json:"published_at"`
	Channel          string    `json:"channel"` // Channel name (e.g., "Facebook Page", "TikTok Account")
}

// PublishAllChannelsResponse represents the response after publishing to all channels
type PublishAllChannelsResponse struct {
	TotalChannels int                      `json:"total_channels"`
	SuccessCount  int                      `json:"success_count"`
	FailureCount  int                      `json:"failure_count"`
	Results       []PublishContentResponse `json:"results"`
	Errors        []PublishChannelError    `json:"errors,omitempty"`
}

// PublishChannelError represents an error for a specific channel
type PublishChannelError struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	Error       string    `json:"error"`
}

// PublishingStatusResponse represents the current publishing status for a content-channel
type PublishingStatusResponse struct {
	ContentChannelID uuid.UUID           `json:"content_channel_id"`
	ContentID        uuid.UUID           `json:"content_id"`
	ChannelID        uuid.UUID           `json:"channel_id"`
	ChannelName      string              `json:"channel_name"`
	Status           enum.AutoPostStatus `json:"status"`
	ExternalPostID   *string             `json:"external_post_id,omitempty"`
	ExternalPostURL  *string             `json:"external_post_url,omitempty"`
	PostURL          *string             `json:"post_url,omitempty"`
	PublishedAt      *time.Time          `json:"published_at,omitempty"`
	LastError        *string             `json:"last_error,omitempty"`
	Metrics          map[string]any      `json:"metrics,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}
