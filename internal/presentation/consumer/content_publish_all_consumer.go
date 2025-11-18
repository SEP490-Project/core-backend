package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// ContentPublishAllConsumer handles batch content publishing messages from RabbitMQ
type ContentPublishAllConsumer struct {
	contentPublishingService iservice.ContentPublishingService
	validator                *validator.Validate
}

// NewContentPublishAllConsumer creates a new content publish all consumer
func NewContentPublishAllConsumer(
	contentPublishingService iservice.ContentPublishingService,
) *ContentPublishAllConsumer {
	return &ContentPublishAllConsumer{
		contentPublishingService: contentPublishingService,
		validator:                validator.New(),
	}
}

// Handle processes content publishing messages for all channels
func (c *ContentPublishAllConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received content publish-all message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.PublishAllChannelsMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal content publish-all message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if err := c.validator.Struct(msg); err != nil {
		zap.L().Error("Invalid content publish-all message",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()))
		return fmt.Errorf("message validation failed: %w", err)
	}

	zap.L().Info("Processing content publish-all request",
		zap.String("content_id", msg.ContentID.String()),
		zap.String("user_id", msg.UserID.String()),
		zap.String("request_id", msg.RequestID))

	// Publish content to all channels
	result, err := c.contentPublishingService.PublishToAllChannels(
		ctx,
		msg.ContentID,
		msg.UserID,
	)

	if err != nil {
		zap.L().Error("Failed to publish content to all channels",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("request_id", msg.RequestID))
		return fmt.Errorf("failed to publish content to all channels: %w", err)
	}

	zap.L().Info("Content published to all channels",
		zap.Int("total_channels", result.TotalChannels),
		zap.Int("success_count", result.SuccessCount),
		zap.Int("failure_count", result.FailureCount),
		zap.String("request_id", msg.RequestID))

	// Log any errors
	for _, channelError := range result.Errors {
		zap.L().Warn("Channel publish failed",
			zap.String("channel_id", channelError.ChannelID.String()),
			zap.String("channel_name", channelError.ChannelName),
			zap.String("error", channelError.Error))
	}

	// If all channels failed, return error to trigger retry
	if result.FailureCount == result.TotalChannels && result.TotalChannels > 0 {
		return fmt.Errorf("all channels failed to publish (%d/%d)", result.FailureCount, result.TotalChannels)
	}

	return nil
}
