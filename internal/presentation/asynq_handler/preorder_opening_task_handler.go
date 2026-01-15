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

// PreOrderOpeningHandler handles scheduled pre-order opening tasks.
// When the AvailabilityStartDate of a limited product is reached,
// this handler transitions the pre-order to the appropriate state
// (AwaitingPickup for self-pickup or InTransit for delivery).
type PreOrderOpeningHandler struct {
	preOrderService      iservice.PreOrderService
	stateTransferService iservice.StateTransferService
	uowFactory           irepository.UnitOfWork
}

func NewPreOrderOpeningHandler(
	preOrderService iservice.PreOrderService,
	stateTransferService iservice.StateTransferService,
	uowFactory irepository.UnitOfWork,
) *PreOrderOpeningHandler {
	return &PreOrderOpeningHandler{
		preOrderService:      preOrderService,
		stateTransferService: stateTransferService,
		uowFactory:           uowFactory,
	}
}

func (h *PreOrderOpeningHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing pre-order opening task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	var payload asynqtask.PreOrderOpeningTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal pre-order opening payload",
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

	// Load the pre-order with necessary relationships
	preOrder, err := uow.PreOrder().GetByID(ctx, payload.PreOrderID, []string{
		"ProductVariant",
		"ProductVariant.Product",
		"ProductVariant.Product.Limited",
	})
	if err != nil {
		zap.L().Error("Failed to load pre-order for opening task",
			zap.Error(err),
			zap.String("preorder_id", payload.PreOrderID.String()))
		return fmt.Errorf("failed to load pre-order: %w", err)
	}
	if preOrder == nil {
		zap.L().Warn("Pre-order not found; skipping opening task",
			zap.String("preorder_id", payload.PreOrderID.String()))
		return nil // Task completed, nothing to do
	}

	// Skip if pre-order is not in PRE_ORDERED status
	if preOrder.Status != enum.PreOrderStatusPreOrdered {
		zap.L().Info("Pre-order is not in PRE_ORDERED status; skipping",
			zap.String("preorder_id", payload.PreOrderID.String()),
			zap.String("current_status", string(preOrder.Status)))
		return nil
	}

	// Determine target state based on delivery method
	var targetState enum.PreOrderStatus
	if preOrder.IsSelfPickedUp {
		targetState = enum.PreOrderStatusAwaitingPickup
	} else {
		targetState = enum.PreOrderStatusInTransit
	}

	// Use system user (uuid.Nil) for the automated transition
	if err := h.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, targetState, uuid.UUID{}, nil, nil); err != nil {
		zap.L().Error("Failed to transition pre-order to opening state",
			zap.Error(err),
			zap.String("preorder_id", payload.PreOrderID.String()),
			zap.String("target_state", string(targetState)))
		return err // Trigger retry
	}

	// Update the associated schedule status to completed
	scheduleFilter := func(db interface{}) interface{} {
		return db
	}
	_ = scheduleFilter // TODO: Update schedule status if needed

	zap.L().Info("Pre-order opening task processed successfully",
		zap.String("preorder_id", payload.PreOrderID.String()),
		zap.String("new_status", string(targetState)),
		zap.Bool("is_self_pickup", preOrder.IsSelfPickedUp))

	return nil
}
