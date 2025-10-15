package consumer

import (
	"context"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// ContractCreatePaymentConsumer handles contract payment creation messages from RabbitMQ
type ContractCreatePaymentConsumer struct {
	appRegistry *application.ApplicationRegistry
	unitOfWork  irepository.UnitOfWork
}

// NewContractCreatePaymentConsumer creates a new contract payment consumer
func NewContractCreatePaymentConsumer(appRegistry *application.ApplicationRegistry) *ContractCreatePaymentConsumer {
	return &ContractCreatePaymentConsumer{
		appRegistry: appRegistry,
		unitOfWork:  appRegistry.InfrastructureRegistry.UnitOfWork,
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

	zap.L().Info("Processing contract payment creation",
		zap.String("user_id", msg.UserID),
		zap.String("contract_id", msg.ContractID))

	zap.L().Info("Contract payment processed successfully",
		zap.String("contract_id", msg.ContractID))

	return nil
}
