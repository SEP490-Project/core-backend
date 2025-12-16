package consumers

import (
	"time"

	"github.com/google/uuid"
)

// ContentScheduleMessage represents the message sent to RabbitMQ for scheduled content publishing
// This message is published with x-delay header for delayed delivery
type ContentScheduleMessage struct {
	ScheduleID       uuid.UUID `json:"schedule_id" validate:"required"`
	ContentChannelID uuid.UUID `json:"content_channel_id" validate:"required"`
	ContentID        uuid.UUID `json:"content_id" validate:"required"`
	ChannelCode      string    `json:"channel_code" validate:"required"` // "WEBSITE", "FACEBOOK", "TIKTOK"
	ScheduledAt      time.Time `json:"scheduled_at" validate:"required"`
	RetryCount       int       `json:"retry_count,omitempty"` // Current retry attempt number
}
