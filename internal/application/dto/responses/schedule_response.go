package responses

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BatchContentScheduleResponse represents the response after batch scheduling content
type BatchContentScheduleResponse struct {
	ContentID         uuid.UUID                         `json:"content_id"`
	ContentTitle      string                            `json:"content_title"`
	TotalScheduled    int                               `json:"total_scheduled"`
	TotalFailed       int                               `json:"total_failed"`
	ScheduledChannels []BatchContentScheduleResultItem  `json:"scheduled_channels"`
	FailedChannels    []BatchContentScheduleFailureItem `json:"failed_channels,omitempty"`
}

// BatchContentScheduleResultItem represents a successfully scheduled channel
type BatchContentScheduleResultItem struct {
	ScheduleID  uuid.UUID `json:"schedule_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	ChannelCode string    `json:"channel_code"`
	ScheduledAt time.Time `json:"scheduled_at"`
	AutoPost    bool      `json:"auto_post"`
}

// BatchContentScheduleFailureItem represents a failed schedule attempt
type BatchContentScheduleFailureItem struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	Error       string    `json:"error"`
}

// SchedulePaginationResponse represents a paginated response for schedules.
type SchedulePaginationResponse PaginationResponse[ScheduleInfoResponse]

// region: ============== Schedule Info Response ==============

type ScheduleInfoResponse struct {
	ID             uuid.UUID           `json:"id"`
	ReferenceID    *uuid.UUID          `json:"reference_id"`
	ReferenceType  *enum.ReferenceType `json:"reference_type"`
	Type           enum.ScheduleType   `json:"type"`
	ScheduledAt    time.Time           `json:"scheduled_at"`
	Status         enum.ScheduleStatus `json:"status"`
	RetryCount     int                 `json:"retry_count"`
	LastError      *string             `json:"last_error,omitempty"`
	ExecutedAt     *time.Time          `json:"executed_at,omitempty"`
	CreatedAt      *time.Time          `json:"created_at,omitempty"`
	UpdatedAt      *time.Time          `json:"updated_at,omitempty"`
	CreatedBy      *uuid.UUID          `json:"created_by,omitempty"`
	CreatedByName  *string             `json:"created_by_name,omitempty"`
	CreatedByEmail *string             `json:"created_by_email,omitempty"`

	// Nested details based on schedule type
	ContentDetails      *dtos.ContentScheduleDetails      `json:"content_details,omitempty"`
	NotificationDetails *dtos.NotificationScheduleDetails `json:"notification_details,omitempty"`
}

func (ScheduleInfoResponse) ToResponse(schedule *model.Schedule) *ScheduleInfoResponse {
	if schedule == nil {
		return nil
	}

	resp := &ScheduleInfoResponse{
		ID:            schedule.ID,
		ReferenceID:   schedule.ReferenceID,
		ReferenceType: schedule.ReferenceType,
		Type:          schedule.Type,
		ScheduledAt:   schedule.ScheduledAt,
		Status:        schedule.Status,
		RetryCount:    schedule.RetryCount,
		LastError:     schedule.LastError,
		ExecutedAt:    schedule.ExecutedAt,
		CreatedAt:     schedule.CreatedAt,
		UpdatedAt:     schedule.UpdatedAt,
		CreatedBy:     &schedule.CreatedBy,
	}
	if schedule.Creator != nil {
		resp.CreatedByName = &schedule.Creator.FullName
		resp.CreatedByEmail = &schedule.Creator.Email
	}

	return resp
}

func (ScheduleInfoResponse) ToListResponse(schedules []model.Schedule) []ScheduleInfoResponse {
	if len(schedules) == 0 {
		return []ScheduleInfoResponse{}
	}
	responses := make([]ScheduleInfoResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = *ScheduleInfoResponse{}.ToResponse(&schedule)
	}
	return responses
}

// endregion

// region: ============== Schedule Detail Response ==============

type ScheduleDetailResponse struct {
	ScheduleInfoResponse
	Metadata any `json:"metadata,omitempty"`
}

func (ScheduleDetailResponse) ToResponse(schedule *model.Schedule) *ScheduleDetailResponse {
	if schedule == nil {
		return nil
	}

	var metadata map[string]any
	if schedule.Metadata != nil {
		_ = json.Unmarshal(schedule.Metadata, &metadata)
	}

	return &ScheduleDetailResponse{
		ScheduleInfoResponse: *ScheduleInfoResponse{}.ToResponse(schedule),
		Metadata:             metadata,
	}
}

// endregion
