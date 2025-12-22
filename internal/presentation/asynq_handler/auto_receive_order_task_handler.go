package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/interfaces/iservice"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// AutoReceiveOrderHandler marks an order as RECEIVED after it has been DELIVERED for some time.
type AutoReceiveOrderHandler struct {
	orderService iservice.OrderService
}

func NewAutoReceiveOrderHandler(orderService iservice.OrderService) *AutoReceiveOrderHandler {
	return &AutoReceiveOrderHandler{orderService: orderService}
}

func (h *AutoReceiveOrderHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing auto receive order task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	var payload asynqtask.AutoReceiveOrderTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal auto receive order payload",
			zap.Error(err),
			zap.ByteString("raw_payload", task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	if payload.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}

	// Use system user (uuid.Nil) for the auto action.
	if err := h.orderService.MarkAsReceived(ctx, payload.OrderID, uuid.Nil); err != nil {
		zap.L().Error("Failed to auto mark order as received",
			zap.Error(err),
			zap.String("order_id", payload.OrderID.String()))
		return err
	}

	zap.L().Info("Order auto marked as received",
		zap.String("order_id", payload.OrderID.String()))
	return nil
}
