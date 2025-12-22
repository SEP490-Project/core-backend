package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// CancelPaymentHandler handles scheduled cancel payment tasks
// Task payload: asynqtask.CancelPaymentTaskPayload
// The payload carries PaymentTransaction ID; the handler loads the transaction and cancels its PayOS link.
type CancelPaymentHandler struct {
	paymentService iservice.PaymentTransactionService
	uowFactory     irepository.UnitOfWork
}

func NewCancelPaymentHandler(
	paymentService iservice.PaymentTransactionService,
	uowFactory irepository.UnitOfWork,
) *CancelPaymentHandler {
	return &CancelPaymentHandler{paymentService: paymentService, uowFactory: uowFactory}
}

func (h *CancelPaymentHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	zap.L().Info("Processing cancel payment task",
		zap.String("task_id", task.ResultWriter().TaskID()),
		zap.Int("payload_size", len(task.Payload())))

	var payload asynqtask.CancelPaymentTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal cancel payment payload",
			zap.Error(err),
			zap.ByteString("raw_payload", task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.PaymentID == "" {
		return errors.New("payment_id is required")
	}

	// Start a unit-of-work.
	uow := h.uowFactory.Begin(ctx)
	defer func() { _ = uow.Rollback() }()

	paymentUUID, err := uuid.Parse(payload.PaymentID)
	if err != nil {
		zap.L().Error("Invalid payment_id in cancel payment payload", zap.Error(err), zap.String("payment_id", payload.PaymentID))
		return fmt.Errorf("invalid payment_id: %w", err)
	}

	transaction, err := uow.PaymentTransaction().GetByID(ctx, paymentUUID, nil)
	if err != nil {
		zap.L().Error("Failed to load payment transaction for cancel", zap.Error(err), zap.String("payment_id", payload.PaymentID))
		return fmt.Errorf("failed to load payment transaction: %w", err)
	}
	if transaction == nil || transaction.PayOSMetadata == nil {
		zap.L().Warn("Payment transaction/payOS metadata not found; skipping cancel", zap.String("payment_id", payload.PaymentID))
		return nil
	}

	orderCode := strconv.FormatInt(transaction.PayOSMetadata.OrderCode, 10)

	if err := h.paymentService.CancelPaymentLink(ctx, uow, orderCode, "Payment link expired"); err != nil {
		zap.L().Error("Failed to cancel payment link from scheduled task",
			zap.Error(err),
			zap.String("payment_id", payload.PaymentID),
			zap.String("order_code", orderCode))
		return err // trigger retry
	}

	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit cancel payment task UoW",
			zap.Error(err),
			zap.String("payment_id", payload.PaymentID),
			zap.String("order_code", orderCode))
		return err
	}

	zap.L().Info("Cancel payment task processed successfully",
		zap.String("payment_id", payload.PaymentID),
		zap.String("order_code", orderCode))
	return nil
}
