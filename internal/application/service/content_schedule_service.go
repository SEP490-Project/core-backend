package service

import (
	"context"
	"core-backend/config"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	asynqClient "core-backend/internal/infrastructure/asynq"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentScheduleService struct {
	scheduleRepo             irepository.ScheduleRepository
	contentChannelRepo       irepository.GenericRepository[model.ContentChannel]
	contentRepo              irepository.GenericRepository[model.Content]
	channelRepo              irepository.GenericRepository[model.Channel]
	unitOfWork               irepository.UnitOfWork
	contentPublishingService iservice.ContentPublishingService
	taskScheduler            *asynqClient.AsynqClient
	asynqConfig              *config.AsynqConfig
}

// NewContentScheduleService creates a new content schedule service
func NewContentScheduleService(
	dbReg *gormrepository.DatabaseRegistry,
	contentPublishingService iservice.ContentPublishingService,
	taskScheduler *asynqClient.AsynqClient,
	asynqConfig *config.AsynqConfig,
) iservice.ContentScheduleService {
	return &contentScheduleService{
		scheduleRepo:             dbReg.ScheduleRepository,
		contentChannelRepo:       dbReg.ContentChannelRepository,
		contentRepo:              dbReg.ContentRepository,
		channelRepo:              dbReg.ChannelRepository,
		contentPublishingService: contentPublishingService,
		taskScheduler:            taskScheduler,
		asynqConfig:              asynqConfig,
	}
}

// ScheduleContent schedules content for future publishing via RabbitMQ delayed message
func (s *contentScheduleService) ScheduleContent(ctx context.Context, req *requests.ScheduleContentRequest) (*responses.ScheduleResponse, error) {
	currentTime := time.Now()
	zap.L().Info("Scheduling content for publishing", zap.Any("request", req))

	// 1. Parse scheduled time
	scheduledAt := utils.ParseLocalTimeWithFallback(req.ScheduledAt, time.RFC3339)
	if scheduledAt == nil {
		zap.L().Warn("Invalid scheduled_at format", zap.String("scheduled_at", req.ScheduledAt))
		return nil, errors.New("invalid scheduled_at format")
	}

	// 2. Validate scheduled time is in the future
	if scheduledAt.Before(currentTime) {
		return nil, errors.New("scheduled_at must be in the future")
	}

	// 3. Validate content channel exists
	contentChannel, err := s.contentChannelRepo.GetByID(ctx, req.ContentChannelID, nil)
	if err != nil || contentChannel == nil {
		return nil, errors.New("content channel not found")
	}

	// 4. Validate content exists and is in approved status
	content, err := s.contentRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND status = ?", contentChannel.ContentID, enum.ContentStatusApproved)
	}, nil)
	if err != nil || content == nil {
		return nil, errors.New("content not found or not approved")
	}

	// 5. Check if there's already a pending schedule for this content channel
	existingSchedule, err := s.scheduleRepo.GetByReferenceID(ctx, req.ContentChannelID)
	if err == nil && existingSchedule != nil && existingSchedule.Status == enum.ScheduleStatusPending {
		return nil, errors.New("content channel already has a pending schedule")
	}

	// 6. Create schedule record
	schedule := &model.Schedule{
		ReferenceID: &req.ContentChannelID,
		Type:        enum.ScheduleTypeContentPublish,
		ScheduledAt: *scheduledAt,
		Status:      enum.ScheduleStatusPending,
		RetryCount:  0,
		CreatedBy:   req.UserID,
	}

	if err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.Schedules().Add(ctx, schedule); err != nil {
			zap.L().Error("Failed to create schedule", zap.Error(err))
			return errors.New("failed to create schedule")
		}

		// 7. Publish delayed message to RabbitMQ
		if err = s.publishDelayedMessage(ctx, schedule); err != nil {
			zap.L().Error("Failed to publish delayed message", zap.Error(err))
			return errors.New("failed to schedule publishing: " + err.Error())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	zap.L().Info("Content scheduled for publishing",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("content_channel_id", req.ContentChannelID.String()),
		zap.Time("scheduled_at", *scheduledAt))

	return &responses.ScheduleResponse{
		ScheduleID:       schedule.ID,
		ContentChannelID: utils.DerefPtr(schedule.ReferenceID, uuid.Nil),
		ScheduledAt:      *scheduledAt,
		Status:           string(schedule.Status),
	}, nil
}

// BatchScheduleContent schedules content to multiple channels at once
func (s *contentScheduleService) BatchScheduleContent(ctx context.Context, req *requests.BatchScheduleRequest) (*responses.BatchScheduleResponse, error) {
	zap.L().Info("Batch scheduling content",
		zap.String("content_id", req.ContentID.String()),
		zap.Int("schedule_count", len(req.Schedules)))

	// 1. Validate content exists and is approved
	content, err := s.contentRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND status = ?", req.ContentID, enum.ContentStatusApproved)
	}, nil)
	if err != nil || content == nil {
		return nil, errors.New("content not found or not approved")
	}

	// 2. Get all content channels for this content
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("content_id = ?", req.ContentID)
	}, []string{"Channel"}, 100, 1)
	if err != nil {
		return nil, errors.New("failed to get content channels")
	}

	// Build content channel lookup by channel ID
	ccByChannelID := make(map[uuid.UUID]*model.ContentChannel)
	for i := range contentChannels {
		ccByChannelID[contentChannels[i].ChannelID] = &contentChannels[i]
	}

	// 3. Process each schedule item
	response := &responses.BatchScheduleResponse{
		ContentID:         req.ContentID,
		ContentTitle:      content.Title,
		ScheduledChannels: make([]responses.BatchScheduleResultItem, 0),
		FailedChannels:    make([]responses.BatchScheduleFailureItem, 0),
	}

	currentTime := time.Now()

	for _, item := range req.Schedules {
		channelID, parseErr := uuid.Parse(item.ChannelID)
		if parseErr != nil {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID: uuid.Nil,
				Error:     "invalid channel ID format",
			})
			continue
		}

		// Find content channel for this channel
		cc, exists := ccByChannelID[channelID]
		if !exists {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID: channelID,
				Error:     "content not assigned to this channel",
			})
			continue
		}

		// Parse scheduled time
		scheduledAt := utils.ParseLocalTimeWithFallback(item.ScheduledAt, time.RFC3339)
		if scheduledAt == nil {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID:   channelID,
				ChannelName: cc.Channel.Name,
				Error:       "invalid scheduled_at format",
			})
			continue
		}

		// Validate scheduled time is in the future
		if scheduledAt.Before(currentTime) {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID:   channelID,
				ChannelName: cc.Channel.Name,
				Error:       "scheduled_at must be in the future",
			})
			continue
		}

		// Check for existing pending schedule
		existingSchedule, _ := s.scheduleRepo.GetByReferenceID(ctx, cc.ID)
		if existingSchedule != nil && existingSchedule.Status == enum.ScheduleStatusPending {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID:   channelID,
				ChannelName: cc.Channel.Name,
				Error:       "channel already has a pending schedule",
			})
			continue
		}

		// Create schedule
		schedule := &model.Schedule{
			ReferenceID: &cc.ID,
			Type:        enum.ScheduleTypeContentPublish,
			ScheduledAt: *scheduledAt,
			Status:      enum.ScheduleStatusPending,
			RetryCount:  0,
			CreatedBy:   req.UserID,
		}

		if err := s.scheduleRepo.Add(ctx, schedule); err != nil {
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID:   channelID,
				ChannelName: cc.Channel.Name,
				Error:       "failed to create schedule",
			})
			continue
		}

		// Publish delayed message
		if err := s.publishDelayedMessage(ctx, schedule); err != nil {
			// Rollback schedule creation
			_ = s.scheduleRepo.DeleteByID(ctx, schedule.ID)
			response.FailedChannels = append(response.FailedChannels, responses.BatchScheduleFailureItem{
				ChannelID:   channelID,
				ChannelName: cc.Channel.Name,
				Error:       "failed to queue schedule: " + err.Error(),
			})
			continue
		}

		// Success
		response.ScheduledChannels = append(response.ScheduledChannels, responses.BatchScheduleResultItem{
			ScheduleID:  schedule.ID,
			ChannelID:   channelID,
			ChannelName: cc.Channel.Name,
			ChannelCode: cc.Channel.Code,
			ScheduledAt: *scheduledAt,
			AutoPost:    item.AutoPost,
		})
	}

	response.TotalScheduled = len(response.ScheduledChannels)
	response.TotalFailed = len(response.FailedChannels)

	zap.L().Info("Batch scheduling completed",
		zap.String("content_id", req.ContentID.String()),
		zap.Int("scheduled", response.TotalScheduled),
		zap.Int("failed", response.TotalFailed))

	return response, nil
}

// RescheduleContent cancels existing schedule and creates a new one
func (s *contentScheduleService) RescheduleContent(ctx context.Context, scheduleID uuid.UUID, req *requests.RescheduleContentRequest) (*responses.ScheduleResponse, error) {
	// 1. Get existing schedule
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID, nil)
	if err != nil || schedule == nil {
		return nil, errors.New("schedule not found")
	}

	// 2. Only pending schedules can be rescheduled
	if schedule.Status != enum.ScheduleStatusPending {
		return nil, errors.New("only pending schedules can be rescheduled")
	}

	// 3. Parse new scheduled time
	newScheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return nil, errors.New("invalid scheduled_at format, use RFC3339 format")
	}

	if newScheduledAt.Before(time.Now()) {
		return nil, errors.New("scheduled_at must be in the future")
	}

	// 4. Cancel the old schedule
	schedule.Status = enum.ScheduleStatusCancelled
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, errors.New("failed to cancel old schedule")
	}

	// 5. Create new schedule
	newSchedule := &model.Schedule{
		ReferenceID: schedule.ReferenceID,
		Type:        enum.ScheduleTypeContentPublish,
		ScheduledAt: newScheduledAt,
		Status:      enum.ScheduleStatusPending,
		RetryCount:  0,
		CreatedBy:   req.UserID,
	}

	if err := s.scheduleRepo.Add(ctx, newSchedule); err != nil {
		// Rollback - restore old schedule
		schedule.Status = enum.ScheduleStatusPending
		_ = s.scheduleRepo.Update(ctx, schedule)
		return nil, errors.New("failed to create new schedule")
	}

	// 6. Publish new delayed message
	if err := s.publishDelayedMessage(ctx, newSchedule); err != nil {
		// Rollback
		_ = s.scheduleRepo.DeleteByID(ctx, newSchedule.ID)
		schedule.Status = enum.ScheduleStatusPending
		_ = s.scheduleRepo.Update(ctx, schedule)
		return nil, errors.New("failed to schedule publishing: " + err.Error())
	}

	zap.L().Info("Content rescheduled",
		zap.String("old_schedule_id", scheduleID.String()),
		zap.String("new_schedule_id", newSchedule.ID.String()),
		zap.Time("new_scheduled_at", newScheduledAt))

	return &responses.ScheduleResponse{
		ScheduleID:       newSchedule.ID,
		ContentChannelID: utils.DerefPtr(newSchedule.ReferenceID, uuid.Nil),
		ScheduledAt:      newScheduledAt,
		Status:           string(newSchedule.Status),
	}, nil
}

// CancelSchedule cancels a pending schedule
func (s *contentScheduleService) CancelSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID, nil)
	if err != nil || schedule == nil {
		return errors.New("schedule not found")
	}

	if schedule.Status != enum.ScheduleStatusPending {
		return errors.New("only pending schedules can be cancelled")
	}

	// Cancel the Asynq task
	uniqueKey := fmt.Sprintf("schedule:%s", scheduleID.String())
	if s.taskScheduler != nil {
		if err := s.taskScheduler.CancelTask(uniqueKey); err != nil {
			// Log but don't fail - the task might have already been processed
			zap.L().Warn("Failed to cancel Asynq task (may have already been processed)",
				zap.String("task_id", uniqueKey),
				zap.Error(err))
		}
	}

	schedule.Status = enum.ScheduleStatusCancelled
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return errors.New("failed to cancel schedule")
	}

	zap.L().Info("Schedule cancelled", zap.String("schedule_id", scheduleID.String()))
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *contentScheduleService) GetSchedule(ctx context.Context, scheduleID uuid.UUID) (*responses.ScheduleItemResponse, error) {
	scheduleDTO, err := s.scheduleRepo.GetScheduleByIDWithDetails(ctx, scheduleID)
	if err != nil || scheduleDTO == nil {
		return nil, errors.New("schedule not found")
	}

	return &responses.ScheduleItemResponse{
		ScheduleID:       scheduleDTO.ScheduleID,
		ContentChannelID: scheduleDTO.ContentChannelID,
		ContentID:        scheduleDTO.ContentID,
		ContentTitle:     scheduleDTO.ContentTitle,
		ContentType:      scheduleDTO.ContentType.String(),
		ChannelID:        scheduleDTO.ChannelID,
		ChannelName:      scheduleDTO.ChannelName,
		ChannelCode:      scheduleDTO.ChannelCode,
		ScheduledAt:      scheduleDTO.ScheduledAt,
		Status:           scheduleDTO.Status.String(),
		RetryCount:       scheduleDTO.RetryCount,
		LastError:        scheduleDTO.LastError,
		ExecutedAt:       scheduleDTO.ExecutedAt,
		CreatedAt:        scheduleDTO.CreatedAt,
		CreatedBy:        scheduleDTO.CreatedByName,
		CreatedByID:      scheduleDTO.CreatedBy,
	}, nil
}

// GetUpcomingSchedules returns schedules for the next N days
func (s *contentScheduleService) GetUpcomingSchedules(ctx context.Context, days int) ([]responses.ScheduledContentItem, error) {
	from := time.Now()
	to := from.AddDate(0, 0, days)

	schedules, err := s.scheduleRepo.GetUpcomingSchedules(ctx, from, to, 100)
	if err != nil {
		return nil, errors.New("failed to get upcoming schedules")
	}

	result := make([]responses.ScheduledContentItem, 0, len(schedules))
	for _, schedule := range schedules {
		// Get details for each schedule
		scheduleDTO, err := s.scheduleRepo.GetScheduleByIDWithDetails(ctx, schedule.ID)
		if err != nil || scheduleDTO == nil {
			continue
		}

		result = append(result, responses.ScheduledContentItem{
			ScheduleID:  scheduleDTO.ScheduleID,
			ContentID:   scheduleDTO.ContentID,
			Title:       scheduleDTO.ContentTitle,
			ChannelName: scheduleDTO.ChannelName,
			ScheduledAt: scheduleDTO.ScheduledAt,
			Status:      scheduleDTO.Status.String(),
			CreatedBy:   scheduleDTO.CreatedByName,
			CreatedByID: scheduleDTO.CreatedBy,
		})
	}

	return result, nil
}

// ListSchedules returns schedules with filtering and pagination
func (s *contentScheduleService) ListSchedules(ctx context.Context, filter *requests.ScheduleFilterRequest) (*responses.ScheduleListResponse, error) {
	// Get schedules
	schedules, total, err := s.scheduleRepo.GetSchedulesWithDetails(ctx, filter)
	if err != nil {
		return nil, errors.New("failed to get schedules")
	}

	// Convert to response
	items := make([]responses.ScheduleItemResponse, 0, len(schedules))
	for _, dto := range schedules {
		items = append(items, responses.ScheduleItemResponse{
			ScheduleID:       dto.ScheduleID,
			ContentChannelID: dto.ContentChannelID,
			ContentID:        dto.ContentID,
			ContentTitle:     dto.ContentTitle,
			ContentType:      dto.ContentType.String(),
			ChannelID:        dto.ChannelID,
			ChannelName:      dto.ChannelName,
			ChannelCode:      dto.ChannelCode,
			ScheduledAt:      dto.ScheduledAt,
			Status:           dto.Status.String(),
			RetryCount:       dto.RetryCount,
			LastError:        dto.LastError,
			ExecutedAt:       dto.ExecutedAt,
			CreatedAt:        dto.CreatedAt,
			CreatedBy:        dto.CreatedByName,
			CreatedByID:      dto.CreatedBy,
		})
	}

	return &responses.ScheduleListResponse{
		Schedules: items,
		Total:     total,
	}, nil
}

// ProcessSchedule is called by the consumer to execute the scheduled publish
func (s *contentScheduleService) ProcessSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	// 1. Get schedule
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID, nil)
	if err != nil || schedule == nil {
		zap.L().Warn("Schedule not found for processing", zap.String("schedule_id", scheduleID.String()))
		return errors.New("schedule not found")
	}

	// 2. Check if schedule is still pending
	if schedule.Status != enum.ScheduleStatusPending {
		zap.L().Info("Schedule is not pending, skipping",
			zap.String("schedule_id", scheduleID.String()),
			zap.String("status", string(schedule.Status)))
		return nil // Not an error, just skip
	}

	// 3. Mark as processing
	schedule.Status = enum.ScheduleStatusProcessing
	if err = s.scheduleRepo.Update(ctx, schedule); err != nil {
		return errors.New("failed to update schedule status")
	}

	// 4. Get content channel and channel details
	contentChannel, err := s.contentChannelRepo.GetByID(ctx, schedule.ReferenceID, nil)
	if err != nil || contentChannel == nil {
		schedule.Status = enum.ScheduleStatusFailed
		errMsg := "content channel not found"
		schedule.LastError = &errMsg
		_ = s.scheduleRepo.Update(ctx, schedule)
		return errors.New(errMsg)
	}

	// 5. Execute the publish
	_, err = s.contentPublishingService.PublishToChannel(ctx, contentChannel.ContentID, contentChannel.ChannelID, schedule.CreatedBy)
	if err != nil {
		// Handle failure with retry logic
		return s.handlePublishFailure(ctx, schedule, err)
	}

	// 6. Mark as completed
	now := time.Now()
	schedule.Status = enum.ScheduleStatusCompleted
	schedule.ExecutedAt = &now
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		zap.L().Error("Failed to mark schedule as completed", zap.Error(err))
	}

	zap.L().Info("Scheduled content published successfully",
		zap.String("schedule_id", scheduleID.String()),
		zap.String("content_channel_id", schedule.ReferenceID.String()))

	return nil
}

// handlePublishFailure handles publish failure with retry logic
func (s *contentScheduleService) handlePublishFailure(ctx context.Context, schedule *model.Schedule, publishErr error) error {
	schedule.RetryCount++
	errMsg := publishErr.Error()
	schedule.LastError = &errMsg

	if schedule.RetryCount >= constant.DefaultMaxScheduleRetries {
		// Max retries exceeded
		schedule.Status = enum.ScheduleStatusFailed
		if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
			zap.L().Error("Failed to update schedule status", zap.Error(err))
		}
		zap.L().Error("Schedule failed after max retries",
			zap.String("schedule_id", schedule.ID.String()),
			zap.Int("retry_count", schedule.RetryCount),
			zap.Error(publishErr))
		return errors.New("publish failed after max retries: " + errMsg)
	}

	// Set back to pending for retry
	schedule.Status = enum.ScheduleStatusPending
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		zap.L().Error("Failed to update schedule for retry", zap.Error(err))
	}

	// Schedule retry with exponential backoff (5min, 10min, 20min)
	retryDelay := time.Duration(5<<schedule.RetryCount) * time.Minute
	schedule.ScheduledAt = time.Now().Add(retryDelay)
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		zap.L().Error("Failed to update schedule retry time", zap.Error(err))
	}

	// Republish with new delay
	if err := s.publishDelayedMessage(ctx, schedule); err != nil {
		zap.L().Error("Failed to republish for retry", zap.Error(err))
	}

	zap.L().Warn("Scheduled publish failed, will retry",
		zap.String("schedule_id", schedule.ID.String()),
		zap.Int("retry_count", schedule.RetryCount),
		zap.Duration("retry_delay", retryDelay),
		zap.Error(publishErr))

	return nil // Return nil since we're handling the retry
}

// publishDelayedMessage schedules a task via Asynq for future execution
func (s *contentScheduleService) publishDelayedMessage(ctx context.Context, schedule *model.Schedule) error {
	if s.taskScheduler == nil {
		return errors.New("task scheduler not initialized")
	}

	// Get content channel for channel code
	contentChannel, err := s.contentChannelRepo.GetByID(ctx, schedule.ReferenceID, []string{"Channel"})
	if err != nil || contentChannel == nil {
		return errors.New("content channel not found: " + err.Error())
	}

	channelCode := ""
	if contentChannel.Channel != nil {
		channelCode = contentChannel.Channel.Code
	}

	// Create task payload
	payload := asynqtask.ContentScheduleTaskPayload{
		ScheduleID:       schedule.ID,
		ContentChannelID: utils.DerefPtr(schedule.ReferenceID, uuid.Nil),
		ContentID:        contentChannel.ContentID,
		ChannelCode:      channelCode,
		ScheduledAt:      schedule.ScheduledAt,
		RetryCount:       schedule.RetryCount,
	}

	// Get task type from config
	taskType := s.asynqConfig.TaskTypes.ContentSchedule
	if taskType == "" {
		taskType = "task:content:schedule"
	}

	// Create unique key for this schedule to prevent duplicates
	uniqueKey := fmt.Sprintf("schedule:%s", schedule.ID.String())

	// Schedule the task with Asynq
	_, err = s.taskScheduler.ScheduleTaskWithUniqueKey(
		ctx,
		taskType,
		payload,
		schedule.ScheduledAt,
		uniqueKey,
		asynq.Queue("default"),
		asynq.MaxRetry(constant.DefaultMaxScheduleRetries),
	)
	if err != nil {
		return errors.New("failed to schedule task: " + err.Error())
	}

	zap.L().Info("Content schedule task created",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("unique_key", uniqueKey),
		zap.Time("scheduled_at", schedule.ScheduledAt))

	return nil
}

// ExecuteScheduledPublish executes the scheduled content publishing
// This is an alias for ProcessSchedule for clarity in consumer usage
func (s *contentScheduleService) ExecuteScheduledPublish(ctx context.Context, scheduleID uuid.UUID) error {
	return s.ProcessSchedule(ctx, scheduleID)
}

// GetScheduleByID returns schedule details by ID (for consumer)
func (s *contentScheduleService) GetScheduleByID(ctx context.Context, scheduleID uuid.UUID) (*dtos.ScheduleDTO, error) {
	return s.scheduleRepo.GetScheduleByIDWithDetails(ctx, scheduleID)
}

// UpdateScheduleStatus updates the status of a schedule
func (s *contentScheduleService) UpdateScheduleStatus(ctx context.Context, scheduleID uuid.UUID, status enum.ScheduleStatus) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID, nil)
	if err != nil || schedule == nil {
		return errors.New("schedule not found")
	}

	schedule.Status = status
	return s.scheduleRepo.Update(ctx, schedule)
}
