package iservice

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ContentScheduleService defines the interface for content scheduling operations
type ContentScheduleService interface {
	// ScheduleContent schedules content for future publishing via RabbitMQ delayed message
	ScheduleContent(ctx context.Context, req *requests.ScheduleContentRequest) (*responses.ScheduleDetailResponse, error)

	// BatchScheduleContent schedules content to multiple channels at once
	BatchScheduleContent(ctx context.Context, req *requests.BatchScheduleRequest) (*responses.BatchContentScheduleResponse, error)

	// RescheduleContent cancels existing schedule and creates a new one
	RescheduleContent(ctx context.Context, scheduleID uuid.UUID, req *requests.RescheduleContentRequest) (*responses.ScheduleDetailResponse, error)

	// ProcessSchedule is called by the consumer to execute the scheduled publish
	ProcessSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// GetScheduleByContentID retrieves schedule by content ID
	GetScheduleByContentID(ctx context.Context, contentID uuid.UUID) ([]dtos.ScheduleDTO, error)
}
