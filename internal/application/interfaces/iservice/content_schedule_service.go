package iservice

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// ContentScheduleService defines the interface for content scheduling operations
type ContentScheduleService interface {
	// ScheduleContent schedules content for future publishing via RabbitMQ delayed message
	ScheduleContent(ctx context.Context, req *requests.ScheduleContentRequest) (*responses.ScheduleResponse, error)

	// RescheduleContent cancels existing schedule and creates a new one
	RescheduleContent(ctx context.Context, scheduleID uuid.UUID, req *requests.RescheduleContentRequest) (*responses.ScheduleResponse, error)

	// CancelSchedule cancels a pending schedule
	CancelSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// GetSchedule retrieves a schedule by ID
	GetSchedule(ctx context.Context, scheduleID uuid.UUID) (*responses.ScheduleItemResponse, error)

	// GetScheduleByID returns schedule details by ID (for consumer)
	GetScheduleByID(ctx context.Context, scheduleID uuid.UUID) (*dtos.ScheduleDTO, error)

	// GetUpcomingSchedules returns schedules for the next N days
	GetUpcomingSchedules(ctx context.Context, days int) ([]responses.ScheduledContentItem, error)

	// ListSchedules returns schedules with filtering and pagination
	ListSchedules(ctx context.Context, filter *requests.ScheduleFilterRequest) (*responses.ScheduleListResponse, error)

	// ProcessSchedule is called by the consumer to execute the scheduled publish
	ProcessSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// ExecuteScheduledPublish executes the scheduled content publishing
	ExecuteScheduledPublish(ctx context.Context, scheduleID uuid.UUID) error

	// UpdateScheduleStatus updates the status of a schedule
	UpdateScheduleStatus(ctx context.Context, scheduleID uuid.UUID, status enum.ScheduleStatus) error
}
