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

// ContentPublishConsumer handles content publishing messages from RabbitMQ
type ContentPublishConsumer struct {
	contentPublishingService iservice.ContentPublishingService
	validator                *validator.Validate
}

// NewContentPublishConsumer creates a new content publish consumer
func NewContentPublishConsumer(
	contentPublishingService iservice.ContentPublishingService,
) *ContentPublishConsumer {
	return &ContentPublishConsumer{
		contentPublishingService: contentPublishingService,
		validator:                validator.New(),
	}
}

// Handle processes content publishing messages for a single channel
func (c *ContentPublishConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received content publish message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.PublishContentMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal content publish message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if err := c.validator.Struct(msg); err != nil {
		zap.L().Error("Invalid content publish message",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("channel_id", msg.ChannelID.String()))
		return fmt.Errorf("message validation failed: %w", err)
	}

	zap.L().Info("Processing content publish request",
		zap.String("content_id", msg.ContentID.String()),
		zap.String("channel_id", msg.ChannelID.String()),
		zap.String("user_id", msg.UserID.String()),
		zap.String("request_id", msg.RequestID))

	// Publish content to channel
	result, err := c.contentPublishingService.PublishToChannel(
		ctx,
		msg.ContentID,
		msg.ChannelID,
		msg.UserID,
	)

	if err != nil {
		zap.L().Error("Failed to publish content to channel",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("channel_id", msg.ChannelID.String()),
			zap.String("request_id", msg.RequestID))
		return fmt.Errorf("failed to publish content: %w", err)
	}

	zap.L().Info("Content published successfully",
		zap.String("content_channel_id", result.ContentChannelID.String()),
		zap.String("post_url", result.PostURL),
		zap.String("channel", result.Channel),
		zap.String("request_id", msg.RequestID))

	return nil
}
