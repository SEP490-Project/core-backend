package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// ContentScheduleHandler handles scheduled content publishing tasks
type ContentScheduleHandler struct {
	scheduleService iservice.ContentScheduleService
	alertManager    iservice.AlertManagerService
}

// NewContentScheduleHandler creates a new content schedule handler

func NewContentScheduleHandler(
	scheduleService iservice.ContentScheduleService,
	alertManager iservice.AlertManagerService,
) *ContentScheduleHandler {
	return &ContentScheduleHandler{
		scheduleService: scheduleService,
		alertManager:    alertManager,
	}
}

// ProcessTask processes scheduled content publishing tasks
func (h *ContentScheduleHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing content schedule task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	// Parse payload
	var payload asynqtask.ContentScheduleTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal content schedule payload",
			zap.Error(err),
			zap.ByteString("raw_payload", task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing scheduled content",
		zap.String("schedule_id", payload.ScheduleID.String()),
		zap.String("content_id", payload.ContentID.String()),
		zap.String("content_channel_id", payload.ContentChannelID.String()),
		zap.String("channel_code", payload.ChannelCode),
		zap.Time("scheduled_at", payload.ScheduledAt),
		zap.Int("retry_count", payload.RetryCount))

	// Execute the scheduled publish
	err := h.scheduleService.ExecuteScheduledPublish(ctx, payload.ScheduleID)
	if err != nil {
		zap.L().Error("Failed to execute scheduled publish",
			zap.Error(err),
			zap.String("schedule_id", payload.ScheduleID.String()),
			zap.Int("retry_count", payload.RetryCount))

		// Raise an alert for failed schedule
		h.raiseScheduleFailedAlert(ctx, payload, err.Error())

		// Return error to trigger Asynq retry
		return fmt.Errorf("failed to execute scheduled publish: %w", err)
	}

	zap.L().Info("Scheduled content published successfully",
		zap.String("schedule_id", payload.ScheduleID.String()),
		zap.String("content_id", payload.ContentID.String()),
		zap.String("channel_code", payload.ChannelCode))

	return nil
}

// raiseScheduleFailedAlert raises an alert when a scheduled publish fails
func (h *ContentScheduleHandler) raiseScheduleFailedAlert(ctx context.Context, payload asynqtask.ContentScheduleTaskPayload, errorMessage string) {
	// Get schedule details for the alert
	schedule, err := h.scheduleService.GetScheduleByID(ctx, payload.ScheduleID)
	if err != nil {
		zap.L().Warn("Failed to get schedule details for alert",
			zap.Error(err),
			zap.String("schedule_id", payload.ScheduleID.String()))
		return
	}

	contentTitle := "Unknown Content"
	if schedule != nil {
		contentTitle = schedule.ContentTitle
	}

	// Raise alert via AlertManager
	if h.alertManager != nil {
		if err := h.alertManager.RaiseScheduleFailedAlert(ctx, payload.ScheduleID, contentTitle, errorMessage); err != nil {
			zap.L().Error("Failed to raise schedule failed alert",
				zap.Error(err),
				zap.String("schedule_id", payload.ScheduleID.String()))
		}
	}
}

// HandleDLQ handles dead-letter queue tasks for schedules that failed all retries
func (h *ContentScheduleHandler) HandleDLQ(ctx context.Context, task *asynq.Task) error {
	zap.L().Warn("Received DLQ task for scheduled content",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	// Parse payload
	var payload asynqtask.ContentScheduleTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal DLQ schedule payload",
			zap.Error(err))
		return nil // Don't retry DLQ parsing failures
	}

	zap.L().Error("Scheduled content permanently failed",
		zap.String("schedule_id", payload.ScheduleID.String()),
		zap.String("content_id", payload.ContentID.String()),
		zap.Int("retry_count", payload.RetryCount))

	// Update schedule status to FAILED
	if err := h.scheduleService.UpdateScheduleStatus(ctx, payload.ScheduleID, enum.ScheduleStatusFailed); err != nil {
		zap.L().Error("Failed to update schedule status to FAILED",
			zap.Error(err),
			zap.String("schedule_id", payload.ScheduleID.String()))
	}

	// Raise critical alert for permanent failure
	h.raiseScheduleFailedAlert(ctx, payload, "Scheduled publish permanently failed after all retries")

	return nil // Acknowledge DLQ task
}
