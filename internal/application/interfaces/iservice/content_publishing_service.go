package iservice

import (
	"context"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ContentPublishingService handles publishing content to social media platforms
type ContentPublishingService interface {
	// PublishToChannel publishes content to a specific social media channel
	// Returns the external post ID, post URL, and any errors
	PublishToChannel(ctx context.Context, contentID uuid.UUID, channelID uuid.UUID, userID uuid.UUID) (*responses.PublishContentResponse, error)

	// PublishToAllChannels publishes content to all assigned channels in content_channels table
	// Returns aggregated results including success/failure counts
	PublishToAllChannels(ctx context.Context, contentID uuid.UUID, userID uuid.UUID) (*responses.PublishAllChannelsResponse, error)

	// GetPublishingStatus retrieves the current publishing status for a content-channel
	// Useful for checking async publishing progress
	GetPublishingStatus(ctx context.Context, contentChannelID uuid.UUID) (*responses.PublishingStatusResponse, error)

	// RetryPublish retries a failed publish attempt for a content-channel
	RetryPublish(ctx context.Context, contentChannelID uuid.UUID, userID uuid.UUID) error
}
