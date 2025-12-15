package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ContentScheduleRepository defines the interface for content schedule data access
type ContentScheduleRepository interface {
	// Create creates a new content schedule
	Create(ctx context.Context, schedule *model.ContentSchedule) error

	// GetByID returns a schedule by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.ContentSchedule, error)

	// GetByContentChannelID returns a schedule by content channel ID
	GetByContentChannelID(ctx context.Context, contentChannelID uuid.UUID) (*model.ContentSchedule, error)

	// Update updates an existing schedule
	Update(ctx context.Context, schedule *model.ContentSchedule) error

	// Delete soft deletes a schedule
	Delete(ctx context.Context, id uuid.UUID) error

	// GetPendingSchedules returns all pending schedules that should be processed
	GetPendingSchedules(ctx context.Context, before time.Time) ([]*model.ContentSchedule, error)

	// GetSchedulesByStatus returns schedules by status
	GetSchedulesByStatus(ctx context.Context, status enum.ScheduleStatus, pageSize, pageNumber int) ([]*model.ContentSchedule, int64, error)

	// GetSchedulesWithDetails returns schedules with content and channel details
	GetSchedulesWithDetails(ctx context.Context, filter *ScheduleFilter) ([]dtos.ScheduleDTO, int64, error)

	// GetUpcomingSchedules returns upcoming schedules within a time range
	GetUpcomingSchedules(ctx context.Context, from, to time.Time, limit int) ([]*model.ContentSchedule, error)

	// CancelScheduleByContentChannelID cancels a schedule by content channel ID
	CancelScheduleByContentChannelID(ctx context.Context, contentChannelID uuid.UUID) error

	// GetScheduleByIDWithDetails returns a single schedule with full details
	GetScheduleByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error)
}

// ScheduleFilter defines filter options for schedule queries
type ScheduleFilter struct {
	Status     *enum.ScheduleStatus
	ChannelID  *uuid.UUID
	From       *time.Time
	To         *time.Time
	PageSize   int
	PageNumber int
}
