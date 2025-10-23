package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContractCreateConsumer handles contract creation messages from RabbitMQ
type ContractCreateConsumer struct {
	appRegistry *application.ApplicationRegistry
	unitOfWork  irepository.UnitOfWork
}

// NewContractCreateConsumer creates a new contract create consumer
func NewContractCreateConsumer(appRegistry *application.ApplicationRegistry) *ContractCreateConsumer {
	return &ContractCreateConsumer{
		appRegistry: appRegistry,
		unitOfWork:  appRegistry.InfrastructureRegistry.UnitOfWork,
	}
}

// Handle processes contract creation messages
func (c *ContractCreateConsumer) Handle(ctx context.Context, body []byte) error {
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Recovered from panic in ContractCreateConsumer.Handle",
				zap.Any("panic", r))
		}
	}()

	zap.L().Info("Received contract creation message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.ContractCreateMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal contract creation message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	zap.L().Info("Processing contract creation",
		zap.String("user_id", msg.UserID.String()))

	success := true
	uow := c.unitOfWork.Begin(ctx)

	history, err := c.appRegistry.ModifiedHistoryService.AddWithUOW(ctx, &requests.CreateModifiedHistoryRequest{
		ReferenceType: enum.ModifiedTypeContract.String(),
		Operation:     enum.ModifiedOperationCreate.String(),
		Description:   fmt.Sprintf("User %s create a new contract", utils.ToString(msg.UserID)),
		ChangedByID:   msg.UserID.String(),
	}, uow)
	if err != nil {
		success = false
		zap.L().Error("Failed to create ModifiedHistory", zap.Error(err))
		return fmt.Errorf("failed to create ModifiedHistory: %w", err)
	}

	// Create contract using ContractService
	var contract *responses.ContractResponse
	contract, err = c.appRegistry.ContractService.CreateContract(ctx, msg.UserID, &msg.Contract, uow)
	if err != nil {
		success = false
		zap.L().Error("Failed to create contract", zap.Error(err))
		return fmt.Errorf("failed to create contract: %w", err)
	}

	updateRequest := &requests.UpdateModifiedHistoryRequest{
		ReferenceID: &contract.ID,
	}
	var status string
	if success {
		uow.Commit()
		status = enum.ModifiedStatusCompleted.String()
	} else {
		uow.Rollback()
		status = enum.ModifiedStatusFailed.String()
	}
	updateRequest.Status = &status

	ID, _ := uuid.Parse(history.ID)
	_, _ = c.appRegistry.ModifiedHistoryService.Update(ctx, ID, updateRequest)

	zap.L().Info("Contract created successfully",
		zap.String("contract_id", contract.ID),
		zap.String("contract_number", contract.ContractNumber))

	// TODO: Publish follow-up events if needed
	// Example: Publish contract-created notification event
	// producer, _ := rabbitmq.GetProducer("notification-email-producer")
	// producer.PublishJSON(ctx, NotificationMessage{...})

	return nil
}
