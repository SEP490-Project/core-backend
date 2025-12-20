package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ScheduleRepository defines the interface for schedule data access
type ScheduleRepository interface {
	GenericRepository[model.Schedule]

	// GetByReferenceID returns a schedule by reference ID
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*model.Schedule, error)

	// GetPendingSchedules returns all pending schedules that should be processed
	GetPendingSchedules(ctx context.Context, before time.Time) ([]*model.Schedule, error)

	// GetSchedulesByStatus returns schedules by status
	GetSchedulesByStatus(ctx context.Context, status enum.ScheduleStatus, pageSize, pageNumber int) ([]*model.Schedule, int64, error)

	// GetSchedulesWithDetails returns schedules with content and channel details
	GetSchedulesWithDetails(ctx context.Context, filter *requests.ScheduleFilterRequest) ([]dtos.ScheduleDTO, int64, error)

	// GetUpcomingSchedules returns upcoming schedules within a time range
	GetUpcomingSchedules(ctx context.Context, from, to time.Time, limit int) ([]*model.Schedule, error)

	// CancelScheduleByReferenceID cancels a schedule by reference ID
	CancelScheduleByReferenceID(ctx context.Context, referenceID uuid.UUID) error

	// GetScheduleByIDWithDetails returns a single schedule with full details
	GetScheduleByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error)

	UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status enum.ScheduleStatus, lastError *string) error
}
