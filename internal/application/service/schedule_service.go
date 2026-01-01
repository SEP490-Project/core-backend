package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	asynqClient "core-backend/internal/infrastructure/asynq"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// scheduleService implements generic schedule operations
type scheduleService struct {
	scheduleRepo  irepository.ScheduleRepository
	taskScheduler *asynqClient.AsynqClient
}

// NewScheduleService creates a new generic schedule service
func NewScheduleService(scheduleRepo irepository.ScheduleRepository, taskScheduler *asynqClient.AsynqClient) iservice.ScheduleService {
	return &scheduleService{
		scheduleRepo:  scheduleRepo,
		taskScheduler: taskScheduler,
	}
}

// GetByID retrieves a schedule by ID
func (s *scheduleService) GetByID(ctx context.Context, id uuid.UUID) (*model.Schedule, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, id, nil)
	if err != nil {
		zap.L().Error("Failed to get schedule by ID", zap.Error(err), zap.String("schedule_id", id.String()))
		return nil, err
	}
	if schedule == nil {
		return nil, errors.New("schedule not found")
	}
	return schedule, nil
}

// GetByIDWithDetails retrieves a schedule by ID with additional details based on type
func (s *scheduleService) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error) {
	// First get the schedule to determine its type
	schedule, err := s.scheduleRepo.GetByID(ctx, id, nil)
	if err != nil {
		zap.L().Error("Failed to get schedule by ID", zap.Error(err), zap.String("schedule_id", id.String()))
		return nil, err
	}
	if schedule == nil {
		return nil, errors.New("schedule not found")
	}

	// Based on schedule type, fetch with appropriate JOINs
	if schedule.Type == enum.ScheduleTypeContentPublish {
		return s.scheduleRepo.GetContentScheduleByIDWithDetails(ctx, id)
	}

	// For other types, return basic details
	return s.scheduleRepo.GetScheduleByIDWithDetails(ctx, id)
}

// List returns schedules with filtering and pagination
func (s *scheduleService) List(ctx context.Context, filter *requests.ScheduleFilterRequest) ([]dtos.ScheduleDTO, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// If filtering for content schedules specifically, use content-specific method
	if filter.ReferenceType != nil && *filter.ReferenceType == enum.ScheduleTypeContentPublish {
		return s.scheduleRepo.GetContentSchedulesWithDetails(ctx, filter)
	}

	return s.scheduleRepo.GetSchedulesWithDetails(ctx, filter)
}

// Cancel cancels a pending schedule and removes the task from Asynq queue
func (s *scheduleService) Cancel(ctx context.Context, scheduleID uuid.UUID, cancelledBy uuid.UUID) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID, nil)
	if err != nil {
		zap.L().Error("Failed to get schedule for cancellation", zap.Error(err), zap.String("schedule_id", scheduleID.String()))
		return err
	}
	if schedule == nil {
		return errors.New("schedule not found")
	}

	if !schedule.IsPending() {
		return errors.New("can only cancel pending schedules")
	}

	// Cancel the Asynq task in Redis
	if s.taskScheduler != nil {
		uniqueKey := fmt.Sprintf("schedule:%s", scheduleID.String())
		if err := s.taskScheduler.CancelTask(uniqueKey); err != nil {
			// Log but don't fail - task may have already been processed or expired
			zap.L().Warn("Failed to cancel Asynq task (may have already been processed)",
				zap.Error(err),
				zap.String("task_id", uniqueKey),
				zap.String("schedule_id", scheduleID.String()))
		}
	}

	// Use model helper method
	schedule.MarkCancelled()

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		zap.L().Error("Failed to cancel schedule", zap.Error(err), zap.String("schedule_id", scheduleID.String()))
		return err
	}

	zap.L().Info("Schedule cancelled",
		zap.String("schedule_id", scheduleID.String()),
		zap.String("cancelled_by", cancelledBy.String()))

	return nil
}

// UpdateStatus updates the status of a schedule using model helpers
func (s *scheduleService) UpdateStatus(ctx context.Context, schedule *model.Schedule, status enum.ScheduleStatus, errorMsg *string) error {
	switch status {
	case enum.ScheduleStatusProcessing:
		schedule.MarkProcessing()
	case enum.ScheduleStatusCompleted:
		schedule.MarkCompleted()
	case enum.ScheduleStatusFailed:
		errStr := "unknown error"
		if errorMsg != nil {
			errStr = *errorMsg
		}
		schedule.MarkFailed(errStr)
	case enum.ScheduleStatusCancelled:
		schedule.MarkCancelled()
	default:
		schedule.Status = status
	}

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		zap.L().Error("Failed to update schedule status",
			zap.Error(err),
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("status", status.String()))
		return err
	}

	return nil
}

// GetUpcoming returns upcoming schedules within the next N days
func (s *scheduleService) GetUpcoming(ctx context.Context, days int, scheduleType *enum.ScheduleType, limit int) ([]dtos.ScheduleDTO, error) {
	if days <= 0 {
		days = 7
	}
	if limit <= 0 {
		limit = 50
	}

	now := time.Now()
	to := now.AddDate(0, 0, days)

	var schedules []*model.Schedule
	var err error

	if scheduleType != nil {
		schedules, err = s.scheduleRepo.GetUpcomingSchedulesByType(ctx, *scheduleType, now, to, limit)
	} else {
		schedules, err = s.scheduleRepo.GetUpcomingSchedules(ctx, now, to, limit)
	}

	if err != nil {
		zap.L().Error("Failed to get upcoming schedules", zap.Error(err))
		return nil, err
	}

	// Convert to DTOs with details based on type
	results := make([]dtos.ScheduleDTO, 0, len(schedules))
	for _, schedule := range schedules {
		dto, err := s.GetByIDWithDetails(ctx, schedule.ID)
		if err != nil {
			zap.L().Warn("Failed to get schedule details", zap.Error(err), zap.String("schedule_id", schedule.ID.String()))
			continue
		}
		if dto != nil {
			results = append(results, *dto)
		}
	}

	return results, nil
}

// GetByReferenceID returns all schedules for a reference ID
func (s *scheduleService) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*model.Schedule, error) {
	return s.scheduleRepo.GetPendingByReferenceID(ctx, referenceID)
}

// CancelByReferenceID cancels all pending schedules for a reference ID and removes tasks from Asynq
func (s *scheduleService) CancelByReferenceID(ctx context.Context, referenceID uuid.UUID, cancelledBy uuid.UUID) error {
	// First, get all pending schedules to cancel their Asynq tasks
	if s.taskScheduler != nil {
		pendingSchedules, err := s.scheduleRepo.GetPendingByReferenceID(ctx, referenceID)
		if err == nil {
			for _, schedule := range pendingSchedules {
				uniqueKey := fmt.Sprintf("schedule:%s", schedule.ID.String())
				if err := s.taskScheduler.CancelTask(uniqueKey); err != nil {
					// Log but continue - task may have already been processed
					zap.L().Warn("Failed to cancel Asynq task (may have already been processed)",
						zap.Error(err),
						zap.String("task_id", uniqueKey),
						zap.String("schedule_id", schedule.ID.String()))
				}
			}
		}
	}

	// Then cancel in database
	err := s.scheduleRepo.CancelScheduleByReferenceID(ctx, referenceID)
	if err != nil {
		zap.L().Error("Failed to cancel schedules by reference ID",
			zap.Error(err),
			zap.String("reference_id", referenceID.String()))
		return err
	}

	zap.L().Info("Schedules cancelled by reference ID",
		zap.String("reference_id", referenceID.String()),
		zap.String("cancelled_by", cancelledBy.String()))

	return nil
}

// GetPendingSchedules returns all pending schedules that should be processed
func (s *scheduleService) GetPendingSchedules(ctx context.Context) ([]*model.Schedule, error) {
	return s.scheduleRepo.GetPendingSchedules(ctx, time.Now())
}
