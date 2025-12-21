package asynqtask

import (
	"time"

	"github.com/google/uuid"
)

// ContentScheduleTaskPayload is the payload for content scheduling tasks
type ContentScheduleTaskPayload struct {
	ScheduleID       uuid.UUID `json:"schedule_id"`
	ContentChannelID uuid.UUID `json:"content_channel_id"`
	ContentID        uuid.UUID `json:"content_id"`
	ChannelCode      string    `json:"channel_code"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	RetryCount       int       `json:"retry_count,omitempty"`
}
