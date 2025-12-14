package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// ContentPublishConsumer handles content publishing messages from RabbitMQ
type ContentPublishConsumer struct {
	contentPublishingService iservice.ContentPublishingService
	alertService             iservice.AlertManagerService
	notificationService      iservice.NotificationService
	validator                *validator.Validate
}

// NewContentPublishConsumer creates a new content publish consumer
func NewContentPublishConsumer(appReg *application.ApplicationRegistry) *ContentPublishConsumer {
	return &ContentPublishConsumer{
		contentPublishingService: appReg.ContentPublishingService,
		alertService:             appReg.AlertManagerService,
		notificationService:      appReg.NotificationService,
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
		// Alert failure
		c.alertFailure(ctx, msg, err)
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
		// Alert failure
		c.alertFailure(ctx, msg, err)
		return fmt.Errorf("failed to publish content: %w", err)
	}

	zap.L().Info("Content published successfully",
		zap.String("content_channel_id", result.ContentChannelID.String()),
		zap.String("post_url", result.PostURL),
		zap.String("channel", result.Channel),
		zap.String("request_id", msg.RequestID))

	return nil
}

func (c *ContentPublishConsumer) alertFailure(ctx context.Context, msg consumers.PublishContentMessage, err error) {
	// Raise System Alert
	alertRequest := requests.RaiseAlertRequest{
		Type:           enum.AlertTypeError,
		Category:       enum.AlertCategoryContentRejected,
		Severity:       enum.AlertSeverityHigh,
		Title:          "Content Publishing Failed",
		Description:    fmt.Sprintf("Content publishing failed. Reason: %s", err.Error()),
		ReferenceID:    &msg.ContentID,
		ReferenceType:  utils.PtrOrNil(enum.ReferenceTypeContent),
		ActionURL:      nil,
		ExpiresInHours: nil,
		TargetRoles:    []enum.UserRole{enum.UserRoleContentStaff, enum.UserRoleAdmin},
	}
	if _, subErr := c.alertService.RaiseAlert(ctx, &alertRequest); subErr != nil {
		zap.L().Error("Failed to raise alert for content publishing failure",
			zap.Error(subErr),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("channel_id", msg.ChannelID.String()),
			zap.String("request_id", msg.RequestID))
		return // Continue to next message
	}

	// Notify related users
	notiReq := requests.PublishInAppRequest{
		UserID: msg.UserID,
		Title:  "Content Publishing Failed",
		Body: fmt.Sprintf("Your content (ID: %s) failed to publish to channel (ID: %s). Reason: %s",
			msg.ContentID.String(), msg.ChannelID.String(), err.Error()),
		Data: map[string]string{
			"content_id": msg.ContentID.String(),
			"channel_id": msg.ChannelID.String(),
			"error":      err.Error(),
		},
	}
	if _, subErr := c.notificationService.CreateAndPublishInApp(ctx, &notiReq); subErr != nil {
		zap.L().Error("Failed to send in-app notification for content publishing failure",
			zap.Error(subErr),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("channel_id", msg.ChannelID.String()),
			zap.String("request_id", msg.RequestID))
		return // Continue to next message
	}
}
