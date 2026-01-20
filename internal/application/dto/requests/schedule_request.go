package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// ScheduleContentRequest represents a request to schedule content for publishing
type ScheduleContentRequest struct {
	ContentChannelID uuid.UUID `json:"content_channel_id" validate:"required,uuid"`
	ScheduledAt      string    `json:"scheduled_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	UserID           uuid.UUID `json:"-"` // Set from JWT context
}

// RescheduleContentRequest represents a request to reschedule content
type RescheduleContentRequest struct {
	ScheduledAt string    `json:"scheduled_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	UserID      uuid.UUID `json:"-"` // Set from JWT context
}

// CancelScheduleRequest represents a request to cancel a scheduled publish
type CancelScheduleRequest struct {
	Reason *string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// ScheduleFilterRequest represents filter parameters for listing schedules
type ScheduleFilterRequest struct {
	PaginationRequest
	Status        *enum.ScheduleStatus `form:"status" validate:"omitempty,oneof=PENDING PROCESSING COMPLETED FAILED CANCELLED"`
	ReferenceID   *uuid.UUID           `form:"reference_id" validate:"omitempty,uuid"`
	ReferenceType *enum.ReferenceType  `form:"reference_type" validate:"omitempty"`
	CreatedBy     *uuid.UUID           `form:"created_by" validate:"omitempty,uuid"` // Filter by creator (for role-based access)
	ContentID     *uuid.UUID           `form:"content_id" validate:"omitempty,uuid"` // Filter by content ID
	ChannelID     *uuid.UUID           `form:"channel_id" validate:"omitempty,uuid"` // Filter by channel ID
	FromDate      *string              `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate        *string              `form:"to_date" validate:"omitempty,datetime=2006-01-02"`
	Days          *int                 `form:"days" validate:"omitempty,min=1,max=90"` // Get schedules for next N days
}

// GetDaysOrDefault returns the days parameter with default of 7
func (r *ScheduleFilterRequest) GetDaysOrDefault() int {
	if r.Days != nil {
		return *r.Days
	}
	return 7
}

// BatchScheduleRequest for scheduling content to multiple channels at once
type BatchScheduleRequest struct {
	ContentID uuid.UUID           `json:"content_id" validate:"required,uuid"`
	Schedules []BatchScheduleItem `json:"schedules" validate:"required,min=1,dive"`
	UserID    uuid.UUID           `json:"-"` // Set from JWT context
}

// BatchScheduleItem represents a single schedule in a batch request
type BatchScheduleItem struct {
	ChannelID   string `json:"channel_id" validate:"required,uuid"`
	ScheduledAt string `json:"scheduled_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	AutoPost    bool   `json:"auto_post"` // For FB/TikTok auto-publishing
}

// BatchScheduleSameTimeRequest for scheduling content to multiple channels with the same time
type BatchScheduleSameTimeRequest struct {
	ContentID   uuid.UUID `json:"content_id" validate:"required,uuid"`
	ChannelIDs  []string  `json:"channel_ids" validate:"required,min=1,dive,uuid"`
	ScheduledAt string    `json:"scheduled_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	AutoPost    bool      `json:"auto_post"` // For FB/TikTok auto-publishing
	UserID      uuid.UUID `json:"-"`         // Set from JWT context
}

// GetSchedulesByContentRequest for getting schedules for a specific content
type GetSchedulesByContentRequest struct {
	ContentID uuid.UUID `json:"-" validate:"required,uuid"` // Set from path parameter
	Status    *string   `form:"status" validate:"omitempty,oneof=PENDING PROCESSING COMPLETED FAILED CANCELLED"`
}
