package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	stringsbuilder "core-backend/pkg/strings_builder"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NotificationScheduledHandler handles scheduled notification tasks
// This handler receives scheduled notification tasks from Asynq and publishes
// them to RabbitMQ for asynchronous processing by the notification consumers
type NotificationScheduledHandler struct {
	notificationService iservice.NotificationService
	unitOfWork          irepository.UnitOfWork
}

// NewNotificationScheduledHandler creates a new notification scheduled handler
func NewNotificationScheduledHandler(
	notificationService iservice.NotificationService,
	unitOfWork irepository.UnitOfWork,
) *NotificationScheduledHandler {
	return &NotificationScheduledHandler{
		notificationService: notificationService,
		unitOfWork:          unitOfWork,
	}
}

// ProcessTask processes scheduled notification tasks
// When a notification task is executed at the scheduled time, this handler
// creates the notification and publishes it to RabbitMQ for delivery
func (h *NotificationScheduledHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing scheduled notification task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	// Parse payload
	var payload asynqtask.ScheduledNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal notification payload",
			zap.Error(err),
			zap.ByteString("raw_payload", task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing scheduled notification",
		zap.String("user_id", payload.UserID.String()),
		zap.String("title", payload.Title),
		zap.Any("types", payload.Types),
		zap.Time("scheduled_at", payload.ScheduledAt))

	// Create and publish notification request
	var channels []string
	if utils.ContainsSlice(payload.Types, enum.NotificationTypeAll) {
		channels = []string{enum.NotificationTypeEmail.String(), enum.NotificationTypePush.String(), enum.NotificationTypeInApp.String()}
	} else {
		channels = utils.MapSlice(payload.Types, func(nt enum.NotificationType) string { return nt.String() })
	}
	req := &requests.PublishNotificationRequest{
		UserID:   payload.UserID,
		Title:    payload.Title,
		Body:     payload.Body,
		Data:     payload.Data,
		Channels: channels,
	}

	// Handle template data if provided
	if payload.TemplateName != "" && len(payload.TemplateData) > 0 {
		req.EmailTemplateName = &payload.TemplateName
		req.EmailTemplateData = payload.TemplateData
	}
	// Handle email-specific fields
	if payload.Subject != "" {
		req.EmailSubject = &payload.Subject
	}

	// Update schedule status to 'PROCESSING', or create new schedule if no ScheduleID is provided.
	schedulesIDs := payload.ScheduleIDs
	createdScheduleMap := make(map[string]model.Schedule) // enum.ChannelType -> model.Schedule
	if err := helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if len(schedulesIDs) == 0 {
			// Create new schedules for each channel if not provided
			schedulesIDs = make([]uuid.UUID, 0, len(channels))
			for _, ch := range channels {
				id := uuid.New()
				scheduleModel := model.Schedule{
					ID:          id,
					Type:        utils.DerefPtr(payload.ScheduleType, enum.ScheduleTypeOther),
					ScheduledAt: payload.ScheduledAt,
					Status:      enum.ScheduleStatusProcessing,
					CreatedBy:   payload.UserID,
				}
				createdScheduleMap[ch] = scheduleModel
				schedulesIDs = append(schedulesIDs, id)
			}

			if len(createdScheduleMap) > 0 {
				schedulesToCreate := utils.MapSlice(utils.GetValues(createdScheduleMap), func(s model.Schedule) *model.Schedule {
					return &s
				})
				if modifiedCount, err := uow.Schedules().BulkAdd(ctx, schedulesToCreate, 100); err != nil {
					return err
				} else if modifiedCount != int64(len(schedulesToCreate)) {
					return fmt.Errorf("failed to create schedules for notification: %w", err)
				}
			}
			return nil
		} else {
			// Update existing schedules to 'PROCESSING'
			return uow.Schedules().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id IN ?", schedulesIDs).Where("deleted_at IS NULL")
			}, map[string]any{"status": enum.ScheduleStatusProcessing})
		}
	}); err != nil {
		zap.L().Error("Failed to complete transaction for scheduled notification",
			zap.String("user_id", payload.UserID.String()),
			zap.String("title", payload.Title),
			zap.Error(err))
		return fmt.Errorf("failed to process scheduled notification: %w", err)
	}
	// Add schedule IDs to notification request
	req.ScheduleIDsByChannels = make(map[enum.NotificationType]uuid.UUID, len(createdScheduleMap))
	for channel, schedule := range createdScheduleMap {
		req.ScheduleIDsByChannels[enum.NotificationType(channel)] = schedule.ID
	}

	// Create and publish the notification via RabbitMQ
	notificationIDsByChannels, err := h.notificationService.CreateAndPublishNotification(ctx, req)
	if err != nil {
		// Update schedule status to 'FAILED'
		h.updateScheduleStatus(ctx, schedulesIDs, enum.ScheduleStatusFailed, utils.PtrOrNil(err.Error()))
		zap.L().Error("Failed to create and publish scheduled notification",
			zap.Error(err),
			zap.String("user_id", payload.UserID.String()),
			zap.String("title", payload.Title))
		return fmt.Errorf("failed to publish notification: %w", err)
	} else if len(notificationIDsByChannels) == len(channels) {
		// Update ReferenceID and type in schedules to link to created notifications
		if err := h.bulkUpdateScheduleReference(ctx, schedulesIDs, notificationIDsByChannels, createdScheduleMap); err != nil {
			// Update schedule status to 'FAILED'
			h.updateScheduleStatus(ctx, schedulesIDs, enum.ScheduleStatusFailed, utils.PtrOrNil(err.Error()))
			zap.L().Error("Failed to update schedule references",
				zap.Error(err),
				zap.String("user_id", payload.UserID.String()),
				zap.String("title", payload.Title))
			return fmt.Errorf("failed to update schedule references: %w", err)
		}
	}

	zap.L().Info("Scheduled notification published successfully",
		zap.String("user_id", payload.UserID.String()),
		zap.String("title", payload.Title),
		zap.Int("notification_count", len(notificationIDsByChannels)))

	return nil
}

// updateScheduleStatus updates the status of the schedule identified by scheduleID
func (h *NotificationScheduledHandler) updateScheduleStatus(ctx context.Context, scheduleID []uuid.UUID, status enum.ScheduleStatus, errorMessage *string) error {
	return helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		return uow.Schedules().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ?", scheduleID).Where("deleted_at IS NULL")
		}, map[string]any{
			"status":     status,
			"last_error": errorMessage,
		})
	})
}

// bulkUpdateScheduleReference updates the reference ID and type for multiple schedules
func (h *NotificationScheduledHandler) bulkUpdateScheduleReference(
	ctx context.Context,
	schedulesIDs []uuid.UUID,
	notificationIDsByChannels map[enum.NotificationType]uuid.UUID,
	createdScheduleMap map[string]model.Schedule,
) error {
	return helper.WithTransaction(ctx, h.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// If the map is empty, it means we used existing IDs. We must fetch them to know which ID is which Type.
		if len(createdScheduleMap) == 0 {
			existingSchedules, _, err := uow.Schedules().GetAll(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id IN ?", schedulesIDs).Where("deleted_at IS NULL")
			}, nil, 0, 0)
			if err != nil {
				return err
			}
			for _, s := range existingSchedules {
				createdScheduleMap[s.Type.String()] = s
			}
		}

		db := uow.DB().WithContext(ctx).Model(new(model.Schedule)).Where("id IN ?", schedulesIDs).Where("deleted_at IS NULL")

		referenceIDUpdateExprArgs := make([]any, 0, len(notificationIDsByChannels)*2)
		referenceIDUpdateExpr := stringsbuilder.NewStringBuilder(5)
		referenceIDUpdateExpr.AppendLine("CASE ")

		for channel, notificationID := range notificationIDsByChannels {
			scheduleToUpdate, exists := createdScheduleMap[channel.String()]
			if !exists {
				continue
			}
			referenceIDUpdateExpr.AppendLine("WHEN ID = ? THEN ? ")
			referenceIDUpdateExprArgs = append(referenceIDUpdateExprArgs, scheduleToUpdate.ID, notificationID)
		}
		referenceIDUpdateExpr.AppendLine("ELSE reference_id END")

		// Skip update if no args generated
		if len(referenceIDUpdateExprArgs) == 0 {
			return nil
		}

		db.Updates(map[string]any{
			"reference_id":   gorm.Expr(referenceIDUpdateExpr.String(), referenceIDUpdateExprArgs...),
			"reference_type": enum.ReferenceTypeNotification,
		})

		return nil
	})
}
