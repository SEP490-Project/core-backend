package responses

import (
	"time"

	"github.com/google/uuid"
)

// ScheduleResponse represents the response after scheduling content
type ScheduleResponse struct {
	ScheduleID       uuid.UUID `json:"schedule_id"`
	ContentChannelID uuid.UUID `json:"content_channel_id"`
	ContentID        uuid.UUID `json:"content_id"`
	ContentTitle     string    `json:"content_title"`
	ChannelName      string    `json:"channel_name"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
}

// ScheduleListResponse represents a list of schedules
type ScheduleListResponse struct {
	Schedules []ScheduleItemResponse `json:"schedules"`
	Total     int64                  `json:"total"`
}

// ScheduleItemResponse represents a single schedule item in a list
type ScheduleItemResponse struct {
	ScheduleID       uuid.UUID  `json:"schedule_id"`
	ContentChannelID uuid.UUID  `json:"content_channel_id"`
	ContentID        uuid.UUID  `json:"content_id"`
	ContentTitle     string     `json:"content_title"`
	ContentType      string     `json:"content_type"` // "POST", "VIDEO"
	ChannelID        uuid.UUID  `json:"channel_id"`
	ChannelName      string     `json:"channel_name"`
	ChannelCode      string     `json:"channel_code"` // "WEBSITE", "FACEBOOK", "TIKTOK"
	ScheduledAt      time.Time  `json:"scheduled_at"`
	Status           string     `json:"status"`
	RetryCount       int        `json:"retry_count"`
	LastError        *string    `json:"last_error,omitempty"`
	ExecutedAt       *time.Time `json:"executed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatedBy        string     `json:"created_by"`
	CreatedByID      uuid.UUID  `json:"created_by_id"`
}

// BatchScheduleResponse represents the response after batch scheduling content
type BatchScheduleResponse struct {
	ContentID         uuid.UUID                  `json:"content_id"`
	ContentTitle      string                     `json:"content_title"`
	TotalScheduled    int                        `json:"total_scheduled"`
	TotalFailed       int                        `json:"total_failed"`
	ScheduledChannels []BatchScheduleResultItem  `json:"scheduled_channels"`
	FailedChannels    []BatchScheduleFailureItem `json:"failed_channels,omitempty"`
}

// BatchScheduleResultItem represents a successfully scheduled channel
type BatchScheduleResultItem struct {
	ScheduleID  uuid.UUID `json:"schedule_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	ChannelCode string    `json:"channel_code"`
	ScheduledAt time.Time `json:"scheduled_at"`
	AutoPost    bool      `json:"auto_post"`
}

// BatchScheduleFailureItem represents a failed schedule attempt
type BatchScheduleFailureItem struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	Error       string    `json:"error"`
}

// SchedulePaginationResponse represents a paginated response for schedules.
type SchedulePaginationResponse PaginationResponse[ScheduleItemResponse]
