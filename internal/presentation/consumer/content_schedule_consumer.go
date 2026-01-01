package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// ContentScheduleConsumer handles scheduled content publishing messages from RabbitMQ
// These messages are delivered via x-delayed-message plugin at the scheduled time
type ContentScheduleConsumer struct {
	scheduleService iservice.ContentScheduleService
	alertManager    iservice.AlertManagerService
	validator       *validator.Validate
}

// NewContentScheduleConsumer creates a new content schedule consumer
func NewContentScheduleConsumer(
	scheduleService iservice.ContentScheduleService,
	alertManager iservice.AlertManagerService,
) *ContentScheduleConsumer {
	return &ContentScheduleConsumer{
		scheduleService: scheduleService,
		alertManager:    alertManager,
		validator:       validator.New(),
	}
}

// Handle processes scheduled content publishing messages
// This is called when the delayed message is delivered at the scheduled time
func (c *ContentScheduleConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received scheduled content message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.ContentScheduleMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal schedule message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if err := c.validator.Struct(msg); err != nil {
		zap.L().Error("Invalid schedule message",
			zap.Error(err),
			zap.String("schedule_id", msg.ScheduleID.String()))
		return fmt.Errorf("message validation failed: %w", err)
	}

	zap.L().Info("Processing scheduled content",
		zap.String("schedule_id", msg.ScheduleID.String()),
		zap.String("content_id", msg.ContentID.String()),
		zap.String("content_channel_id", msg.ContentChannelID.String()),
		zap.String("channel_code", msg.ChannelCode),
		zap.Time("scheduled_at", msg.ScheduledAt),
		zap.Int("retry_count", msg.RetryCount))

	// Execute the scheduled publish
	err := c.scheduleService.ExecuteScheduledPublish(ctx, msg.ScheduleID)
	if err != nil {
		zap.L().Error("Failed to execute scheduled publish",
			zap.Error(err),
			zap.String("schedule_id", msg.ScheduleID.String()),
			zap.Int("retry_count", msg.RetryCount))

		// Raise an alert for failed schedule
		c.raiseScheduleFailedAlert(ctx, msg, err.Error())

		// Return error to trigger retry (if configured in RabbitMQ)
		return fmt.Errorf("failed to execute scheduled publish: %w", err)
	}

	zap.L().Info("Scheduled content published successfully",
		zap.String("schedule_id", msg.ScheduleID.String()),
		zap.String("content_id", msg.ContentID.String()),
		zap.String("channel_code", msg.ChannelCode))

	return nil
}

// raiseScheduleFailedAlert raises an alert when a scheduled publish fails
func (c *ContentScheduleConsumer) raiseScheduleFailedAlert(ctx context.Context, msg consumers.ContentScheduleMessage, errorMessage string) {
	// Get schedule details for the alert
	schedule, err := c.scheduleService.GetScheduleByID(ctx, msg.ScheduleID)
	if err != nil {
		zap.L().Warn("Failed to get schedule details for alert",
			zap.Error(err),
			zap.String("schedule_id", msg.ScheduleID.String()))
		return
	}

	contentTitle := "Unknown Content"
	if schedule != nil && schedule.ContentDetails != nil {
		contentTitle = schedule.ContentDetails.ContentTitle
	}

	// Raise alert via AlertManager
	if c.alertManager != nil {
		if err := c.alertManager.RaiseScheduleFailedAlert(ctx, msg.ScheduleID, contentTitle, errorMessage); err != nil {
			zap.L().Error("Failed to raise schedule failed alert",
				zap.Error(err),
				zap.String("schedule_id", msg.ScheduleID.String()))
		}
	}
}

// HandleDLQ handles dead-letter queue messages for schedules that failed all retries
func (c *ContentScheduleConsumer) HandleDLQ(ctx context.Context, body []byte) error {
	zap.L().Warn("Received DLQ message for scheduled content",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.ContentScheduleMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal DLQ schedule message",
			zap.Error(err))
		return nil // Don't retry DLQ parsing failures
	}

	zap.L().Error("Scheduled content permanently failed",
		zap.String("schedule_id", msg.ScheduleID.String()),
		zap.String("content_id", msg.ContentID.String()),
		zap.Int("retry_count", msg.RetryCount))

	// Update schedule status to FAILED
	if err := c.scheduleService.UpdateScheduleStatus(ctx, msg.ScheduleID, enum.ScheduleStatusFailed); err != nil {
		zap.L().Error("Failed to update schedule status to FAILED",
			zap.Error(err),
			zap.String("schedule_id", msg.ScheduleID.String()))
	}

	// Raise critical alert for permanent failure
	c.raiseScheduleFailedAlert(ctx, msg, "Scheduled publish permanently failed after all retries")

	return nil // Acknowledge DLQ message
}
