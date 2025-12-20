package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// EnqueueTask enqueues a task for immediate processing
func (ac *AsynqClient) EnqueueTask(ctx context.Context, taskType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)
	info, err := ac.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		zap.L().Error("Failed to enqueue task",
			zap.String("task_type", taskType),
			zap.Error(err))
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	zap.L().Info("Task enqueued",
		zap.String("task_id", info.ID),
		zap.String("task_type", taskType),
		zap.String("queue", info.Queue))

	return info, nil
}

// ScheduleTask schedules a task to be processed at a specific time
// If the scheduled time is in the past, the task will be processed immediately
func (ac *AsynqClient) ScheduleTask(ctx context.Context, taskType string, payload any, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	// If scheduled time is in the past, process immediately
	if processAt.Before(time.Now()) {
		zap.L().Warn("Scheduled time is in the past, processing immediately",
			zap.String("task_type", taskType),
			zap.Time("scheduled_at", processAt))
		processAt = time.Now()
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)

	// Add ProcessAt option
	allOpts := append([]asynq.Option{asynq.ProcessAt(processAt)}, opts...)

	info, err := ac.client.EnqueueContext(ctx, task, allOpts...)
	if err != nil {
		zap.L().Error("Failed to schedule task",
			zap.String("task_type", taskType),
			zap.Time("process_at", processAt),
			zap.Error(err))
		return nil, fmt.Errorf("failed to schedule task: %w", err)
	}

	zap.L().Info("Task scheduled",
		zap.String("task_id", info.ID),
		zap.String("task_type", taskType),
		zap.Time("process_at", processAt),
		zap.String("queue", info.Queue))

	return info, nil
}

// ScheduleTaskWithDelay schedules a task to be processed after a delay
func (ac *AsynqClient) ScheduleTaskWithDelay(ctx context.Context, taskType string, payload any, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)

	// Add ProcessIn option for delay
	allOpts := append([]asynq.Option{asynq.ProcessIn(delay)}, opts...)

	info, err := ac.client.EnqueueContext(ctx, task, allOpts...)
	if err != nil {
		zap.L().Error("Failed to schedule delayed task",
			zap.String("task_type", taskType),
			zap.Duration("delay", delay),
			zap.Error(err))
		return nil, fmt.Errorf("failed to schedule delayed task: %w", err)
	}

	zap.L().Info("Task scheduled with delay",
		zap.String("task_id", info.ID),
		zap.String("task_type", taskType),
		zap.Duration("delay", delay),
		zap.String("queue", info.Queue))

	return info, nil
}

// ScheduleTaskWithUniqueKey schedules a task with a unique key to prevent duplicates
func (ac *AsynqClient) ScheduleTaskWithUniqueKey(ctx context.Context, taskType string, payload any, processAt time.Time, uniqueKey string, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	// If scheduled time is in the past, process immediately
	if processAt.Before(time.Now()) {
		zap.L().Warn("Scheduled time is in the past, processing immediately",
			zap.String("task_type", taskType),
			zap.Time("scheduled_at", processAt))
		processAt = time.Now()
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)

	// Calculate TTL for uniqueness based on processing time
	ttl := time.Until(processAt) + 24*time.Hour // Keep unique for 24 hours after scheduled time

	// Add ProcessAt and TaskID options
	allOpts := append([]asynq.Option{
		asynq.ProcessAt(processAt),
		asynq.TaskID(uniqueKey),
		asynq.Unique(ttl),
	}, opts...)

	info, err := ac.client.EnqueueContext(ctx, task, allOpts...)
	if err != nil {
		zap.L().Error("Failed to schedule unique task",
			zap.String("task_type", taskType),
			zap.String("unique_key", uniqueKey),
			zap.Time("process_at", processAt),
			zap.Error(err))
		return nil, fmt.Errorf("failed to schedule unique task: %w", err)
	}

	zap.L().Info("Unique task scheduled",
		zap.String("task_id", info.ID),
		zap.String("task_type", taskType),
		zap.String("unique_key", uniqueKey),
		zap.Time("process_at", processAt),
		zap.String("queue", info.Queue))

	return info, nil
}

// CancelTask cancels a scheduled or pending task by its ID
func (ac *AsynqClient) CancelTask(taskID string) error {
	err := ac.inspector.DeleteTask("default", taskID)
	if err != nil {
		// Try other queues
		if err2 := ac.inspector.DeleteTask("critical", taskID); err2 == nil {
			return nil
		}
		if err2 := ac.inspector.DeleteTask("low", taskID); err2 == nil {
			return nil
		}
		zap.L().Error("Failed to cancel task", zap.String("task_id", taskID), zap.Error(err))
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	zap.L().Info("Task cancelled", zap.String("task_id", taskID))
	return nil
}

// RescheduleTask reschedules a task to a new time
// If the new time is in the past, the task will be processed immediately
func (ac *AsynqClient) RescheduleTask(ctx context.Context, taskID string, taskType string, payload any, newProcessAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	// First, try to delete the existing task (ignore error if not found)
	_ = ac.CancelTask(taskID)

	// Create new unique key
	newTaskID := fmt.Sprintf("%s-%s", taskID, uuid.New().String()[:8])

	// Schedule the new task
	return ac.ScheduleTaskWithUniqueKey(ctx, taskType, payload, newProcessAt, newTaskID, opts...)
}
