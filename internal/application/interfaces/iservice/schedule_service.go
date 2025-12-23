package iservice

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ScheduleService defines the interface for generic schedule operations
type ScheduleService interface {
	// GetByID retrieves a schedule by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.Schedule, error)

	// GetByIDWithDetails retrieves a schedule by ID with additional details based on type
	GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error)

	// List returns schedules with filtering and pagination
	List(ctx context.Context, filter *requests.ScheduleFilterRequest) ([]dtos.ScheduleDTO, int64, error)

	// Cancel cancels a pending schedule
	Cancel(ctx context.Context, scheduleID uuid.UUID, cancelledBy uuid.UUID) error

	// UpdateStatus updates the status of a schedule using model helpers
	UpdateStatus(ctx context.Context, schedule *model.Schedule, status enum.ScheduleStatus, errorMsg *string) error

	// GetUpcoming returns upcoming schedules within the next N days
	GetUpcoming(ctx context.Context, days int, scheduleType *enum.ScheduleType, limit int) ([]dtos.ScheduleDTO, error)

	// GetByReferenceID returns all schedules for a reference ID
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*model.Schedule, error)

	// CancelByReferenceID cancels all pending schedules for a reference ID
	CancelByReferenceID(ctx context.Context, referenceID uuid.UUID, cancelledBy uuid.UUID) error

	// GetPendingSchedules returns all pending schedules that should be processed
	GetPendingSchedules(ctx context.Context) ([]*model.Schedule, error)
}
