package consumer

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationInAppConsumer handles in-app notification messages from RabbitMQ
type NotificationInAppConsumer struct {
	sseService             iservice.SSEService
	notificationRepository irepository.NotificationRepository
	scheduleRepository     irepository.ScheduleRepository
	userService            iservice.UserService
	unitOfWork             irepository.UnitOfWork
}

// NewNotificationInAppConsumer creates a new in-app notification consumer
func NewNotificationInAppConsumer(
	sseService iservice.SSEService,
	dbRegistry *gormrepository.DatabaseRegistry,
	userService iservice.UserService,
	unitOfWork irepository.UnitOfWork,
) *NotificationInAppConsumer {
	return &NotificationInAppConsumer{
		sseService:             sseService,
		notificationRepository: dbRegistry.NotificationRepository,
		scheduleRepository:     dbRegistry.ScheduleRepository,
		userService:            userService,
		unitOfWork:             unitOfWork,
	}
}

// Handle processes in-app notification messages
func (c *NotificationInAppConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received in-app notification message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.InAppNotificationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		// Try to unmarshal as UnifiedNotificationMessage
		var unifiedMsg consumers.UnifiedNotificationMessage
		if errUnified := json.Unmarshal(body, &unifiedMsg); errUnified == nil && unifiedMsg.UserID != uuid.Nil {
			// Convert Unified to InApp message
			msg = *unifiedMsg.ToInAppRequest()
		} else {

			zap.L().Error("Failed to unmarshal in-app notification message",
				zap.Error(err),
				zap.String("body", string(body)))
			return err // Parsing errors should not retry
		}
	}

	zap.L().Info("Processing in-app notification",
		zap.String("notification_id", msg.NotificationID.String()),
		zap.String("user_id", msg.UserID.String()),
		zap.String("title", msg.Title))

	severity := msg.Severity.String()
	if severity == "" {
		severity = enum.NotificationSeverityInfo.String()
	}

	// Send real-time event via SSE
	// We send the full message data so the frontend can display a toast/popup
	eventData := map[string]any{
		"id":         msg.NotificationID,
		"user_id":    msg.UserID,
		"title":      msg.Title,
		"body":       msg.Message,
		"severity":   severity,
		"type":       enum.NotificationTypeInApp,
		"data":       msg.Data,
		"created_at": msg.CreatedAt,
	}

	if err := c.sseService.SendEvent(msg.UserID, "notification", eventData); err != nil {
		zap.L().Warn("Failed to send SSE event",
			zap.String("user_id", msg.UserID.String()),
			zap.Error(err))
		// We don't return error here because the notification is still "delivered" to the DB
		// and the user will see it when they refresh or check notifications list.
		// SSE is best-effort.
	}

	// Also send updated unread count
	count, err := c.notificationRepository.CountUnread(ctx, msg.UserID, []enum.NotificationType{enum.NotificationTypeInApp})
	if err == nil {
		_ = c.sseService.SendUnreadCount(msg.UserID, count)
	}

	// Update notification status to SENT
	// Note: For in-app, "SENT" might mean "Available in DB".
	// If the notification was created as PENDING, we mark it as SENT now.
	attempt := model.DeliveryAttempt{
		Timestamp: time.Now(),
		Status:    string(enum.NotificationStatusSent),
		Error:     "",
	}

	if err = helper.WithTransaction(ctx, c.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.Notifications().UpdateDeliveryAttempt(ctx, msg.NotificationID, attempt); err != nil {
			zap.L().Error("Failed to update delivery attempt",
				zap.String("notification_id", msg.NotificationID.String()),
				zap.Error(err))
			return err
		}

		if err = uow.Notifications().UpdateStatus(ctx, msg.NotificationID, enum.NotificationStatusSent); err != nil {
			zap.L().Error("Failed to update notification status",
				zap.String("notification_id", msg.NotificationID.String()),
				zap.Error(err))
			return err
		}
		if msg.ScheduleID != nil {
			if err = uow.Schedules().UpdateScheduleStatus(ctx, *msg.ScheduleID, enum.ScheduleStatusCompleted, nil); err != nil {
				zap.L().Error("Failed to update schedule status to COMPLETED",
					zap.String("notification_id", msg.NotificationID.String()),
					zap.String("schedule_id", msg.ScheduleID.String()),
					zap.Error(err))
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
