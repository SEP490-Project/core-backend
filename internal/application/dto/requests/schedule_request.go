package requests

import "github.com/google/uuid"

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
	Status    *string `form:"status" validate:"omitempty,oneof=PENDING PROCESSING COMPLETED FAILED CANCELLED"`
	ChannelID *string `form:"channel_id" validate:"omitempty,uuid"`
	FromDate  *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate    *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`
	Days      *int    `form:"days" validate:"omitempty,min=1,max=90"` // Get schedules for next N days
}

// GetDaysOrDefault returns the days parameter with default of 7
func (r *ScheduleFilterRequest) GetDaysOrDefault() int {
	if r.Days != nil {
		return *r.Days
	}
	return 7
}
