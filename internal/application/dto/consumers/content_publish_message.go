package consumers

import "github.com/google/uuid"

// PublishContentMessage represents the message structure for publishing content to a single channel
// sent through RabbitMQ queue.content.publish
type PublishContentMessage struct {
	ContentID uuid.UUID `json:"content_id" validate:"required"`
	ChannelID uuid.UUID `json:"channel_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	RequestID string    `json:"request_id,omitempty"` // Optional tracking ID for request tracing
}

// PublishAllChannelsMessage represents the message structure for publishing content to all channels
// sent through RabbitMQ queue.content.publish_all
type PublishAllChannelsMessage struct {
	ContentID uuid.UUID `json:"content_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	RequestID string    `json:"request_id,omitempty"` // Optional tracking ID for request tracing
}
