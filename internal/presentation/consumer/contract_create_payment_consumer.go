package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContractCreatePaymentConsumer handles contract payment creation messages from RabbitMQ
type ContractCreatePaymentConsumer struct {
	appRegistry            *application.ApplicationRegistry
	contractPaymentService iservice.ContractPaymentService
	modifiedHistoryService iservice.ModifiedHistoryService
	unitOfWork             irepository.UnitOfWork
}

// NewContractCreatePaymentConsumer creates a new contract payment consumer
func NewContractCreatePaymentConsumer(appRegistry *application.ApplicationRegistry) *ContractCreatePaymentConsumer {
	return &ContractCreatePaymentConsumer{
		appRegistry:            appRegistry,
		contractPaymentService: appRegistry.ContractPaymentService,
		modifiedHistoryService: appRegistry.ModifiedHistoryService,
		unitOfWork:             appRegistry.InfrastructureRegistry.UnitOfWork,
	}
}

// Handle processes contract payment creation messages
func (c *ContractCreatePaymentConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received contract payment creation message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.ContractCreatePaymentMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal contract payment message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	var userID uuid.UUID
	var contractID uuid.UUID
	var err error

	userID, err = uuid.Parse(msg.UserID)
	if err != nil {
		zap.L().Error("Failed to parse user ID",
			zap.String("user_id", msg.UserID),
			zap.Error(err))
		return fmt.Errorf("failed to parse user ID: %w", err)
	}
	contractID, err = uuid.Parse(msg.ContractID)
	if err != nil {
		zap.L().Error("Failed to parse contract ID",
			zap.String("contract_id", msg.ContractID),
			zap.Error(err))
		return fmt.Errorf("failed to parse contract ID: %w", err)
	}

	zap.L().Info("Processing contract payment creation",
		zap.String("user_id", msg.UserID),
		zap.String("contract_id", msg.ContractID))

	uow := c.unitOfWork.Begin()

	if err = c.contractPaymentService.CreateContractPaymentsFromContract(ctx, userID, contractID, uow); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to create contract payments",
			zap.String("user_id", msg.UserID),
			zap.String("contract_id", msg.ContractID),
			zap.Error(err))
		return fmt.Errorf("failed to create contract payments: %w", err)
	}

	uow.Commit()

	zap.L().Info("Contract payment processed successfully",
		zap.String("contract_id", msg.ContractID))

	return nil
}
