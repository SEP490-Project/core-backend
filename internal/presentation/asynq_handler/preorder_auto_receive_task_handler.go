package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// PreOrderAutoReceiveHandler handles scheduled pre-order auto-receive tasks.
// When a pre-order has been in DELIVERED status for the configured number of days
// (default 30 days), this handler automatically transitions it to RECEIVED status.
type PreOrderAutoReceiveHandler struct {
	preOrderService      iservice.PreOrderService
	stateTransferService iservice.StateTransferService
	uowFactory           irepository.UnitOfWork
}

func NewPreOrderAutoReceiveHandler(
	preOrderService iservice.PreOrderService,
	stateTransferService iservice.StateTransferService,
	uowFactory irepository.UnitOfWork,
) *PreOrderAutoReceiveHandler {
	return &PreOrderAutoReceiveHandler{
		preOrderService:      preOrderService,
		stateTransferService: stateTransferService,
		uowFactory:           uowFactory,
	}
}

func (h *PreOrderAutoReceiveHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing pre-order auto-receive task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	var payload asynqtask.PreOrderAutoReceiveTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal pre-order auto-receive payload",
			zap.Error(err),
			zap.ByteString("raw_payload", task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.PreOrderID == uuid.Nil {
		return errors.New("preorder_id is required")
	}

	// Start a unit-of-work
	uow := h.uowFactory.Begin(ctx)
	defer func() { _ = uow.Rollback() }()

	// Load the pre-order to check current status
	preOrder, err := uow.PreOrder().GetByID(ctx, payload.PreOrderID, nil)
	if err != nil {
		zap.L().Error("Failed to load pre-order for auto-receive task",
			zap.Error(err),
			zap.String("preorder_id", payload.PreOrderID.String()))
		return fmt.Errorf("failed to load pre-order: %w", err)
	}
	if preOrder == nil {
		zap.L().Warn("Pre-order not found; skipping auto-receive task",
			zap.String("preorder_id", payload.PreOrderID.String()))
		return nil // Task completed, nothing to do
	}

	// Only auto-receive if pre-order is still in DELIVERED status
	// Skip if customer has already marked as received, requested compensation, etc.
	if preOrder.Status != enum.PreOrderStatusDelivered {
		zap.L().Info("Pre-order is not in DELIVERED status; skipping auto-receive",
			zap.String("preorder_id", payload.PreOrderID.String()),
			zap.String("current_status", string(preOrder.Status)))
		return nil
	}

	// Use system user (uuid.Nil) for the automated transition
	// No file URL required for auto-receive (proof of delivery was provided when marking as delivered)
	if err := h.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusReceived, uuid.UUID{}, nil, nil); err != nil {
		zap.L().Error("Failed to auto-mark pre-order as received",
			zap.Error(err),
			zap.String("preorder_id", payload.PreOrderID.String()))
		return err // Trigger retry
	}

	zap.L().Info("Pre-order auto-marked as received successfully",
		zap.String("preorder_id", payload.PreOrderID.String()),
		zap.String("previous_status", string(enum.PreOrderStatusDelivered)),
		zap.String("new_status", string(enum.PreOrderStatusReceived)))

	return nil
}
